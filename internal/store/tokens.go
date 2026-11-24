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

// ListTokensByUser 返回用户创建的 Token，默认不含已撤销。
func (s *Store) ListTokensByUser(ctx context.Context, userID int64) ([]model.APIToken, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, user_id, name, hash, prefix, scopes, expires_at, last_used_at, created_at, revoked_at
		FROM api_tokens WHERE user_id = ? AND revoked_at IS NULL ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.APIToken, 0)
	for rows.Next() {
		var t model.APIToken
		var expires, lastUsed, revoked sql.NullInt64
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Hash, &t.Prefix, &t.Scopes, &expires, &lastUsed, &t.CreatedAt, &revoked); err != nil {
			return nil, err
		}
		if expires.Valid {
			t.ExpiresAt = &expires.Int64
		}
		if lastUsed.Valid {
			t.LastUsedAt = &lastUsed.Int64
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

// CreateToken 持久化 Token 元数据。
func (s *Store) CreateToken(ctx context.Context, t *model.APIToken) error {
	res, err := s.DB.ExecContext(ctx, `INSERT INTO api_tokens (user_id, name, hash, prefix, scopes, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, t.UserID, t.Name, t.Hash, t.Prefix, t.Scopes, t.ExpiresAt, t.CreatedAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = id
	return nil
}

// RevokeToken 撤销本人 Token。
func (s *Store) RevokeToken(ctx context.Context, userID, id int64) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE api_tokens SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL`,
		time.Now().Unix(), id, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// RecordTokenAudit 记录 Token 访问元数据，不包含请求内容。
func (s *Store) RecordTokenAudit(ctx context.Context, tokenID int64, scope, result, clientNote string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO api_token_audit (token_id, requested_at, scope, result, client_note)
		VALUES (?, ?, ?, ?, ?)`, tokenID, time.Now().Unix(), scope, result, clientNote)
	return err
}

// TouchToken 更新最后使用时间，最多每五分钟一次。
func (s *Store) TouchToken(ctx context.Context, id int64) error {
	now := time.Now().Unix()
	_, err := s.DB.ExecContext(ctx, `UPDATE api_tokens SET last_used_at = ?
		WHERE id = ? AND (last_used_at IS NULL OR last_used_at < ?)`, now, id, now-300)
	return err
}
