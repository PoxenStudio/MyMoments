# 09 · Tauri 应用方案（macOS / iOS / Android）

> 状态：已实施。工程位于 `apps/`，发布流水线见 `.github/workflows/release-apps.yml`。**不做代码签名**，只输出未签名安装包，由用户自行签名安装。

## 1. 背景与目标

家时光当前以**响应式 Web（PWA）**交付，桌面与手机浏览器完整可用（见 [05-pwa](05-pwa.md)）。原生壳在三端补齐 PWA 无法覆盖的诉求：

1. **原生应用形态**：独立 Dock / 桌面图标，不被淹没在浏览器中，可上架 App Store / Google Play。
2. **全屏沉浸（桌面）**：作为家庭面板常驻屏（客厅/床头屏），需要全屏 webview 展示。
3. **关闭即退出（桌面）**：点击窗口关闭按钮真正结束进程，而非 macOS 默认「关窗不退」。

本方案引入 **Tauri v2** 跨平台壳，覆盖 **macOS 桌面 + iOS + Android（arm64）**，采用 **连接已有服务器** 形态（已确认）：应用不内置后端，webview 加载用户指定的服务器地址，数据仍在服务器端，三端共享同一套壳逻辑。

## 2. 需求与约束

| 项 | 内容 |
|---|---|
| 复用 webui | 三端 webview 直接加载服务器 URL，前后端同源，**零前端改造** |
| 全屏显示 | 桌面全屏可退出；移动端天然全屏（含 safe area 适配） |
| 应用关闭处理 | macOS 关窗 = 退出；Android 返回键语义明确；iOS 走系统生命周期 |
| 后端形态 | 连接已有服务器（非 sidecar），数据在远端 |
| 平台 | macOS（Apple Silicon）、iOS、Android（arm64） |
| 目标架构 | `aarch64-apple-darwin` / `aarch64-apple-ios` / `aarch64-linux-android` |

## 3. 现状分析与复用性

结论先行：**当前架构对「原生壳」天然友好，几乎无需改动前端与后端。**

| 现状 | 对原生壳的影响 |
|---|---|
| 前后端同源（Go 服务同时提供 `/api/v1` 与静态资源，见 [01-architecture](01-architecture.md)） | 三端直接加载服务器 URL 即复用，无 CORS/跨域问题 |
| `api/client.ts` 以相对路径 `BASE='/api/v1'` 请求 | 无需为原生壳改 API 地址 |
| 路由 `createWebHistory()`（HTML5 history，见 [04-frontend](04-frontend.md)） | 依赖服务器 SPA fallback；Go 服务已具备，刷新/深链正常 |
| PWA `vite-plugin-pwa` 注入 Service Worker | 原生壳中 SW 冗余且可能缓存旧壳，见 §6.7 |
| 已有 `safe-area-inset-bottom` 适配（见 [05-pwa](05-pwa.md)） | iOS 底部 Home 指示条已兼容，移动壳直接受益 |
| 存在 `target="_blank"` 外链（事件地点地图链接） | 需拦截导航，外链交系统浏览器，见 §6.5 |

**复用结论**：原生壳 = 系统 WebView 全屏加载服务器 URL，前端与后端代码不动，仅新增 `desktop/`（或统一 `apps/`）目录。

**与现有 PWA 的定位**：PWA 继续服务「浏览器直接使用」场景；原生壳的增量价值是**应用商店分发 + 原生推送 + 桌面常驻全屏**。若移动端仅需「能安装到主屏」，PWA 已足够，原生壳主要在需要商店分发或原生能力时启用（见 §4 取舍）。

## 4. 技术选型评估

### 4.1 三端统一壳：Tauri 2 vs Capacitor vs 纯 PWA

