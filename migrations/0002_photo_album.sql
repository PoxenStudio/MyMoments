-- 首页照片墙：存储照片元数据，文件落在 data/photo_album 目录。

CREATE TABLE photo_album (
  id            INTEGER PRIMARY KEY,
  storage_key   TEXT    NOT NULL,
  original_name TEXT,
  mime_type     TEXT    NOT NULL,
  size_bytes    INTEGER NOT NULL,
  fit           TEXT    NOT NULL DEFAULT 'cover',
  created_by    INTEGER NOT NULL REFERENCES users(id),
  created_at    INTEGER NOT NULL
);
CREATE INDEX idx_photo_album_created ON photo_album (created_at);
