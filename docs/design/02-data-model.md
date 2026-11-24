# 02 · 数据模型与迁移

实体以 PRD §17 为基线，物理表名与列名在此确定。约定：

- 主键统一 `INTEGER PRIMARY KEY`（SQLite rowid 别名）或 `TEXT` UUID。
- `created_at` / `updated_at` 为 INTEGER（Unix 秒）；date-only 用 `TEXT`（`YYYY-MM-DD`）；local datetime 用 `TEXT`（`YYYY-MM-DD HH:MM:SS`，家庭时区语义）。
- 外键启用（`PRAGMA foreign_keys=ON`），删除策略按产品规则显式声明。
- 成员账号**软删除**（`status`），不硬删，保留「已停用成员」展示名。

## 1. 时间三类语义（PRD §17.1）

| 类型 | 存储 | 示例 | 场景 |
|---|---|---|---|
| date-only | `TEXT 'YYYY-MM-DD'` | `2026-11-24` | 生日月日、事件日期、回忆日期 |
| local datetime | `TEXT 'YYYY-MM-DD HH:MM:SS'` | `2026-11-24 09:30:00` | 事件开始（无时区歧义，家庭时区） |
| timestamp | `INTEGER`（Unix 秒） | `1769...` | 创建/更新时间、通知时间 |

重复计算、生日距离、日历日界、提醒触发统一走 `internal/service/chrono` 包；家庭时区从 `household_settings.timezone` 读取，默认 `Asia/Shanghai`。

## 2. 核心表 DDL

### 2.1 家庭与用户

```sql
CREATE TABLE household_settings (
  id                 INTEGER PRIMARY KEY,
  name               TEXT    NOT NULL DEFAULT '我的家',
  timezone           TEXT    NOT NULL DEFAULT 'Asia/Shanghai',
  week_start         INTEGER NOT NULL DEFAULT 1,          -- 1=周一
  default_weather_city TEXT,
  default_reminder_offset INTEGER NOT NULL DEFAULT 0,     -- 分钟
  default_dashboard  TEXT,                                -- JSON 卡片布局（管理员发布）
  created_at         INTEGER NOT NULL,
  updated_at         INTEGER NOT NULL
);

CREATE TABLE users (
  id                 INTEGER PRIMARY KEY,
  username           TEXT    NOT NULL UNIQUE,             -- 唯一登录名
  display_name       TEXT    NOT NULL,
  password_hash      TEXT    NOT NULL,
  avatar_key         TEXT,
  role               TEXT    NOT NULL DEFAULT 'member',   -- admin | member
  status             TEXT    NOT NULL DEFAULT 'active',   -- active | inactive
  must_change_password INTEGER NOT NULL DEFAULT 0,
  relation           TEXT,                                -- 父母/子女/祖辈/其他（仅展示）
  sort_order         INTEGER NOT NULL DEFAULT 0,
  created_at         INTEGER NOT NULL,
  updated_at         INTEGER NOT NULL
);

CREATE TABLE user_preferences (
  user_id            INTEGER PRIMARY KEY REFERENCES users(id),
  locale             TEXT    NOT NULL DEFAULT 'zh-CN',
  date_format        TEXT    NOT NULL DEFAULT '24h',
  dashboard          TEXT,                                -- JSON 个人布局；NULL=跟随家庭默认
  weather_city       TEXT,
  weather_auto       INTEGER NOT NULL DEFAULT 1,
  notification_prefs TEXT,
  updated_at         INTEGER NOT NULL
);
```

### 2.2 待办与模板