| 维度 | Tauri 2 | Capacitor | 纯 PWA（现有） |
|---|---|---|---|
| iOS / Android | 支持（系统 WebView） | 支持（成熟） | 无需原生，浏览器/PWA |
| 桌面 | 支持（macOS/Linux/Windows） | 需另配 Electron | 浏览器 |
| WebView 引擎 | 系统 WKWebView / WebView | 系统 WebView | 浏览器 |
| 语言栈 | Rust | JS/TS | 无 |
| 三端一套工程 | ✅ 一套 Rust 工程 | ❌ 移动(Capacitor) + 桌面(Electron) 两套 | ✅ |
| 商店分发 | 支持 | 支持 | 需额外商店封装 |
| 原生推送 | 可接 APNs/FCM（写 Rust 桥） | 插件生态成熟 | Web Push（待定，见 05） |
| 体积/内存 | 小 | 移动端小、桌面 Electron 大 | 无 |

**结论**：选 **Tauri 2**，三端统一。理由：

- 需求本质是「系统 WebView 加载远程 URL + 全屏 + 生命周期控制」，Rust 代码量极小，学习成本可控。
- Tauri 2 一套工程同时产出 macOS / iOS / Android，避免 Capacitor「移动 + Electron 桌面」两套栈的维护割裂。
- 体积与内存显著优于 Electron（桌面端），移动端与 Capacitor 同为系统 WebView，无劣势。

**取舍说明**：

- 若团队倾向纯 JS 栈、且桌面端优先级低，Capacitor 移动端更成熟、插件生态更丰富；但桌面壳需另立 Electron，与「三端统一」目标冲突。
- 原生推送（APNs/FCM）无论 Tauri 还是 Capacitor 都需服务端推送基础设施，属后续独立工程，不阻塞首版壳（首版沿用 PWA 的站内通知，见 [05-pwa](05-pwa.md) §6）。

**备选说明**（本轮不采用）：

- **sidecar 内置后端**：把 Go 服务作为子进程随应用启停。iOS/Android 不允许任意子进程 + 本地监听端口，且与本项目「家庭自托管 + 多端共用同一服务器」定位相悖。已确认不采用。

## 5. 架构设计

### 5.1 统一壳模式架构

```text
┌────────────── 原生壳（Tauri 2，一套 Rust 工程，三端）─────────────────────┐
│  系统 WebView 加载内容（WKWebView / WebView）                              │
│  ┌──────────────────────────────────────────────────────────────────────┐ │
│  │  首次启动：本地引导页（frontendDist，输入服务器地址）                    │ │
│  │  配置后：navigate → 用户服务器 URL（复用现有 webui）                    │ │
│  └──────────────────────────────────────────────────────────────────────┘ │
│  Rust 主进程                                                              │
│  ├─ 服务器地址读写（本地配置，§6.4）                                       │
│  ├─ 生命周期：桌面 fullscreen / CloseRequested；移动端返回键、前后台        │
│  └─ 导航拦截：外链交系统浏览器（§6.5）                                     │
└──────────────────────────────┬───────────────────────────────────────────┘
                               │ HTTPS（或局域网 HTTP，见 §6.6）
                               ▼
                 现有 Go 服务（家庭自托管 / 局域网 / 公网反向代理）
```

### 5.2 启动流程（三端一致）

```text
启动 → 读取本地服务器地址配置
  ├─ 已配置 → 全屏加载该 URL
  └─ 未配置 → 加载本地引导页，用户输入地址并保存 → navigate 到该 URL
```

桌面端单窗口承载「引导页 → 主界面」两阶段；移动端为单 WebView，同样以 `navigate` 切换，避免多页面管理。

## 6. 关键技术点设计

### 6.1 加载远程服务器

- 服务器地址来自本地配置，**不硬编码**进 `tauri.conf.json`。
- 已配置时以 `WebviewUrl::External(url)` 加载；未配置时以 `frontendDist` 的本地引导页加载，配置完成后 `navigate(url)` 切换。
- 桌面端用 `WebviewWindowBuilder`，移动端用 `WebviewBuilder`（移动端无多窗口，单 WebView）。
- 加载目标限定 `http` / `https`；其余导航一律拦截（§6.5）。

