-- 生日档案 + 公历/农历日期子表（方案 C）。
-- 项目尚未发布，直接重建 birthdays，并把旧数据搬迁为公历日期。

DELETE FROM reminders
WHERE source_type = 'event'
  AND source_id IN (SELECT id FROM calendar_events WHERE source = 'birthday');
DELETE FROM reminders WHERE source_type = 'birthday';
DELETE FROM calendar_events WHERE source = 'birthday';

CREATE TABLE birthdays_new (
  id         INTEGER PRIMARY KEY,
  name       TEXT    NOT NULL,
  note       TEXT,
  avatar_key TEXT,
  contact_id INTEGER REFERENCES contacts(id) ON DELETE SET NULL,
  user_id    INTEGER REFERENCES users(id) ON DELETE SET NULL,
  created_by INTEGER NOT NULL REFERENCES users(id),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

INSERT INTO birthdays_new (id, name, note, avatar_key, contact_id, user_id, created_by, created_at, updated_at)
SELECT id, name, note, avatar_key, contact_id, user_id, created_by, created_at, updated_at FROM birthdays;

CREATE TABLE birthday_dates_new (
  id                INTEGER PRIMARY KEY,
  birthday_id       INTEGER NOT NULL REFERENCES birthdays_new(id) ON DELETE CASCADE,
  calendar          TEXT    NOT NULL CHECK (calendar IN ('solar', 'lunar')),
  month             INTEGER NOT NULL,
  day               INTEGER NOT NULL,
  year_unknown      INTEGER NOT NULL DEFAULT 0,
  birth_year        INTEGER,
  reminder_offset   INTEGER,
  calendar_event_id INTEGER REFERENCES calendar_events(id) ON DELETE SET NULL,
  created_at        INTEGER NOT NULL,
  updated_at        INTEGER NOT NULL,
  UNIQUE (birthday_id, calendar)
);

INSERT INTO birthday_dates_new
  (birthday_id, calendar, month, day, year_unknown, birth_year, reminder_offset, created_at, updated_at)
SELECT id, 'solar', month, day, year_unknown, birth_year,
       COALESCE(reminder_offset, 0), created_at, updated_at
FROM birthdays;

DROP TABLE birthdays;
ALTER TABLE birthdays_new RENAME TO birthdays;
ALTER TABLE birthday_dates_new RENAME TO birthday_dates;

CREATE INDEX idx_birthday_dates_birthday ON birthday_dates (birthday_id);