```sql
CREATE TABLE task_templates (
  id             INTEGER PRIMARY KEY,
  name           TEXT    NOT NULL,
  fields         TEXT    NOT NULL,                        -- JSON 默认字段
  icon           TEXT,
  color          TEXT,
  default_assignee_ids TEXT,
  subtask_template TEXT,
  enabled        INTEGER NOT NULL DEFAULT 1,
  sort_order     INTEGER NOT NULL DEFAULT 0,
  created_by     INTEGER NOT NULL REFERENCES users(id),
  created_at     INTEGER NOT NULL
);

CREATE TABLE tasks (
  id             INTEGER PRIMARY KEY,
  title          TEXT    NOT NULL,
  description    TEXT,
  status         TEXT    NOT NULL DEFAULT 'open',         -- open | completed
  priority       TEXT,                                    -- high | normal | low
  start_date     TEXT,                                    -- date-only
  due_date       TEXT,                                    -- date-only
  due_time       TEXT,                                    -- HH:MM，可空
  template_id    INTEGER REFERENCES task_templates(id),
  recurrence_rule TEXT,                                   -- JSON RRULE 子集
  series_id      TEXT,                                    -- 重复系列标识
  occurrence_date TEXT,                                   -- 具体发生项日期（重复任务）
  color          TEXT,
  icon           TEXT,
  visibility     TEXT    NOT NULL DEFAULT 'household',    -- household | selected | private
  created_by     INTEGER NOT NULL REFERENCES users(id),
  archived_at    INTEGER,
  created_at     INTEGER NOT NULL,
  updated_at     INTEGER NOT NULL
);

CREATE TABLE task_assignees (
  task_id         INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  user_id         INTEGER NOT NULL REFERENCES users(id),
  PRIMARY KEY (task_id, user_id)
);

CREATE TABLE task_completions (
  id              INTEGER PRIMARY KEY,
  task_id         INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  occurrence_date TEXT    NOT NULL,
  series_id       TEXT,
  done_by_user_id INTEGER REFERENCES users(id),           -- 勾选者
  actual_user_id  INTEGER REFERENCES users(id),           -- 实际完成者，可空
  completed_at    INTEGER NOT NULL,
  undone_at       INTEGER                                -- 撤销时间；NULL=有效
);

CREATE TABLE task_media (
  id              INTEGER PRIMARY KEY,
  task_id         INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  occurrence_date TEXT,
  storage_key     TEXT    NOT NULL,                       -- /data/media/<uuid>
  media_type      TEXT    NOT NULL,                       -- image | audio | video
  mime_type       TEXT    NOT NULL,
  size_bytes      INTEGER NOT NULL,
  duration_ms     INTEGER,
  created_by      INTEGER NOT NULL REFERENCES users(id),
  created_at      INTEGER NOT NULL
);

CREATE TABLE task_comments (
  id         INTEGER PRIMARY KEY,
  task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  author_id  INTEGER NOT NULL REFERENCES users(id),
  body       TEXT    NOT NULL,
  created_at INTEGER NOT NULL
);
```

### 2.3 行事历与重复

```sql
CREATE TABLE calendar_events (
  id             INTEGER PRIMARY KEY,
  title          TEXT    NOT NULL,
  description    TEXT,
  start_local    TEXT    NOT NULL,                        -- local datetime
  end_local      TEXT,                                    -- local datetime；全天为 NULL
  all_day        INTEGER NOT NULL DEFAULT 0,
  location       TEXT,
  color          TEXT,
  icon           TEXT,
  recurrence_rule TEXT,
  series_id      TEXT,
  visibility     TEXT    NOT NULL DEFAULT 'household',
  created_by     INTEGER NOT NULL REFERENCES users(id),
  source         TEXT    NOT NULL DEFAULT 'manual',       -- manual | birthday
  created_at     INTEGER NOT NULL,
  updated_at     INTEGER NOT NULL
);

CREATE TABLE event_assignees (
  event_id        INTEGER NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
  user_id         INTEGER NOT NULL REFERENCES users(id),
  PRIMARY KEY (event_id, user_id)
);

CREATE TABLE recurrence_exceptions (
  id              INTEGER PRIMARY KEY,
  series_id       TEXT    NOT NULL,
  occurrence_date TEXT    NOT NULL,
  action          TEXT    NOT NULL,                       -- cancel | reschedule | override
  override_fields TEXT,                                   -- JSON 字段覆盖
  UNIQUE (series_id, occurrence_date)
);
```

### 2.4 提醒

```sql
CREATE TABLE reminders (
  id              INTEGER PRIMARY KEY,
  user_id         INTEGER NOT NULL REFERENCES users(id),  -- 接收人
  source_type     TEXT    NOT NULL,                       -- task | event | birthday
  source_id       INTEGER NOT NULL,
  occurrence_date TEXT,                                   -- 发生项日期
  remind_at       INTEGER NOT NULL,                       -- 触发时间（timestamp）
  status          TEXT    NOT NULL DEFAULT 'pending',     -- pending | dismissed | delivered
  delivered_at    INTEGER,
  idempotency_key TEXT    NOT NULL UNIQUE
);
CREATE INDEX idx_reminders_due ON reminders (remind_at, status);
```

### 2.5 通讯录、生日

```sql
CREATE TABLE contacts (
  id           INTEGER PRIMARY KEY,
  name         TEXT    NOT NULL,
  organization TEXT,
  title        TEXT,
  note         TEXT,
  birthday_on  TEXT,                                      -- date-only，可空
  category_id  INTEGER,
  avatar_key   TEXT,
  owner_scope  TEXT    NOT NULL DEFAULT 'household',      -- household | private
  created_by   INTEGER NOT NULL REFERENCES users(id),
  created_at   INTEGER NOT NULL,
  updated_at   INTEGER NOT NULL
);

CREATE TABLE contact_items (                               -- 电话/邮箱/地址多值
  id          INTEGER PRIMARY KEY,
  contact_id  INTEGER NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  kind        TEXT    NOT NULL,                           -- phone | email | address
  label       TEXT,
  value       TEXT    NOT NULL
);

CREATE TABLE birthdays (
  id            INTEGER PRIMARY KEY,
  name          TEXT    NOT NULL,
  month         INTEGER NOT NULL,
  day           INTEGER NOT NULL,
  year_unknown  INTEGER NOT NULL DEFAULT 0,
  birth_year    INTEGER,
  avatar_key    TEXT,
  contact_id    INTEGER REFERENCES contacts(id),
  user_id       INTEGER REFERENCES users(id),
  reminder_offset INTEGER,                                -- 提前提醒天数
  calendar_event_id INTEGER REFERENCES calendar_events(id),
  created_by    INTEGER NOT NULL REFERENCES users(id)
);
```

