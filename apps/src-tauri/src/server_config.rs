use std::fs;
use std::path::PathBuf;

use tauri::{AppHandle, Manager, Url};

const CONFIG_FILE: &str = "server.json";
const CONFIG_KEY: &str = "server_url";

/// 把用户输入规整为 http/https 服务器地址；未写协议时按 http 补齐。
pub fn normalize(input: &str) -> Result<Url, String> {
    let trimmed = input.trim();
    if trimmed.is_empty() {
        return Err("请输入服务器地址".into());
    }
    let candidate = if trimmed.contains("://") {
        trimmed.to_string()
    } else {
        format!("http://{trimmed}")
    };
    let url = Url::parse(&candidate).map_err(|_| "服务器地址格式不正确".to_string())?;
    if !matches!(url.scheme(), "http" | "https") {
        return Err("仅支持 http:// 或 https:// 地址".into());
    }
    if url.host_str().is_none() {
        return Err("服务器地址缺少主机名".into());
    }
    Ok(url)
}

/// 读取已保存的服务器地址；未配置或文件损坏时返回 None。
pub fn load(app: &AppHandle) -> Option<String> {
    let path = config_path(app).ok()?;
    let text = fs::read_to_string(path).ok()?;
    let value: serde_json::Value = serde_json::from_str(&text).ok()?;
    value
        .get(CONFIG_KEY)
        .and_then(|item| item.as_str())
        .map(str::to_string)
}

/// 持久化服务器地址，供下次启动自动连接。
pub fn save(app: &AppHandle, url: &str) -> Result<(), String> {
    let path = config_path(app)?;
    let payload = serde_json::json!({ CONFIG_KEY: url });
    let bytes = serde_json::to_vec_pretty(&payload).map_err(|err| err.to_string())?;
    fs::write(path, bytes).map_err(|err| err.to_string())
}

fn config_path(app: &AppHandle) -> Result<PathBuf, String> {
    let dir = app.path().app_config_dir().map_err(|err| err.to_string())?;
    fs::create_dir_all(&dir).map_err(|err| err.to_string())?;
    Ok(dir.join(CONFIG_FILE))
}
