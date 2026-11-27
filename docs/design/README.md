# 朝夕（MyMoments）技术方案

本目录是朝夕首期的技术设计文档，作为 [PRD](../spec/PRD-MyMoments.md) 的实现蓝图。产品行为以 PRD 为基线，本文档确定字段类型、HTTP 接口、迁移脚本、组件拆分与部署方式。

## 文档索引

| 文档 | 内容 |
|---|---|
| [01-architecture.md](01-architecture.md) | 系统架构、技术栈、进程模型、模块映射、目录结构 |
| [02-data-model.md](02-data-model.md) | SQLite 表结构、DDL、向前迁移、时间/重复语义 |
| [03-backend-api.md](03-backend-api.md) | Go 服务、路由、鉴权、错误码、分页、幂等、API Token、OpenAPI |
| [04-frontend.md](04-frontend.md) | Vue 3 架构、路由、状态、组件库、卡片系统、设计令牌 |
| [05-pwa.md](05-pwa.md) | PWA 清单、Service Worker、离线策略、安装与更新、通知 |
| [06-security.md](06-security.md) | 认证、会话、密码、权限执行、安全头、媒体访问控制 |
| [07-deployment.md](07-deployment.md) | Docker、Compose、健康检查、备份恢复、迁移与升级 |
| [08-style-customization.md](08-style-customization.md) | 样式自定义：三层级主题继承、色调/圆角/背景图/卡片角图、头像昵称、快速恢复默认 |
| [09-tauri-apps.md](09-tauri-apps.md) | Tauri 跨平台壳：macOS / iOS / Android 连接已有服务器、全屏 webview、生命周期与未签名打包流水线 |

## 技术栈总览

| 层 | 选型 | 说明 |
|---|---|---|
| 后端 | Go 1.23+ | `net/http` + Go 1.22 模式路由，自建轻量中间件链 |
| 数据库 | SQLite（`modernc.org/sqlite`） | 纯 Go、免 CGO，便于 amd64/arm64 交叉编译；WAL + 外键 |
| 前端 | Vue 3.5 + TypeScript + Vite | Composition API + `<script setup>`；Pinia + vue-router |
| UI | 自研 `App*` 组件库 | 令牌取自 DESIGN.md；**不使用 Vuetify**（见下方说明） |
| PWA | `vite-plugin-pwa` + Workbox | standalone、可安装、离线壳、网络优先 |
| 认证 | Session Cookie + 服务端会话 | argon2id 口令哈希；API 客户端用 Bearer Token |
| 部署 | Docker 单容器 + Compose | 非 root、健康检查、`linux/amd64` + `linux/arm64` |

## 关键架构决策

1. **单容器单体**：Go 服务内嵌 Vue 构建产物，SQLite 与媒体文件位于同一持久化卷。一个库只允许一个实例读写，不做横向扩展（PRD §18.3）。
2. **日期语义统一**：date-only（本地日历日）、local datetime、带时区 timestamp 三类严格区分；家庭时区是重复、提醒、日历日界的唯一计算基准（PRD §17.1）。
3. **权限服务端强制执行**：列表、详情、搜索、卡片摘要、通知、API 全部走同一套授权判定；前端隐藏仅是体验优化（PRD §4.2.6）。
4. **卡片摘要隔离**：首页聚合接口按卡片拆分子查询，单卡片失败不使整页空白（PRD §18）。
5. **PWA 数据策略**：业务数据不落浏览器为唯一事实源；Service Worker 只缓存应用壳与静态资源，API 走网络优先、失败降级为可理解的错误态（PRD §18、§05）。

## 实施里程碑（8 个）

将 PRD §20 的 11 步交付顺序收敛为 8 个里程碑。每个里程碑的退出标准：**可运行 + 对应 PRD §19 验收项通过 + 迁移可前向**。依赖大体线性，M6 可与 M5 部分并行。

| 里程碑 | 名称 | 核心范围 | 验收锚点 | 设计文档 |
|---|---|---|---|---|
| M0 | 工程底座与部署骨架 | Go/Vue 工程与目录、CI（fmt/lint/typecheck/test）、多阶段 Dockerfile、Compose、健康检查、迁移框架、空库初始化入口占位 | 能 `docker compose up` 启动空服务，`/readyz` 反映迁移状态 | 01、07 |
| M1 | 初始化、账号与权限 | 首次初始化（并发安全）、登录/会话、管理员/成员、临时密码+首次改密、角色/停用、家庭设置、权限执行框架 | §19.1、§19.6（权限） | 02、06 |
| M2 | 待办、模板与媒体 | tasks/templates、重复（单一开放实例）、完成记录（勾选者+实际完成者）、指派、提醒、媒体上传+缩略图+Range、首页待办卡 | §19.3、§19.5（媒体） | 02、03 |
| M3 | 日历、生日与通讯录 | events、四视图、待办/生日图层、生日→日历、重复例外、contacts+多值 | §19.3、§19.5（生日） | 02、03 |
| M4 | 留言、便签与 @提及 | messages/replies、notes、mentions、inbox_notifications、个人消息列表、回忆日期/发生日期 | §19.4 | 02、03 |
| M5 | 时光轴 | 聚合投影、双向游标分页、类型/成员筛选、深链、回顾卡 | §19.5 | 03 |
| M6 | 首页卡片系统与天气 | 卡片个性化（排序/尺寸/背景）、家庭默认布局、天气代理+省市、天气卡 | §19.2、§19.4（天气） | 04 |
| M7 | API Token + AI Skill + 收尾 | 版本化 API 收敛、OpenAPI、Token/scope/审计、AI Skill、跨模块搜索、无障碍/移动回归、备份恢复验证、PWA 打磨 | §19.7、§19.6、§19.4（搜索） | 03、05 |

实施节奏建议：M0→M1→M2→M3→M4→M5 严格串行（前一个提供数据与权限基础）；M6 依赖 M2–M5 的卡片摘要接口，可在 M5 收尾时并行启动；M7 在功能稳定后统一收敛 API 契约与 PWA 细节。

## 与现有文档的关系（需团队确认）

[`docs/CodingStandards.md`](../CodingStandards.md)（STD-ENG-001）§15–18 的 WebUI 部分引用 **Vuetify** 主题、`--at-color-primary: #2563EB` 蓝色主色、Vuetify 断点（600/960/1280/1600），以及 Rust/Protobuf/NATS/ClickHouse 等微服务规范——这些与该规范本身更像某 IoT 平台的通用模板，与本项目（Vue3 + Go + SQLite 单容器）及 [DESIGN.md](../DESIGN.md) 的「暖黑墨色 + 白底」编辑部品牌**相冲突**。

本方案的处理口径：

- 沿用 CodingStandards 的**组件纪律与命名**（`App*` 前缀、`--at-` 令牌前缀、`useXxxStore`、`use` 前缀组合式函数、目录结构约定）。
- 令牌**数值**以 DESIGN.md 为准（主色 `#292524` 墨色、画布 `#f5f5f5`、无高饱和 CTA 色），替换 §15.1 的蓝色表。
- **不引入 Vuetify**，改用自研 `App*` 基础组件承载 DESIGN.md 品牌。
- 断点以 DESIGN.md 为准（`<640` / `640–1024` / `>=1024`），与 PRD §16.3 一致。

> 建议：为 MyMoments 单独修订 CodingStandards 的 WebUI 章节，删除与本品无关的 Rust/Protobuf/NATS/ClickHouse 条目。此项列入 [07-deployment](07-deployment.md) 之外的技术债，不阻塞首期开发。
