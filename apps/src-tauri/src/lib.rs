mod server_config;

use std::sync::{Arc, Mutex};

use tauri::{Manager, Url, WebviewUrl, WebviewWindowBuilder};

pub const MAIN_WINDOW: &str = "main";

/// 本地引导页：相对路径用于窗口初始加载，绝对地址用于从服务器页返回。
const GUIDE_PAGE: &str = "index.html";
const GUIDE_URL: &str = "tauri://localhost/index.html";

/// 当前允许在 webview 内导航的服务器来源，跨回调共享。
#[derive(Clone, Default)]
pub struct SharedOrigin(Arc<Mutex<Option<String>>>);

fn is_app_scheme(url: &Url) -> bool {
    matches!(url.scheme(), "tauri" | "asset" | "about" | "blob" | "data")
}

fn remember(state: &SharedOrigin, url: &Url) {
    if let Ok(mut slot) = state.0.lock() {
        *slot = Some(url.origin().ascii_serialization());
    }
}

fn forget(state: &SharedOrigin) {
    if let Ok(mut slot) = state.0.lock() {
        *slot = None;
    }
}

/// 只放行应用内部页面与已配置服务器同源的导航，其余交给系统浏览器。
fn allow_navigation(url: &Url, state: &SharedOrigin) -> bool {
    if is_app_scheme(url) {
        return true;
    }
    let current = state.0.lock().ok().and_then(|slot| slot.clone());
    if current.as_deref() == Some(url.origin().ascii_serialization().as_str()) {
        return true;
    }
    if matches!(url.scheme(), "http" | "https" | "mailto" | "tel" | "sms") {
        let _ = tauri_plugin_opener::open_url(url.as_str(), None::<&str>);
    }
    false
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
    remember(state.inner(), &target);
    window.navigate(target).map_err(|err| err.to_string())
}

fn create_window(
    app: &tauri::AppHandle,
    target: WebviewUrl,
    state: &SharedOrigin,
) -> tauri::Result<()> {
    let navigation_state = state.clone();
    #[allow(unused_mut)]
    let mut builder = WebviewWindowBuilder::new(app, MAIN_WINDOW, target)
        .title("家时光")
        .inner_size(1280.0, 800.0)
        .min_inner_size(480.0, 360.0)
        .on_navigation(move |url| allow_navigation(url, &navigation_state));
    #[cfg(desktop)]
    {
        builder = builder.fullscreen(true);
    }
    builder.build()?;
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
    let menu = MenuBuilder::new(app).item(&app_menu).build()?;
    app.set_menu(menu)?;
    Ok(())
}

fn handle_menu_event(app: &tauri::AppHandle, id: &str) {
    if id != "change-server" {
        return;
    }
    if let Some(state) = app.try_state::<SharedOrigin>() {
        forget(state.inner());
    }
    if let (Some(window), Ok(guide)) = (app.get_webview_window(MAIN_WINDOW), Url::parse(GUIDE_URL)) {
        let _ = window.navigate(guide);
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let state = SharedOrigin::default();

    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .manage(state)
        .invoke_handler(tauri::generate_handler![save_server_url])
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

            let target = match server_config::load(&handle) {
                Some(saved) => match server_config::normalize(&saved) {
                    Ok(url) => {
                        remember(&state, &url);
                        WebviewUrl::External(url)
                    }
                    Err(_) => WebviewUrl::App(GUIDE_PAGE.into()),
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