### 2.6 留言、便签、提及、通知

```sql
CREATE TABLE message_posts (
  id             INTEGER PRIMARY KEY,
  body           TEXT    NOT NULL,
  author_id      INTEGER NOT NULL REFERENCES users(id),
  occurrence_date TEXT,                                   -- 可选「发生日期」补记过去
  pinned         INTEGER NOT NULL DEFAULT 0,
  visibility     TEXT    NOT NULL DEFAULT 'household',
  created_at     INTEGER NOT NULL
);

CREATE TABLE message_replies (
  id          INTEGER PRIMARY KEY,
  post_id     INTEGER NOT NULL REFERENCES message_posts(id) ON DELETE CASCADE,
  author_id   INTEGER NOT NULL REFERENCES users(id),
  body        TEXT    NOT NULL,
  created_at  INTEGER NOT NULL
);

CREATE TABLE notes (
  id             INTEGER PRIMARY KEY,
  title          TEXT,
  body           TEXT,
  color          TEXT,
  pinned         INTEGER NOT NULL DEFAULT 0,
  category_id    INTEGER,
  author_id      INTEGER NOT NULL REFERENCES users(id),
  memory_date    TEXT,                                    -- 可选「回忆日期」
  visibility     TEXT    NOT NULL DEFAULT 'household',
  created_at     INTEGER NOT NULL,
  updated_at     INTEGER NOT NULL
);

CREATE TABLE mentions (
  id             INTEGER PRIMARY KEY,
  content_type   TEXT    NOT NULL,                        -- task_comment | message | note
  content_id     INTEGER NOT NULL,
  user_id        INTEGER NOT NULL REFERENCES users(id),
  position_range TEXT,                                    -- 正文位置/引用范围
  created_at     INTEGER NOT NULL
);

CREATE TABLE inbox_notifications (
  id              INTEGER PRIMARY KEY,
  user_id         INTEGER NOT NULL REFERENCES users(id),
  source_type     TEXT    NOT NULL,
  source_id       INTEGER NOT NULL,
  summary         TEXT,
  created_at      INTEGER NOT NULL,
  read_at         INTEGER,
  invalidated_at  INTEGER,
  idempotency_key TEXT    NOT NULL UNIQUE
);
CREATE INDEX idx_inbox_user ON inbox_notifications (user_id, read_at);
```

### 2.7 会话与 Token

```sql
CREATE TABLE sessions (
  id            TEXT PRIMARY KEY,                         -- 随机 256bit
  user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at    INTEGER NOT NULL,
  created_at    INTEGER NOT NULL,
  last_seen_at  INTEGER NOT NULL,
  user_agent    TEXT,
  ip            TEXT
);

CREATE TABLE api_tokens (
  id            INTEGER PRIMARY KEY,
  user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name          TEXT    NOT NULL,
  hash          TEXT    NOT NULL UNIQUE,                  -- SHA-256(明文)
  prefix        TEXT    NOT NULL,                         -- 展示用短前缀
  scopes        TEXT    NOT NULL,                         -- JSON: {tasks:[read,write],...}
  expires_at    INTEGER,                                  -- NULL=不过期
  last_used_at  INTEGER,
  created_at    INTEGER NOT NULL,
  revoked_at    INTEGER
);

CREATE TABLE api_token_audit (
  id            INTEGER PRIMARY KEY,
  token_id      INTEGER NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
  requested_at  INTEGER NOT NULL,
  scope         TEXT    NOT NULL,
  result        TEXT    NOT NULL,                         -- ok | denied | error
  client_note   TEXT
);
```

## 3. 时光轴（非持久化投影）

`timeline` 不建表，按 PRD §17「非持久化聚合投影」实现。聚合键 `occurrence_key = (source_type, source_id, occurrence_date)`，来源：

| 来源 | occurrence_date 语义 |
|---|---|
| task_completion | 完成时间对应日期 |
| task（未完成） | due_date / start_date；逾期项留在原到期日 |
| task（重复，未来） | 按重复规则在窗口内生成，不落库 |
| calendar_event | 事件发生日期（含重复展开） |
| birthday | 每年生日日 |
| message_post | occurrence_date 或 created_at 日期 |
| note | memory_date 或 created_at 日期 |

游标分页与稳定排序见 [03-backend-api.md](03-backend-api.md)。

## 4. 迁移策略

- 向前迁移，仅追加不修改已发布迁移（CodingStandards §9）。
- 迁移文件 `migrations/NNNN_name.sql`，内嵌进二进制（`embed`），启动时按序执行并记录 `schema_migrations.version`。
- 迁移失败：服务不启动、不报告健康、输出可诊断错误；不覆盖旧数据（PRD §18.3）。
- 破坏性变更需附备份与回滚说明（PRD §18.3：回滚以恢复数据卷为准）。
