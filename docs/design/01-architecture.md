# 01 · 系统架构

## 1. 总体架构

家时光是**单容器单体应用**：一个 Go 进程同时提供版本化 JSON API 和静态资源（Vue 构建产物），数据落在 SQLite 与本地媒体目录。

```text
┌────────────────────────────── 浏览器 / PWA ──────────────────────────────┐
│  Vue 3 SPA（含 Service Worker）                                          │
│  ├─ 首页卡片网格 / 行事历 / 待办 / 时光轴 / 家庭 / 留言 / 便签 / 设置      │
└───────────────────────────────┬──────────────────────────────────────────┘
                                 │ HTTPS（反向代理终止 TLS）
                                 ▼
┌────────────────────────── Docker 容器（非 root）─────────────────────────┐
│  Go 服务（单一进程）                                                     │
│  ├─ HTTP 层：/api/v1/*（JSON）+ /（静态资源）+ /healthz + /readyz         │
│  ├─ 中间件：会话认证 / 权限 / 限流 / 日志 / 恢复 / CORS(同源)              │
│  ├─ 业务：dashboard · task · calendar · note · message · contact ·        │
│  │        birthday · reminder · timeline · weather · token               │
│  ├─ 存储：SQLite（WAL） + 本地媒体目录                                    │
│  └─ 提醒调度器（进程内定时扫描）                                          │
└──────────────────────┬──────────────────────┬────────────────────────────┘
                       │                      │
              /data/app.db（SQLite）   /data/media/（媒体字节）
                       └──────────┬───────────┘
                        同一持久化卷（named volume / bind mount）
```

## 2. 技术栈与理由

| 组件 | 选型 | 理由 |
|---|---|---|
| 语言/运行时 | Go 1.23+ | PRD 指定；单二进制部署、低内存、便于交叉编译 |
| HTTP | `net/http` + Go 1.22 `ServeMux` 模式路由 | 减少依赖；1.22 起支持 `GET /api/v1/tasks/{id}` 通配 |
| SQLite 驱动 | `modernc.org/sqlite` | 纯 Go 免 CGO，`GOOS=linux GOARCH=arm64` 免交叉编译工具链 |
| 口令哈希 | `golang.org/x/crypto/argon2`（argon2id） | 抗 GPU；参数见 [06-security](06-security.md) |
| OpenAPI | 手写 `openapi.yaml` + `kin-openapi` 校验 | 作为 Web/移动端/AI Skill 唯一接口契约 |
| 前端 | Vue 3.5 + TypeScript + Vite | PRD 指定；`<script setup>` + Composition API |
| 状态 | Pinia | PRD 指定集中式状态 |
| 路由 | vue-router 4 | SPA + deep-link（`?open=&date=`） |
| 日期 | `date-fns` + `date-fns-tz`（或 `dayjs` + 时区插件） | date-only 与家庭时区计算需统一封装 |
| PWA | `vite-plugin-pwa` + Workbox | 生成 manifest 与 Service Worker |

## 3. 进程模型与并发

- 单进程；HTTP 请求并发由 `net/http` goroutine 处理。
- SQLite 使用 WAL 模式，写请求串行化（单写者），读并发；单实例天然规避多写者问题。
- **提醒调度器**：进程内每 30s 扫描一次 `reminder` 表（`due_at <= now` 且未处理），批量投递站内通知；Web Push 待定（PRD §21.5）。调度器与 HTTP 共享同一 `context`，随优雅关闭退出。
- **优雅关闭**：收到 SIGTERM → 停止接受新连接 → 等待在途请求（超时上限 10s）→ 停止调度器 → 关闭 SQLite（checkpoint）→ 退出。与 PRD §18.3「SIGTERM 优雅关闭」一致。

## 4. 请求生命周期

```text
请求 → recover 中间件 → 会话/Token 认证 → 限流 → 路由 → 参数校验
     → 业务 handler（事务边界）→ 权限判定 → 读写 SQLite/媒体 → 序列化响应
     → 审计(可选) → 结构化日志 → 响应
```

- 认证二选一：`Cookie`（浏览器会话）或 `Authorization: Bearer <token>`（API 客户端）。
- 权限判定在 handler 内、写操作前执行；不依赖前端传入的可见性。

## 5. 模块 → 服务端/前端映射

| PRD 章节 | 功能 | 服务端模块 | 前端视图/卡片 |
|---|---|---|---|
| §6 | 首页与卡片 | `dashboard` | 首页 + 卡片注册表 |
| §7 | 家庭/成员 | `family` | 家庭 → 成员 |
| §8 | 待办/模板 | `task`, `template` | 待办、首页待办卡 |
| §9 | 行事历 | `calendar` | 行事历、日历卡 |
| §10 | 生日/通讯录 | `birthday`, `contact` | 家庭 → 生日/通讯录 |
| §11 | 留言板 | `message` | 留言板、留言卡 |
| §12 | 便签 | `note` | 便签、便签卡 |
| §13 | 时光轴 | `timeline` | 时光轴、回顾卡 |
| §14 | 提醒/天气 | `reminder`, `weather` | 消息、提醒卡、天气卡 |
| §15 | 设置 | `settings` | 设置 |
| §18 | API Token/Skill | `token`, `openapi` | 设置 → API 与 AI Agent |

## 6. 目录结构

```text
MyMoments/
├─ cmd/server/            # 入口 main，只做装配
├─ internal/
│  ├─ http/               # 路由、中间件、handler
│  ├─ auth/               # 会话、密码、Token
│  ├─ service/            # 业务用例（dashboard/task/calendar/...）
│  ├─ store/              # SQLite 数据访问 + 迁移
│  ├─ media/              # 媒体存储、缩略图、Range 流
│  ├─ remind/             # 提醒调度器
│  └─ weather/            # 天气代理（超时/缓存/降级）
├─ migrations/            # 向前 SQL 迁移（embed）
├─ web/                   # Vue 3 前端（独立 Vite 工程）
├─ skills/mymoments/      # AI Agent SKILL.md
├─ openapi.yaml
├─ Dockerfile
└─ docker-compose.yml
```

依赖方向单向：`http → service → store`；`service` 不反向依赖 `http`。共享跨模块查询（如时光轴聚合）放 `service/timeline`，不散落各模块。
