# 03 · 后端 API 设计

## 1. 路由与中间件

统一前缀 `/api/v1`。使用 Go 1.22 `ServeMux` 模式路由：

```go
mux.HandleFunc("GET /api/v1/tasks", h.ListTasks)
mux.HandleFunc("POST /api/v1/tasks", h.CreateTask)
mux.HandleFunc("GET /api/v1/tasks/{id}", h.GetTask)
mux.HandleFunc("PATCH /api/v1/tasks/{id}", h.UpdateTask)
mux.HandleFunc("DELETE /api/v1/tasks/{id}", h.DeleteTask)
```

中间件链（从外到内）：`recover → logging → auth → rateLimit → handler`。`auth` 解析 Cookie 会话或 Bearer Token，注入 `ctx` 中的主体（user_id + 有效 scope）。

## 2. 资源端点清单

| 资源 | 方法 | 路径 | 说明 |
|---|---|---|---|
| 仪表盘 | GET | `/api/v1/dashboard` | 返回当前用户卡片布局 + 各卡片摘要（分块） |
| 仪表盘布局 | GET/PUT | `/api/v1/dashboard/layout` | 个人布局读写 |
| 家庭默认布局 | PUT | `/api/v1/household/default-layout` | 仅管理员 |
| 待办 | GET/POST | `/api/v1/tasks` | 列表（筛选/分页）/创建 |
| 待办 | GET/PATCH/DELETE | `/api/v1/tasks/{id}` | 详情/更新/删除 |
| 待办完成 | POST/DELETE | `/api/v1/tasks/{id}/completion` | 完成（含发生项）/撤销 |
| 待办模板 | GET/POST | `/api/v1/templates` | 模板管理 |
| 日历 | GET/POST | `/api/v1/events` | 按范围查询/创建 |
| 日历 | GET/PATCH/DELETE | `/api/v1/events/{id}` | 详情/更新/删除 |
| 便签 | GET/POST | `/api/v1/notes` | 列表/创建 |
| 便签 | GET/PATCH/DELETE | `/api/v1/notes/{id}` | 详情/更新/删除 |
| 留言 | GET/POST | `/api/v1/messages` | 列表/创建 |
| 留言 | GET/DELETE | `/api/v1/messages/{id}` | 详情/删除 |
| 留言回复 | POST | `/api/v1/messages/{id}/replies` | 回复 |
| 联系人 | GET/POST | `/api/v1/contacts` | 列表/创建 |
| 联系人 | GET/PATCH/DELETE | `/api/v1/contacts/{id}` | 详情/更新/删除 |
| 生日 | GET/POST | `/api/v1/birthdays` | 列表/创建 |
| 生日 | PATCH/DELETE | `/api/v1/birthdays/{id}` | 更新/删除 |
| 提醒 | GET/PATCH | `/api/v1/reminders` `/api/v1/reminders/{id}` | 我的提醒；标记已读/完成 |
| 通知 | GET | `/api/v1/notifications` | 个人消息列表 |
| 通知 | PATCH | `/api/v1/notifications/{id}/read` | 标记已读 |
| 时光轴 | GET | `/api/v1/timeline` | 聚合，游标分页，`?direction=past|future` |
| 天气 | GET | `/api/v1/weather` | 服务端代理，返回摘要 |
| 媒体 | GET | `/api/v1/media/{id}` | 鉴权媒体流（支持 Range） |
| 媒体 | POST | `/api/v1/media` | 上传（multipart） |
| 搜索 | GET | `/api/v1/search?q=` | 跨模块（待办/日历/通讯录/留言/便签） |
| 家庭/成员 | GET/POST/PATCH | `/api/v1/family/members...` | 管理员管理成员 |
| Token | GET/POST/DELETE | `/api/v1/tokens` | 个人 Token 管理 |

## 3. 响应与错误码

成功：直接返回资源或 `{ "data": ..., "page": { "next_cursor": "..." } }`。

错误统一信封（非 2xx）：

```json
{
  "error": {
    "code": "validation.invalid_field",
    "message": "截止时间不能早于开始时间",
    "fields": [{ "field": "due_date", "reason": "before start" }]
  }
}
```

稳定错误码（客户端据此映射文案与动作，PRD §5.3 三态）：

