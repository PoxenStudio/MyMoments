# 05 · PWA 实现方案

朝夕以**响应式 Web (PWA)** 交付（PRD §1 客户端），桌面与手机浏览器完整可用，可安装、离线壳、后续移动端复用同一 API。PWA 用 `vite-plugin-pwa` + Workbox 生成，不引入独立移动框架。

## 1. Web App Manifest

`public/manifest.webmanifest`：

```json
{
  "name": "朝夕",
  "short_name": "朝夕",
  "description": "家庭生活记录与规划面板",
  "lang": "zh-CN",
  "dir": "ltr",
  "start_url": "/",
  "scope": "/",
  "display": "standalone",
  "orientation": "any",
  "background_color": "#f5f5f5",
  "theme_color": "#f5f5f5",
  "icons": [
    { "src": "/icons/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/icons/icon-512.png", "sizes": "512x512", "type": "image/png" },
    { "src": "/icons/icon-maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable" }
  ]
}
```

- 图标由 `docs/logo/mymoment.svg` 派生，生成 192 / 512 / maskable-512 / apple-touch-icon（180）。
- `theme_color` 取画布 `#f5f5f5`，与 DESIGN.md 一致；不设高饱和主题色。

## 2. Service Worker 缓存策略

Workbox `generateSW`，按资源类型分策略：

| 资源 | 策略 | 理由 |
|---|---|---|
| 应用壳（index.html） | `NetworkFirst`（离线回退缓存壳） | 保证最新版本，断网仍可启动 SPA |
| 静态资源（带 hash 的 JS/CSS/字体/图标） | `CacheFirst`（不可变） | 内容寻址，长期缓存 |
| `/api/v1/*` | `NetworkOnly` | 业务数据以服务端为唯一事实源，不在浏览器离线持久化（PRD §18） |
| 图片缩略图（已鉴权返回） | `StaleWhileRevalidate`（可选） | 仅用于已鉴权视图的观感缓存，不回退为事实源 |

**数据原则**：离线时不伪造「无数据」，而是让 SPA 壳正常加载，API 请求失败时由 `AppErrorState` 呈现「网络连接失败 + 重试」。已登录会话下的只读摘要可选做「上次浏览快照」缓存，但明确标注非实时（首期默认关闭）。

### Workbox 配置要点

```ts
VitePWA({
  registerType: 'autoUpdate',
  manifest: false,            // 使用 public/manifest.webmanifest
  workbox: {
    navigateFallback: '/index.html',
    navigateFallbackDenylist: [/^\/api\//, /^\/openapi/],
    runtimeCaching: [
      { urlPattern: /^\/api\/v1\//, handler: 'NetworkOnly' },
      { urlPattern: /\/assets\//, handler: 'CacheFirst', options: { cacheName: 'assets' } }
    ]
  }
})
```

## 3. 安装（Installability）

- 满足 PWA 安装条件：HTTPS、有效 manifest、注册 Service Worker。
- 监听 `beforeinstallprompt`，在「设置 → 关于/安装」提供「安装到主屏幕」按钮；已安装则隐藏。
- iOS Safari 不支持 `beforeinstallprompt`：用 `apple-mobile-web-app-capable` / `apple-mobile-web-app-title` / `apple-touch-icon` meta 标签，引导用户「分享 → 添加到主屏幕」。

```html
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="default">
<meta name="apple-mobile-web-app-title" content="朝夕">
<link rel="apple-touch-icon" href="/icons/apple-touch-icon.png">
```

## 4. 更新流程

- `registerType: 'autoUpdate'`：新 SW 后台安装，`skipWaiting + clients.claim` 激活。
- 前端监听 `serviceWorkerUpdate` 事件，弹出「新版本已就绪」提示，用户刷新即得最新（避免打断进行中的填写/上传）。
- 应用壳带 hash，版本更新走 CacheFirst 自动失效旧缓存。

## 5. 离线与安全区

- `offline.html` 仅作为极端回退（导航离线且无缓存壳时）；常规离线由 SPA 壳 + 错误态承接。
- iOS standalone 底部安全区：容器使用 `padding-bottom: env(safe-area-inset-bottom)`，避免底部导航被 Home 指示条遮挡（PRD §5.2 底部导航不得遮挡内容）。
- `viewport` 加 `viewport-fit=cover`，配合 safe-area 变量。

## 6. 通知（PRD §21.5，分期）

| 能力 | 状态 | 实现 |
|---|---|---|
| 站内消息列表 + @未读角标 | 首期必需 | 轮询/SSE 拉取 `inbox_notifications`，与 PWA 无关 |
| 应用前台本地提醒 | 首期 | 应用打开时用 `Notification` API 展示到点提醒 |
| 后台/锁屏提醒（Web Push） | 待定 | 需 VAPID + 推送订阅表 + 服务端推送；家庭自托管 + 反向代理下需额外说明 HTTPS 与推送服务可达性 |

**Web Push 设计预留**：若启用，新增 `push_subscriptions` 表（user_id、endpoint、p256dh、auth、失效时间）；服务端提醒调度器命中 `remind_at` 时向订阅端点推送；浏览器无权限或被代理阻断时回退站内通知。首期不承诺后台推送（PRD §21.5）。

## 7. 验收映射

对应 PRD §19 的可安装/离线相关项：

- [ ] PWA 可安装、`display: standalone` 启动、主题色/图标正确（§19 整体 + §8）。
- [ ] 断网时应用壳可打开，卡片/详情呈现网络失败态而非空白（§19.6）。
- [ ] 手机底部导航不被 Home 指示条遮挡（§5.2）。
- [ ] Service Worker 更新不打断进行中操作，提示后刷新生效（§4）。
