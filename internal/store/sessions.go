package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mymoments/internal/model"
)

// CreateSession 写入服务端会话。
func (s *Store) CreateSession(ctx context.Context, session *model.Session) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO sessions (id, user_id, expires_at, created_at, last_seen_at, user_agent, ip)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, session.ID, session.UserID, session.ExpiresAt, session.CreatedAt, session.LastSeenAt, session.UserAgent, session.IP)
	return err
}

// GetSession 读取未过期会话。
func (s *Store) GetSession(ctx context.Context, id string) (*model.Session, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, user_id, expires_at, created_at, last_seen_at FROM sessions WHERE id = ?`, id)
	var sess model.Session
	if err := row.Scan(&sess.ID, &sess.UserID, &sess.ExpiresAt, &sess.CreatedAt, &sess.LastSeenAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if sess.ExpiresAt < time.Now().Unix() {
		_, _ = s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
		return nil, ErrNotFound
	}
	return &sess, nil
}

// TouchSession 低频刷新最后活跃时间。
func (s *Store) TouchSession(ctx context.Context, id string, now int64) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE id = ?`, now, id)
	return err
}

// DeleteSession 删除单个会话。
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteUserSessionsExcept 删除某用户的其他会话。
func (s *Store) DeleteUserSessionsExcept(ctx context.Context, userID int64, keep string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ? AND id <> ?`, userID, keep)
	return err
}

// DeleteUserSessions 删除某用户全部会话。
func (s *Store) DeleteUserSessions(ctx context.Context, userID int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

// CleanupExpiredSessions 清理过期记录。
func (s *Store) CleanupExpiredSessions(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, time.Now().Unix())
	return err
}