### 6.2 全屏显示

| 平台 | 方案 |
|---|---|
| macOS | 窗口 `fullscreen: true` 进入系统全屏 Space；保留标题栏（`decorations` 默认）以便退出全屏与关闭 |
| iOS | 原生全屏；沿用前端已有 `viewport-fit=cover` + `safe-area-inset-bottom`，无需额外处理 |
| Android | 原生全屏；可按需通过 Tauri/主题配置沉浸式（隐藏状态栏/导航栏），首版保持默认 |

### 6.3 应用关闭处理（三端语义不同）

| 平台 | 关闭语义 | 实现 |
|---|---|---|
| macOS | 关窗 = 退出 | 拦截 `CloseRequested` → 进程退出 |
| iOS | 无「关闭」概念，系统管理前后台/被杀 | 无需特殊处理；本地配置持久保存，前台恢复自动连回 |
| Android | 首页返回键 = 退出；非首页回退浏览历史 | 沿用 Android WebView 默认返回栈：可回退时回退，已到首页时由系统结束应用 |

macOS 示意：

```rust
.on_window_event(|window, event| {
    if let tauri::WindowEvent::CloseRequested { api, .. } = event {
        api.prevent_close();                 // 阻止默认的「仅关窗」
        window.app_handle().exit(0);         // 真正退出应用
    }
})
```

说明：

- 连接已有服务器形态下**无子进程需清理**，关闭处理仅需「关窗 = 退出」。
- 本地服务器地址配置持久保存，退出不清理，下次启动自动连回。
- Android 返回键不注册自定义监听，交给系统 WebView 默认行为：非首页回退历史，首页（无法再回退）时退出应用，不做二次确认。

### 6.4 服务器地址配置与持久化

| 环节 | 方案 |
|---|---|
| 首次引导 | 本地静态引导页 `apps/ui/`，输入服务器地址（`https://moments.example.com` 或 `http://192.168.1.5:8080`），校验后保存 |
| 存储 | 应用配置目录下的 `server.json`（形如 `{"server_url": "https://..."}`），Rust 直接读写，无额外插件依赖 |
| 修改入口 | macOS 菜单「家时光 → 更改服务器地址…」清空来源记忆并返回引导页；移动端重装或清除数据后回到引导页 |

### 6.5 外链与导航拦截

- 主界面存在 `target="_blank"` 的地图外链（[EventDetailDialog.vue](../../web/src/components/business/EventDetailDialog.vue)）。
- 通过 `on_navigation` 拦截：同源 http/https → 放行；其余 http/https → 交系统浏览器（`tauri-plugin-opener`）；`mailto:`/`tel:` → 交系统处理。
- 禁止 webview 打开任意非白名单 URL，避免被重定向到外部页面后失去「返回」能力。

### 6.6 明文 HTTP：ATS（iOS）与 cleartext（Android）

家庭自托管常以局域网 HTTP 访问，两端默认都阻断明文：

- **iOS**：`Info.plist` 注入 `NSAppTransportSecurity → NSAllowsLocalNetworking = true`，仅放开本地网络明文，比 `NSAllowsArbitraryLoads` 更收敛。仓库内的 `apps/src-tauri/Info.plist` 提供该键；由于 iOS 工程由 `tauri ios init` 重新生成，流水线在 init 之后再用 `PlistBuddy` 兜底写入。
- **Android**：`tauri android init` 生成的 `build.gradle.kts` 默认设置 `usesCleartextTraffic = "false"`；流水线在 init 之后把它改为 `"true"`（应用只访问用户自己配置的服务器）。
- 优先引导 HTTPS（反向代理，见 [07-deployment](07-deployment.md)）。

### 6.7 Service Worker 处理

- 前端 `injectRegister: 'auto'` 会在 webview 中尝试注册 SW。原生壳中 SW 的离线/更新价值有限，反而可能缓存旧壳。
- **首版策略**：不改前端，接受 SW 注册失败/无效；若实测出现旧壳缓存，再评估通过 UA 或构建参数在原生壳中禁用 SW 注入。
- 列为待验证项（§9），不阻塞方案成立。

