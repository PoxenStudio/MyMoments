package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mymoments/internal/model"
)

// GetTokenByHash 按哈希读取有效 Token。
func (s *Store) GetTokenByHash(ctx context.Context, hash string) (*model.APIToken, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, user_id, name, hash, prefix, scopes, expires_at, last_used_at, created_at, revoked_at
		FROM api_tokens WHERE hash = ?`, hash)
	var t model.APIToken
	var expires, lastUsed, revoked sql.NullInt64
	if err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.Hash, &t.Prefix, &t.Scopes, &expires, &lastUsed, &t.CreatedAt, &revoked); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if revoked.Valid || (expires.Valid && expires.Int64 < time.Now().Unix()) {
		return nil, ErrNotFound
	}
	if expires.Valid {
		t.ExpiresAt = &expires.Int64
	}
	if lastUsed.Valid {
		t.LastUsedAt = &lastUsed.Int64
	}
	if revoked.Valid {
		t.RevokedAt = &revoked.Int64
	}
	return &t, nil
}

// TouchToken 更新最后使用时间，每五分钟一次。
func (s *Store) TouchToken(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, time.Now().Unix(), id)
	return err
}
