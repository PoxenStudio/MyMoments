package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mymoments/internal/model"
)

const mediaColumns = `id, task_id, occurrence_date, storage_key, original_name, media_type, mime_type, size_bytes,
	duration_ms, created_by, created_at`

func scanMedia(scanner interface{ Scan(...any) error }) (*model.TaskMedia, error) {
	var m model.TaskMedia
	var occ, name sql.NullString
	var duration sql.NullInt64
	if err := scanner.Scan(&m.ID, &m.TaskID, &occ, &m.StorageKey, &name, &m.MediaType, &m.MimeType, &m.SizeBytes,
		&duration, &m.CreatedBy, &m.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	m.OccurrenceDate = ps(occ)
	m.OriginalName = ps(name)
	if duration.Valid {
		m.DurationMs = &duration.Int64
	}
	return &m, nil
}

// CreateMedia 写入媒体元数据。
func (s *Store) CreateMedia(ctx context.Context, m *model.TaskMedia) (int64, error) {
	now := time.Now().Unix()
	res, err := s.DB.ExecContext(ctx, `INSERT INTO task_media
		(task_id, occurrence_date, storage_key, original_name, media_type, mime_type, size_bytes, duration_ms, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.TaskID, ps2(m.OccurrenceDate), m.StorageKey, ps2(m.OriginalName), m.MediaType, m.MimeType, m.SizeBytes,
		nid(m.DurationMs), m.CreatedBy, now)
	if err != nil {
		return 0, err
	}
	m.CreatedAt = now
	return res.LastInsertId()
}

// GetMedia 按 ID 读取媒体元数据。
func (s *Store) GetMedia(ctx context.Context, id int64) (*model.TaskMedia, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+mediaColumns+` FROM task_media WHERE id=?`, id)
	return scanMedia(row)
}

// ListMedia 返回任务某发生项的附件。
func (s *Store) ListMedia(ctx context.Context, taskID int64, occurrenceDate *string) ([]model.TaskMedia, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+mediaColumns+` FROM task_media WHERE task_id=?
		AND (? IS NULL OR occurrence_date=? OR occurrence_date IS NULL) ORDER BY created_at`, taskID, occurrenceDate, occurrenceDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.TaskMedia, 0)
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// DeleteMedia 删除媒体元数据。
func (s *Store) DeleteMedia(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM task_media WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CountMedia 统计任务发生项附件数量。
func (s *Store) CountMedia(ctx context.Context, taskID int64, occurrenceDate *string) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM task_media WHERE task_id=?
		AND (? IS NULL OR occurrence_date=? OR occurrence_date IS NULL)`, taskID, occurrenceDate, occurrenceDate).Scan(&n)
	return n, err
}

// MediaTotalBytes 统计家庭媒体总占用。
func (s *Store) MediaTotalBytes(ctx context.Context) (int64, error) {
	var n int64
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(size_bytes), 0) FROM task_media`).Scan(&n)
	return n, err
}