### 6.8 目标架构（arm64）

| 平台 | Rust target | 构建环境 |
|---|---|---|
| macOS | `aarch64-apple-darwin`（本机 Apple Silicon） | macOS + Xcode CLT |
| iOS 真机 | `aarch64-apple-ios` | macOS + Xcode |
| iOS 模拟器 | `aarch64-apple-ios-sim`（可选） | macOS + Xcode |
| Android | `aarch64-linux-android`（arm64-v8a） | Android SDK + NDK |

- 仅出 arm64 首版（用户明确），Android 不构建 x86/armeabi-v7a，后续按需补。
- **不做代码签名**：macOS 产物为未签名 `.dmg`，iOS 为未签名 `.ipa`，Android 为未签名 `.apk`，由用户自行签名安装（见 §11）。

## 7. 目录与文件结构

```text
apps/
├─ package.json                  # 固定 @tauri-apps/cli 版本，仅作构建入口
├─ package-lock.json
├─ .gitignore                    # node_modules / src-tauri/target / src-tauri/gen
├─ ui/                           # 本地引导页（输入服务器地址）
│  ├─ index.html
│  └─ main.js
└─ src-tauri/
   ├─ Cargo.toml                 # tauri 2、tauri-plugin-opener、serde_json
   ├─ build.rs
   ├─ tauri.conf.json            # productName/identifier/fullscreen/frontendDist=../ui
   ├─ Info.plist                 # CFBundleDisplayName + ATS（NSAllowsLocalNetworking）
   ├─ capabilities/default.json  # core:default、opener:default
   ├─ icons/                     # 由 docs/logo/mymoment.png 派生（32/128/256/512 + icns）
   └─ src/
      ├─ main.rs
      ├─ lib.rs                  # 窗口导航、导航拦截、菜单、关闭处理
      └─ server_config.rs        # 服务器地址校验与持久化
```

> 图标复用现有品牌资源 `docs/logo/mymoment.png`（与 PWA 图标同源），不新增设计资产；流水线每次构建前用 `tauri icon` 重新生成完整尺寸集。

## 8. 实施步骤与状态

| 步骤 | 内容 | 状态 |
|---|---|---|
| 1 | `src-tauri` 工程 + 窗口全屏 + 关闭即退出 | 已完成（`apps/src-tauri`） |
| 2 | 本地引导页 + 服务器地址读写 + 自动连回 | 已完成（`apps/ui`、`server_config.rs`） |
| 3 | 导航拦截 + 外链交系统浏览器 | 已完成（`lib.rs::allow_navigation`） |
| 4 | iOS ATS 例外 + Android cleartext + 图标 | 已完成（`Info.plist`、流水线注入） |
| 5 | iOS / Android 工程接入（Tauri mobile） | 由流水线 `ios init` / `android init` 生成 |
| 6 | 打包分发（不签名） | 已完成（`release-apps.yml`） |

## 9. 风险与开放问题

| 风险 / 问题 | 影响 | 处置建议 |
|---|---|---|
| WKWebView / WebView 中 Service Worker 行为不确定 | 可能缓存旧壳 | 待验证；必要时禁用原生壳 SW（§6.7） |
| 公网裸 HTTP 需放宽 ATS/cleartext | 安全隐患 | 引导 HTTPS；仅放开本地网络明文 |
| iOS App Store 审核（ATS 例外、HTTP） | 可能被拒 | 优先 HTTPS；ATS 仅 `NSAllowsLocalNetworking`，避免 ArbitraryLoads |
| Tauri 2 移动端相对桌面端年轻 | 原生能力桥接需写 Rust | 首版仅壳 + 生命周期，原生能力（推送等）延后独立评估 |
| 服务器地址变更/失效 | 白屏或连不上 | 桌面菜单可回到引导页；移动端清除数据后重入 |
| 多服务器/多家庭切换 | 当前单地址 | 后续扩展为「服务器列表」选择器 |
| 未签名产物的安装体验 | macOS Gatekeeper / iOS 需自签 / Android 需允许未知来源 | 见 §11 用户侧说明 |
| 自签名 HTTPS 证书 | 默认被系统 WebView 拒绝 | 优先受信证书；如需自签再评估证书信任注入 |

