-- 初始架构：家时光（MyMoments）完整基线。
-- 项目尚未正式部署，历史增量迁移已合并为此文件。

CREATE TABLE household_settings (
  id                      INTEGER PRIMARY KEY,
  name                    TEXT    NOT NULL DEFAULT '我的家',
  timezone                TEXT    NOT NULL DEFAULT 'Asia/Shanghai',
  week_start              INTEGER NOT NULL DEFAULT 1,
  default_weather_city    TEXT,
  default_weather_lat     REAL,
  default_weather_lon     REAL,
  default_reminder_offset INTEGER NOT NULL DEFAULT 0,
  default_dashboard       TEXT,
  style                   TEXT,
  date_format             TEXT    NOT NULL DEFAULT '24h',
  created_at              INTEGER NOT NULL,
  updated_at              INTEGER NOT NULL
);

CREATE TABLE users (
  id                   INTEGER PRIMARY KEY,
  username             TEXT    NOT NULL UNIQUE,
  display_name         TEXT    NOT NULL,
  password_hash        TEXT    NOT NULL,
  avatar_key           TEXT,
  role                 TEXT    NOT NULL DEFAULT 'member',
  status               TEXT    NOT NULL DEFAULT 'active',
  must_change_password INTEGER NOT NULL DEFAULT 0,
  relation             TEXT,
  sort_order           INTEGER NOT NULL DEFAULT 0,
  created_at           INTEGER NOT NULL,
  updated_at           INTEGER NOT NULL
);

CREATE TABLE user_preferences (
  user_id            INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  locale             TEXT    NOT NULL DEFAULT 'zh-CN',
  date_format        TEXT    NOT NULL DEFAULT '24h',
  dashboard          TEXT,
  weather_city       TEXT,
  weather_lat        REAL,
  weather_lon        REAL,
  weather_auto       INTEGER NOT NULL DEFAULT 0,
  notification_prefs TEXT,
  style              TEXT,
  updated_at         INTEGER NOT NULL
);

CREATE TABLE sessions (
  id           TEXT PRIMARY KEY,
  user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at   INTEGER NOT NULL,
  created_at   INTEGER NOT NULL,
  last_seen_at INTEGER NOT NULL,
  user_agent   TEXT,
  ip           TEXT
);
CREATE INDEX idx_sessions_user ON sessions (user_id);
CREATE INDEX idx_sessions_expires ON sessions (expires_at);

CREATE TABLE task_templates (
  id                   INTEGER PRIMARY KEY,
  name                 TEXT    NOT NULL,
  title                TEXT    NOT NULL DEFAULT '',
  description          TEXT,
  fields               TEXT    NOT NULL DEFAULT '{}',
  icon                 TEXT,
  color                TEXT,
  priority             TEXT,
  default_assignee_ids TEXT,
  default_due_time     TEXT,
  recurrence_rule      TEXT,
  reminder_offset      INTEGER,
  subtask_template     TEXT,
  enabled              INTEGER NOT NULL DEFAULT 1,
  sort_order           INTEGER NOT NULL DEFAULT 0,
  created_by           INTEGER NOT NULL REFERENCES users(id),
  created_at           INTEGER NOT NULL
);

CREATE TABLE tasks (
  id               INTEGER PRIMARY KEY,
  title            TEXT    NOT NULL,
  description      TEXT,
  status           TEXT    NOT NULL DEFAULT 'todo',
  priority         TEXT    NOT NULL DEFAULT 'normal',
  start_date       TEXT,
  due_date         TEXT,
  due_time         TEXT,
  template_id      INTEGER REFERENCES task_templates(id) ON DELETE SET NULL,
  recurrence_rule  TEXT,
  series_id        TEXT,
  occurrence_date  TEXT,
  color            TEXT,
  icon             TEXT,
  visibility       TEXT    NOT NULL DEFAULT 'household',
  completed_at     INTEGER,
  archived_at      INTEGER,
  reminder_offsets TEXT,
  created_by       INTEGER NOT NULL REFERENCES users(id),
  created_at       INTEGER NOT NULL,
  updated_at       INTEGER NOT NULL
);
CREATE INDEX idx_tasks_status ON tasks (status, archived_at);
CREATE INDEX idx_tasks_due ON tasks (due_date);
CREATE INDEX idx_tasks_series ON tasks (series_id);

CREATE TABLE task_assignees (
  task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id),
  PRIMARY KEY (task_id, user_id)
);

CREATE TABLE task_subtasks (
  id         INTEGER PRIMARY KEY,
  task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  title      TEXT    NOT NULL,
  done       INTEGER NOT NULL DEFAULT 0,
  position   INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL
);

