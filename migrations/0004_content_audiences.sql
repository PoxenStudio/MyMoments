-- 留言与便签的「指定成员可见」受众表。
-- content_type: message | note
CREATE TABLE IF NOT EXISTS content_audiences (
  content_type TEXT    NOT NULL,
  content_id   INTEGER NOT NULL,
  user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  PRIMARY KEY (content_type, content_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_content_audiences_user ON content_audiences (user_id, content_type);