| code | HTTP | 场景 |
|---|---|---|
| `auth.unauthenticated` | 401 | 未登录/无凭据 |
| `auth.session_expired` | 401 | 会话过期 → 引导重新登录 |
| `auth.token_expired` | 401 | Token 过期 |
| `auth.rate_limited` | 429 | 限流 |
| `permission.denied` | 403 | 无权限（不泄露对象是否存在） |
| `validation.invalid_field` | 422 | 参数校验失败 |
| `not_found` | 404 | 对象不存在 |
| `conflict` | 409 | 幂等键不一致 / 并发冲突 |
| `media.too_large` | 413 | 超出家庭容量/单文件上限 |
| `media.unsupported_type` | 415 | 不支持的媒体类型 |
| `idempotency.mismatch` | 409 | 同键不同请求体 |
| `server.unavailable` | 503 | 上游/内部错误 → 稍后再试 |
| `network.*` | — | 客户端本地判定（无响应），非服务端返回 |

认证失败（401/403）不区分「Token 是否存在/属于谁」；`not_found` 与 `permission.denied` 对无权读取的对象统一返回 `not_found`，避免泄露存在性。

## 4. 分页与幂等

- **游标分页**：列表与时光轴用 `?cursor=<opaque>&limit=`（默认 50，上限 200）。游标为服务端签名的不透明串（含排序键 + 偏移），防篡改。响应含 `next_cursor`。
- **幂等**：`POST`/`PATCH` 支持 `Idempotency-Key` 头；服务端按 `(user_id, key)` 缓存结果 24h，重复提交返回首次结果（PRD §17.2）。重复任务/提醒/通知的生成另有业务幂等键（`reminders.idempotency_key`、`inbox_notifications.idempotency_key`）。

## 5. 权限判定

统一授权函数：

```go
func CanRead(ctx, user, obj) bool
func CanWrite(ctx, user, obj) bool
func CanDelete(ctx, user, obj) bool
```

规则（PRD §4.2）：

- `visibility=private` → 仅创建者可读写；管理员不越权。
- `visibility=selected` → 创建者 + 指定成员可读写；范围修改仅创建者。
- `visibility=household` → 有权用该模块的成员可读写（首期默认）；删除/归档/停用仅管理员或创建者。
- Token 有效权限 = `Token.scopes ∩ 创建者当前模块权限 ∩ 对象可见范围`（PRD §18.1）。
- 所有列表/详情/搜索/摘要/通知复用同一函数，禁止前端绕过。

## 6. API Token 与 AI Agent

- Token 明文仅创建时返回一次；库中存 `SHA-256(明文)` + 短前缀（PRD §18.1）。
- scope 编码为 JSON `{ "<module>": ["read","write","delete"] }`；`family/timeline/reminders/weather` 首期仅 `read`。
- 认证：`Authorization: Bearer <token>`；限流按 Token 与账号两级。
- 审计：`api_token_audit` 记录 token_id、时间、scope、结果、客户端标识；不记录正文/内容。
- OpenAPI：`openapi.yaml` 随镜像提供，路径 `GET /openapi.yaml`；作为唯一接口契约，CI 校验与实现一致（契约测试）。

## 7. 天气代理

- `GET /api/v1/weather` 由服务端代理上游（服务商待定，PRD §21.1），带超时（如 5s）、缓存（如 10min）与降级。
- 上游故障返回 503 或缓存值 + `stale:true`，不阻塞首页其它卡片（PRD §18）。
- 位置：优先用户偏好城市坐标；省市数据为版本化本地源，升级不覆盖用户已存坐标（PRD §18）。

## 8. 时光轴聚合

`GET /api/v1/timeline?anchor=YYYY-MM-DD&direction=past|future&types=&member=&cursor=`

- 以 `anchor` 为锚点，向前/向后游标分页；对每类来源执行受限窗口查询后合并排序。
- 排序键：`(occurrence_date DESC|ASC, sort_rank, id)`，保证稳定。
- 权限过滤在 SQL 查询层与结果装配层双重执行；无权限对象的媒体预览不生成。
- 性能预算（PRD §21.9）单页 ≤ 200 条、单来源查询超时与降级阈值在实现期用基准确定。
