# 06 · 安全设计

## 1. 认证与会话

- **浏览器会话**：服务端会话存 SQLite `sessions` 表，Cookie 名 `mymoments_session`，属性 `HttpOnly; Secure; SameSite=Lax; Path=/`。会话 ID 为 256-bit 随机值。
- **滑动过期**：`last_seen_at` 滚动刷新；绝对过期 `expires_at`（默认 30 天）。
- **口令哈希**：argon2id（`golang.org/x/crypto/argon2`），参数 `time=3, memory=64MB, threads=4, keyLen=32`，随机 16-byte salt；哈希串存库。
- **口令策略**：最小 8 位；首次登录强制改密（`must_change_password`）；管理员重置后旧会话失效并强制改密（PRD §19.1）。
- **登录限流**：5 次/分钟/用户名+IP，失败统一提示，不区分「用户不存在」与「口令错误」。
- **登出/重置**：删除当前会话；「重置密码」使该用户全部会话失效（`sessions` 按 user_id 删除）。

## 2. CSRF 与请求伪造

- 状态变更请求（POST/PATCH/DELETE）校验 `Origin`/`Referer` 与 Host 一致；`SameSite=Lax` 作为纵深防御。
- Bearer Token 请求无 Cookie，天然免疫 CSRF，走 Token 校验路径。
- 拒绝跨站表单；API 不提供 JSONP 或 `callback` 参数。

## 3. 授权强制执行

- 统一授权函数（见 [03-backend-api](03-backend-api.md) §5），在 handler 内、写操作前调用。
- 前端隐藏仅是体验；任何列表/详情/搜索/摘要/通知/API 均后端判定（PRD §4.2.6）。
- `not_found` 与 `permission.denied` 对外统一为 404，不泄露对象存在性。

## 4. API Token

- 明文仅创建时展示一次；库存 `SHA-256` 哈希 + 短前缀。
- 认证仅 `Authorization: Bearer <token>`；禁止 Token 入 URL、查询参数、日志、Skill 示例（PRD §18.1）。
- 有效权限 = `Token.scopes ∩ 账号模块权限 ∩ 对象可见范围`；`delete` 需显式授予。
- 成员停用/删除时级联撤销其全部 Token（`api_tokens` ON DELETE CASCADE 或显式撤销）。
- 审计仅记录元数据，不记录正文/内容/完整查询结果。

## 5. 安全响应头

| 头 | 值 |
|---|---|
| `Content-Security-Policy` | `default-src 'self'; img-src 'self' data: blob:; media-src 'self' blob:; style-src 'self' 'unsafe-inline'; script-src 'self'`（构建产物用严格 CSP，避免 `unsafe-eval`） |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains`（HTTPS 部署时） |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Permissions-Policy` | `geolocation=(self)`（仅天气定位需要） |

## 6. 媒体访问控制

- 媒体字节存 `/data/media/<uuid>`，**不**放公开静态目录（PRD §18）。
- 唯一访问路径 `GET /api/v1/media/{id}`，经会话/Token 鉴权 + 对象可见范围校验；支持 `Range` 流式与正确 `Content-Type`。
- 上传校验扩展名 + MIME + 大小上限；缩略图由服务端生成；禁止路径穿越（storage_key 为服务端生成的 UUID，不采信文件名）。
- 无权限成员无法查看/播放/下载附件（PRD §19.5）。

## 7. 密钥与日志

- 密钥（会话 secret、天气服务商密钥、VAPID 私钥等）经环境变量注入，不写入镜像、默认配置或版本库（PRD §18）。
- 日志脱敏：禁止记录口令、明文 Token、会话 ID、完整报文敏感字段、定位轨迹（PRD §18）。
- 错误响应不返回内部堆栈、SQL 或凭据（PRD §18）。

## 8. 与 CodingStandards 对齐

安全红线（CodingStandards §10）全部适用：错误分层、结构化日志、脱敏、不吞错。首期无 Rust/Protobuf/NATS 相关项，忽略 CodingStandards 中的微服务安全条目。
