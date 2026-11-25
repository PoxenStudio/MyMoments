-- 提及目标多态化：支持 @ 家庭成员（user）或 @ 联系人（contact）。
-- 家庭成员可视为特殊联系人；@ 表示关联，不是负责人。
CREATE TABLE mentions_new (
  id             INTEGER PRIMARY KEY,
  content_type   TEXT    NOT NULL,
  content_id     INTEGER NOT NULL,
  target_type    TEXT    NOT NULL,
  target_id      INTEGER NOT NULL,
  position_range TEXT,
  created_at     INTEGER NOT NULL
);

INSERT INTO mentions_new (id, content_type, content_id, target_type, target_id, position_range, created_at)
  SELECT id, content_type, content_id, 'user', user_id, position_range, created_at FROM mentions;

DROP TABLE mentions;
ALTER TABLE mentions_new RENAME TO mentions;
CREATE INDEX idx_mentions_content ON mentions (content_type, content_id);
CREATE INDEX idx_mentions_target ON mentions (target_type, target_id);

-- 留言与便签的软关闭标记：当被 @ 的对象全部移除后置为关闭，时间线仍可见。
ALTER TABLE message_posts ADD COLUMN closed_at INTEGER;
ALTER TABLE notes ADD COLUMN closed_at INTEGER;
