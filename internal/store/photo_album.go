package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mymoments/internal/model"
)

const photoColumns = `id, storage_key, original_name, mime_type, size_bytes, fit, created_by, created_at`

func scanPhoto(scanner interface{ Scan(...any) error }) (*model.PhotoAlbum, error) {
	var p model.PhotoAlbum
	var name sql.NullString
	if err := scanner.Scan(&p.ID, &p.StorageKey, &name, &p.MimeType, &p.SizeBytes, &p.Fit, &p.CreatedBy, &p.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	p.OriginalName = ps(name)
	return &p, nil
}

// CreatePhoto 写入照片元数据。
func (s *Store) CreatePhoto(ctx context.Context, p *model.PhotoAlbum) (int64, error) {
	now := time.Now().Unix()
	res, err := s.DB.ExecContext(ctx, `INSERT INTO photo_album
		(storage_key, original_name, mime_type, size_bytes, fit, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.StorageKey, ps2(p.OriginalName), p.MimeType, p.SizeBytes, p.Fit, p.CreatedBy, now)
	if err != nil {
		return 0, err
	}
	p.CreatedAt = now
	return res.LastInsertId()
}

// GetPhoto 按 ID 读取照片元数据。
func (s *Store) GetPhoto(ctx context.Context, id int64) (*model.PhotoAlbum, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+photoColumns+` FROM photo_album WHERE id=?`, id)
	return scanPhoto(row)
}

// ListPhotos 按创建顺序返回全部照片。
func (s *Store) ListPhotos(ctx context.Context) ([]model.PhotoAlbum, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+photoColumns+` FROM photo_album ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.PhotoAlbum, 0)
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// DeletePhoto 删除照片元数据。
func (s *Store) DeletePhoto(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM photo_album WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CountPhotos 统计照片数量。
func (s *Store) CountPhotos(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM photo_album`).Scan(&n)
	return n, err
}