CREATE TABLE task_completions (
  id              INTEGER PRIMARY KEY,
  task_id         INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  occurrence_date TEXT    NOT NULL,
  series_id       TEXT,
  done_by_user_id INTEGER REFERENCES users(id),
  actual_user_id  INTEGER REFERENCES users(id),
  completed_at    INTEGER NOT NULL,
  undone_at       INTEGER,
  next_task_id    INTEGER
);
CREATE INDEX idx_task_completions_task ON task_completions (task_id, occurrence_date);

CREATE TABLE task_media (
  id              INTEGER PRIMARY KEY,
  task_id         INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  occurrence_date TEXT,
  storage_key     TEXT    NOT NULL,
  original_name   TEXT,
  media_type      TEXT    NOT NULL,
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

CREATE TABLE calendar_events (
  id               INTEGER PRIMARY KEY,
  title            TEXT    NOT NULL,
  description      TEXT,
  start_local      TEXT    NOT NULL,
  end_local        TEXT,
  all_day          INTEGER NOT NULL DEFAULT 0,
  location         TEXT,
  color            TEXT,
  icon             TEXT,
  recurrence_rule  TEXT,
  series_id        TEXT,
  visibility       TEXT    NOT NULL DEFAULT 'household',
  source           TEXT    NOT NULL DEFAULT 'manual',
  source_ref       TEXT,
  reminder_offsets TEXT,
  created_by       INTEGER NOT NULL REFERENCES users(id),
  created_at       INTEGER NOT NULL,
  updated_at       INTEGER NOT NULL
);
CREATE INDEX idx_events_start ON calendar_events (start_local);
CREATE INDEX idx_events_series ON calendar_events (series_id);

CREATE TABLE event_assignees (
  event_id INTEGER NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
  user_id  INTEGER NOT NULL REFERENCES users(id),
  PRIMARY KEY (event_id, user_id)
);

CREATE TABLE recurrence_exceptions (
  id              INTEGER PRIMARY KEY,
  series_id       TEXT    NOT NULL,
  occurrence_date TEXT    NOT NULL,
  action          TEXT    NOT NULL,
  override_fields TEXT,
  UNIQUE (series_id, occurrence_date)
);

CREATE TABLE reminders (
  id              INTEGER PRIMARY KEY,
  user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  source_type     TEXT    NOT NULL,
  source_id       INTEGER NOT NULL,
  occurrence_date TEXT,
  title           TEXT,
  remind_at       INTEGER NOT NULL,
  status          TEXT    NOT NULL DEFAULT 'pending',
  delivered_at    INTEGER,
  idempotency_key TEXT    NOT NULL UNIQUE,
  created_at      INTEGER NOT NULL
);
CREATE INDEX idx_reminders_due ON reminders (remind_at, status);
CREATE INDEX idx_reminders_user ON reminders (user_id, status);

CREATE TABLE contacts (
  id               INTEGER PRIMARY KEY,
  name             TEXT    NOT NULL,
  organization     TEXT,
  title            TEXT,
  note             TEXT,
  birthday_on      TEXT,
  category         TEXT,
  avatar_key       TEXT,
  owner_scope      TEXT    NOT NULL DEFAULT 'household',
  birthday_month   INTEGER,
  birthday_day     INTEGER,
  birth_year       INTEGER,
  show_in_calendar INTEGER NOT NULL DEFAULT 0,
  wechat           TEXT,
  office_phone     TEXT,
  created_by       INTEGER NOT NULL REFERENCES users(id),
  created_at       INTEGER NOT NULL,
  updated_at       INTEGER NOT NULL
);

CREATE TABLE contact_items (
  id               INTEGER PRIMARY KEY,
  contact_id       INTEGER NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  kind             TEXT    NOT NULL,
  label            TEXT,
  value            TEXT    NOT NULL,
  position         INTEGER NOT NULL DEFAULT 0,
  is_primary_call  INTEGER NOT NULL DEFAULT 0,
  is_primary_sms   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE birthdays (
  id                INTEGER PRIMARY KEY,
  name              TEXT    NOT NULL,
  month             INTEGER NOT NULL,
  day               INTEGER NOT NULL,
  year_unknown      INTEGER NOT NULL DEFAULT 0,
  birth_year        INTEGER,
  note              TEXT,
  avatar_key        TEXT,
  contact_id        INTEGER REFERENCES contacts(id) ON DELETE SET NULL,
  user_id           INTEGER REFERENCES users(id) ON DELETE SET NULL,
  reminder_offset   INTEGER,
  calendar_event_id INTEGER REFERENCES calendar_events(id) ON DELETE SET NULL,
  created_by        INTEGER NOT NULL REFERENCES users(id),
  created_at        INTEGER NOT NULL,
  updated_at        INTEGER NOT NULL
);

CREATE TABLE contact_relations (
  contact_id         INTEGER NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  related_contact_id INTEGER NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  relation           TEXT    NOT NULL DEFAULT 'emergency',
  PRIMARY KEY (contact_id, related_contact_id)
);

CREATE TABLE contact_reminders (
  id              INTEGER PRIMARY KEY,
  contact_id      INTEGER NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  kind            TEXT    NOT NULL,
  title           TEXT    NOT NULL DEFAULT '',
  frequency       TEXT    NOT NULL DEFAULT '',
  due_date        TEXT,
  recurrence_rule TEXT,
  color           TEXT,
  enabled         INTEGER NOT NULL DEFAULT 1,
  created_at      INTEGER NOT NULL,
  updated_at      INTEGER NOT NULL
);
CREATE INDEX idx_contact_reminders_contact ON contact_reminders (contact_id);

CREATE TABLE message_posts (
  id              INTEGER PRIMARY KEY,
  body            TEXT    NOT NULL,
  author_id       INTEGER NOT NULL REFERENCES users(id),
  occurrence_date TEXT,
  pinned          INTEGER NOT NULL DEFAULT 0,
  visibility      TEXT    NOT NULL DEFAULT 'household',
  edited_at       INTEGER,
  closed_at       INTEGER,
  created_at      INTEGER NOT NULL,
  updated_at      INTEGER NOT NULL
);
CREATE INDEX idx_messages_created ON message_posts (created_at);

CREATE TABLE message_replies (
  id         INTEGER PRIMARY KEY,
  post_id    INTEGER NOT NULL REFERENCES message_posts(id) ON DELETE CASCADE,
  author_id  INTEGER NOT NULL REFERENCES users(id),
  body       TEXT    NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE TABLE notes (
  id          INTEGER PRIMARY KEY,
  title       TEXT,
  body        TEXT,
  color       TEXT,
  pinned      INTEGER NOT NULL DEFAULT 0,
  category    TEXT,
  author_id   INTEGER NOT NULL REFERENCES users(id),
  memory_date TEXT,
  visibility  TEXT    NOT NULL DEFAULT 'household',
  closed_at   INTEGER,
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL
);
CREATE INDEX idx_notes_updated ON notes (updated_at);

CREATE TABLE note_items (
  id         INTEGER PRIMARY KEY,
  note_id    INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  text       TEXT    NOT NULL,
  checked    INTEGER NOT NULL DEFAULT 0,
  position   INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL
);

CREATE TABLE inbox_notifications (
  id              INTEGER PRIMARY KEY,
  user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  source_type     TEXT    NOT NULL,
  source_id       INTEGER NOT NULL,
  summary         TEXT,
  actor_id        INTEGER REFERENCES users(id),
  occurrence_date TEXT,
  created_at      INTEGER NOT NULL,
  read_at         INTEGER,
  invalidated_at  INTEGER,
  idempotency_key TEXT    NOT NULL UNIQUE
);
CREATE INDEX idx_inbox_user ON inbox_notifications (user_id, read_at);

CREATE TABLE api_tokens (
  id           INTEGER PRIMARY KEY,
  user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name         TEXT    NOT NULL,
  hash         TEXT    NOT NULL UNIQUE,
  prefix       TEXT    NOT NULL,
  scopes       TEXT    NOT NULL,
  expires_at   INTEGER,
  last_used_at INTEGER,
  created_at   INTEGER NOT NULL,
  revoked_at   INTEGER
);

CREATE TABLE api_token_audit (
  id           INTEGER PRIMARY KEY,
  token_id     INTEGER NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
  requested_at INTEGER NOT NULL,
  scope        TEXT    NOT NULL,
  result       TEXT    NOT NULL,
  client_note  TEXT
);

CREATE TABLE idempotency_keys (
  user_id    INTEGER NOT NULL,
  key        TEXT    NOT NULL,
  status     INTEGER NOT NULL,
  body       TEXT    NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (user_id, key)
);

CREATE TABLE search_index_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE content_audiences (
  content_type TEXT    NOT NULL,
  content_id   INTEGER NOT NULL,
  user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  PRIMARY KEY (content_type, content_id, user_id)
);
CREATE INDEX idx_content_audiences_user ON content_audiences (user_id, content_type);

CREATE TABLE mentions (
  id             INTEGER PRIMARY KEY,
  content_type   TEXT    NOT NULL,
  content_id     INTEGER NOT NULL,
  target_type    TEXT    NOT NULL,
  target_id      INTEGER NOT NULL,
  position_range TEXT,
  created_at     INTEGER NOT NULL
);
CREATE INDEX idx_mentions_content ON mentions (content_type, content_id);
CREATE INDEX idx_mentions_target ON mentions (target_type, target_id);
