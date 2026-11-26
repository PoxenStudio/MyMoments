mod server_config;

use std::sync::{Arc, Mutex};

use tauri::{Manager, Url, WebviewUrl, WebviewWindowBuilder};

pub const MAIN_WINDOW: &str = "main";

/// 本地引导页：相对路径用于窗口初始加载，绝对地址用于从服务器页返回。
const GUIDE_PAGE: &str = "index.html";
const GUIDE_URL: &str = "tauri://localhost/index.html";

/// 允许在 webview 内导航的来源集合：开发服务器与已配置服务器，跨回调共享。
#[derive(Clone, Default)]
pub struct SharedOrigin(Arc<Mutex<Vec<String>>>);

fn is_app_scheme(url: &Url) -> bool {
    matches!(url.scheme(), "tauri" | "asset" | "about" | "blob" | "data")
}

fn origin_of(url: &Url) -> String {
    url.origin().ascii_serialization()
}

fn allow_origin(state: &SharedOrigin, url: &Url) {
    let origin = origin_of(url);
    if let Ok(mut list) = state.0.lock() {
        if !list.iter().any(|item| item == &origin) {
            list.push(origin);
        }
    }
}

fn reset_origins(state: &SharedOrigin, keep: Option<&Url>) {
    if let Ok(mut list) = state.0.lock() {
        list.clear();
        if let Some(url) = keep {
            list.push(origin_of(url));
        }
    }
}

fn is_allowed(url: &Url, state: &SharedOrigin) -> bool {
    if is_app_scheme(url) {
        return true;
    }
    let origin = origin_of(url);
    state
        .0
        .lock()
        .map(|list| list.iter().any(|item| item == &origin))
        .unwrap_or(false)
}

/// 只放行应用内部页面与已配置服务器同源的导航，其余交给系统浏览器。
fn allow_navigation(url: &Url, state: &SharedOrigin) -> bool {
    if is_allowed(url, state) {
        return true;
    }
    if matches!(url.scheme(), "http" | "https" | "mailto" | "tel" | "sms") {
        let _ = tauri_plugin_opener::open_url(url.as_str(), None::<&str>);
    }
    false
}

/// 开发模式下引导页由 tauri-cli 的本地服务器提供，需与服务器地址一并放行。
fn dev_url(app: &tauri::AppHandle) -> Option<Url> {
    app.config().build.dev_url.clone()
}

/// 用 /readyz 探测服务器是否可用；超时或非 2xx 视为不可用。
fn server_ready(base: &Url) -> bool {
    let mut target = base.clone();
    target.set_path("/readyz");
    target.set_query(None);
    target.set_fragment(None);
    let client = match reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(3))
        .build()
    {
        Ok(client) => client,
        Err(_) => return false,
    };
    client
        .get(target)
        .send()
        .map(|resp| resp.status().is_success())
        .unwrap_or(false)
}

#[tauri::command]
fn save_server_url(
    app: tauri::AppHandle,
    window: tauri::WebviewWindow,
    state: tauri::State<'_, SharedOrigin>,
    url: String,
) -> Result<(), String> {
    let target = server_config::normalize(&url)?;
    server_config::save(&app, target.as_str())?;
    allow_origin(state.inner(), &target);
    window.navigate(target).map_err(|err| err.to_string())
}

/// 返回本机保存过的服务器地址，最近使用的在前。
#[tauri::command]
fn recent_servers(app: tauri::AppHandle) -> Vec<String> {
    server_config::load_history(&app)
}

/// 桌面窗口全屏由前端通过 window 插件命令控制（见 capabilities）。
fn create_window(
    app: &tauri::AppHandle,
    target: WebviewUrl,
    state: &SharedOrigin,
) -> tauri::Result<()> {
    let navigation_state = state.clone();
    WebviewWindowBuilder::new(app, MAIN_WINDOW, target)
        .title("家时光")
        .inner_size(1280.0, 800.0)
        .min_inner_size(480.0, 360.0)
        .on_navigation(move |url| allow_navigation(url, &navigation_state))
        .build()?;
    Ok(())
}

#[cfg(desktop)]
fn install_menu(app: &tauri::AppHandle) -> tauri::Result<()> {
    use tauri::menu::{MenuBuilder, MenuItemBuilder, PredefinedMenuItem, SubmenuBuilder};

    let change = MenuItemBuilder::with_id("change-server", "更改服务器地址…").build(app)?;
    let quit = PredefinedMenuItem::quit(app, Some("退出家时光"))?;
    let app_menu = SubmenuBuilder::new(app, "家时光")
        .item(&change)
        .separator()
        .item(&quit)
        .build()?;
    // macOS 的复制/粘贴快捷键由菜单栏转发，缺少编辑菜单会导致输入框无法粘贴。
    let edit = SubmenuBuilder::new(app, "编辑")
        .undo()
        .redo()
        .separator()
        .cut()
        .copy()
        .paste()
        .select_all()
        .build()?;
    let menu = MenuBuilder::new(app).item(&app_menu).item(&edit).build()?;
    app.set_menu(menu)?;
    Ok(())
}

fn handle_menu_event(app: &tauri::AppHandle, id: &str) {
    if id != "change-server" {
        return;
    }
    let dev = dev_url(app);
    if let Some(state) = app.try_state::<SharedOrigin>() {
        reset_origins(state.inner(), dev.as_ref());
    }
    let guide = dev.or_else(|| Url::parse(GUIDE_URL).ok());
    if let (Some(window), Some(guide)) = (app.get_webview_window(MAIN_WINDOW), guide) {
        let _ = window.navigate(guide);
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let state = SharedOrigin::default();

    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .manage(state)
        .invoke_handler(tauri::generate_handler![save_server_url, recent_servers])
        .on_menu_event(|app, event| handle_menu_event(app, &event.id().0))
        .on_window_event(|window, event| {
            // macOS：关闭窗口即退出应用，而不是仅隐藏窗口常驻 Dock。
            #[cfg(desktop)]
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                window.app_handle().exit(0);
            }
            #[cfg(not(desktop))]
            let _ = (window, event);
        })
        .setup(|app| {
            let handle = app.handle().clone();
            let state = app.state::<SharedOrigin>().inner().clone();

            if let Some(dev) = dev_url(&handle) {
                allow_origin(&state, &dev);
            }

            let target = match server_config::load(&handle) {
                Some(saved) => match server_config::normalize(&saved) {
                    // 服务器不可达时回到地址输入页，避免白屏。
                    Ok(url) if server_ready(&url) => {
                        allow_origin(&state, &url);
                        WebviewUrl::External(url)
                    }
                    _ => WebviewUrl::App(GUIDE_PAGE.into()),
                },
                None => WebviewUrl::App(GUIDE_PAGE.into()),
            };

            create_window(&handle, target, &state)?;

            #[cfg(desktop)]
            install_menu(&handle)?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running 家时光");
}
