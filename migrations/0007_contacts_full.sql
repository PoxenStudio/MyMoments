-- 通讯录扩展：生日（可无年份）、微信号、办公电话、手机号默认标记、关联联系人、问候提醒。
ALTER TABLE contacts ADD COLUMN birthday_month INTEGER;
ALTER TABLE contacts ADD COLUMN birthday_day INTEGER;
ALTER TABLE contacts ADD COLUMN birth_year INTEGER;
ALTER TABLE contacts ADD COLUMN show_in_calendar INTEGER NOT NULL DEFAULT 0;
ALTER TABLE contacts ADD COLUMN wechat TEXT;
ALTER TABLE contacts ADD COLUMN office_phone TEXT;

ALTER TABLE contact_items ADD COLUMN is_primary_call INTEGER NOT NULL DEFAULT 0;
ALTER TABLE contact_items ADD COLUMN is_primary_sms INTEGER NOT NULL DEFAULT 0;

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