## 10. 验收标准（首版）

- [ ] macOS 应用可安装，独立图标，全屏加载服务器，关闭窗口后进程退出、Dock 无残留。
- [ ] iOS 真机（arm64）可安装运行，全屏加载服务器，登录/卡片/日历等与浏览器一致。
- [ ] Android（arm64-v8a）可安装运行，首页按返回键退出应用，非首页回退历史。
- [ ] 三端首次启动输入服务器地址后自动连回；事件地点外链在系统浏览器打开。
- [ ] 局域网 `http://` 三端可正常访问（ATS / cleartext 例外生效），HTTPS 正常工作。
- [ ] `release-apps.yml` 一次运行产出 macOS `.dmg`、iOS `.ipa`、Android `.apk` 三个未签名安装包（产物名统一小写）。

## 11. 构建与发布（GitHub Actions）

流水线：[`release-apps.yml`](../../.github/workflows/release-apps.yml)。三端并行构建，产物上传到同一个 GitHub Release。

### 11.1 触发与版本

| 项 | 规则 |
|---|---|
| 触发 | `workflow_dispatch`（可指定 tag）或 `release: published` |
| 版本号 | 取 `apps/src-tauri/tauri.conf.json` 的 `version` |
| Release | tag 已存在则复用，否则创建 draft release |

### 11.2 各端任务

| 任务 | Runner | 关键步骤 | 产物 |
|---|---|---|---|
| `macos` | `macos-latest` | `tauri build --target aarch64-apple-darwin --bundles app,dmg` | `mymoments_<ver>_macos_arm64.dmg` |
| `ios` | `macos-latest` | `tauri ios init --ci` → PlistBuddy 注入 ATS → `tauri ios build --no-sign` | `mymoments_<ver>_ios_unsigned.ipa` |
| `android` | `ubuntu-latest` | `tauri android init --ci` → 放开 cleartext → `tauri android build --apk --target aarch64` | `mymoments_<ver>_android_arm64_unsigned.apk` |

- 三端构建前都先跑 `tauri icon ../docs/logo/mymoment.png`，保证 iOS/Android 图标齐全。
- **不设置任何签名密钥**：不注入 `APPLE_CERTIFICATE`、`TAURI_SIGNING_PRIVATE_KEY`、Android keystore，因此 `tauri build` 走未签名/临时签名路径，`ios build --no-sign` 跳过签名，Android release 构建产出 `-unsigned.apk`。

### 11.3 用户侧签名与安装

| 平台 | 用户操作 |
|---|---|
| macOS | 打开 dmg 拖入「应用程序」；首次启动右键「打开」绕过 Gatekeeper（未公证）。如需正式分发，自行用开发者证书重签并公证。 |
| iOS | 用 Apple 配置工具或 `codesign` + 自签描述文件对 `.ipa` 重签后安装（如 Sideloadly / AltStore 流程）。 |
| Android | `apksigner sign --ks <keystore> --out signed.apk mymoments_<ver>_android_arm64_unsigned.apk` 后安装，或在设备上允许未知来源直接安装（系统会提示未签名拒绝，故需先签名）。 |

### 11.4 本地构建

```bash
cd apps
npm ci
npm run tauri -- icon ../docs/logo/mymoment.png   # 可选，刷新图标
npm run tauri -- dev                              # macOS 调试
npm run tauri -- build --target aarch64-apple-darwin --bundles dmg
```

移动端需先 `npm run tauri -- ios init` / `android init`，再 `ios dev` / `android dev`。
