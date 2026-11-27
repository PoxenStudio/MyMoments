# 朝夕（MyMoments）AI Skill

朝夕是一个自托管的家庭协作应用，提供待办、行事历、生日、通讯录、留言、便签、时光轴与提醒。这个 Skill 说明如何通过 HTTP API 让 AI Agent 安全地读取或整理家庭信息。

## 连接信息

- Base URL：你的朝夕实例地址，例如 `https://home.example.com/api/v1`
- 认证：`Authorization: Bearer <API_TOKEN>`
- 内容类型：`Content-Type: application/json`
- 时间格式：日期 `YYYY-MM-DD`，日期时间 `YYYY-MM-DD HH:MM:SS`（家庭时区，默认 `Asia/Shanghai`）
- 时间戳：Unix 秒

API Token 在「设置 → AI 与 API」中创建。Token 只代表创建者本人的权限，并且会叠加内容可见范围（家庭共享 / 指定成员 / 私密）。

## 权限范围

Token 按模块授权，模块与操作如下：

| 模块 | 说明 | 可用操作 |
| --- | --- | --- |
| tasks | 待办与子任务 | read / write / delete |
| calendar | 日程 | read / write / delete |
| birthdays | 生日 | read / write / delete |
| contacts | 通讯录 | read / write / delete |
| messages | 留言 | read / write / delete |
| notes | 便签 | read / write / delete |
| family | 家庭成员 | read |
| timeline | 时光轴 | read |
| reminders | 提醒 | read |
| weather | 天气 | read |

没有授予的操作会返回 `403 permission.denied`。删除权限默认不开启。

## 常用接口

### 读取

- `GET /tasks?scope=open&limit=20` 待办（`scope`: `open` / `completed` / `archived` / `all`）
- `GET /tasks/{id}` 待办详情（含子任务、指派人、完成记录）
- `GET /events?from=YYYY-MM-DD&to=YYYY-MM-DD` 日程
- `GET /birthdays` 生日（含 `daysUntil`、`age`）
- `GET /contacts?q=关键词` 通讯录
- `GET /messages?limit=20` 留言
- `GET /notes?limit=20` 便签
- `GET /timeline?direction=past&anchor=YYYY-MM-DD` 时光轴
- `GET /reminders?status=pending` 提醒
- `GET /weather?lat=31.23&lon=121.47&city=上海` 天气
- `GET /search?q=关键词` 跨模块搜索

### 写入

- `POST /tasks` 创建待办
  ```json
  { "title": "交水电费", "dueDate": "2026-11-30", "dueTime": "18:00", "priority": "normal", "visibility": "household" }
  ```
- `PATCH /tasks/{id}` 修改待办
- `POST /tasks/{id}/completion` 完成待办（支持重复待办自动生成下一次）
- `POST /events` 创建日程
  ```json
  { "title": "家庭聚餐", "startLocal": "2026-11-29 18:00:00", "allDay": false, "visibility": "household" }
  ```
- `POST /birthdays` 创建生日：`{ "name": "妈妈", "month": 12, "day": 1, "yearUnknown": true }`
- `POST /contacts` 创建联系人：`{ "name": "张阿姨", "items": [{ "kind": "phone", "value": "138..." }] }`
- `POST /messages` 发布留言：`{ "body": "周末一起吃饭吧", "visibility": "household" }`
- `POST /messages/{id}/replies` 回复留言：`{ "body": "好呀" }`
- `POST /notes` 创建便签：`{ "title": "周末计划", "body": "去公园", "visibility": "household" }`

## 可见范围

- `household`：全家可见
- `selected`：仅创建者与被指定的成员可见，需要 `audienceIds` 或指派人
- `private`：仅自己可见

AI 只能看到 Token 创建者有权访问的内容；写入时默认使用 `household`，涉及私密信息时请先获得用户确认。

## 建议的 Agent 行为

1. 先读取再修改，避免覆盖他人内容。
2. 批量操作前向用户确认，尤其是删除。
3. 不要把 Token 写入日志或发送到第三方。
4. 遇到 `401` 表示 Token 无效或已撤销；`403` 表示权限不足；`409` 表示版本冲突或重复。
5. 读取大量数据时使用 `limit` 与日期范围。

## 示例

```bash
curl -s https://home.example.com/api/v1/tasks?scope=open \
  -H "Authorization: Bearer $MYMOMENTS_TOKEN"

curl -s -X POST https://home.example.com/api/v1/tasks \
  -H "Authorization: Bearer $MYMOMENTS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"买牛奶","dueDate":"2026-11-25"}'
```
