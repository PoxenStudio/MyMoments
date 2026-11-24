package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mymoments/internal/model"
)

const notificationColumns = `n.id, n.user_id, n.source_type, n.source_id, n.summary, n.actor_id,
	COALESCE(au.display_name, ''), n.occurrence_date, n.created_at, n.read_at, n.invalidated_at`

func scanNotification(scanner interface{ Scan(...any) error }) (*model.Notification, error) {
	var it model.Notification
	var summary, occurrence sql.NullString
	var actor sql.NullInt64
	var readAt, invalidated sql.NullInt64
	if err := scanner.Scan(&it.ID, &it.UserID, &it.SourceType, &it.SourceID, &summary, &actor,
		&it.ActorName, &occurrence, &it.CreatedAt, &readAt, &invalidated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	it.Summary = ps(summary)
	it.OccurrenceDate = ps(occurrence)
	if actor.Valid {
		it.ActorID = &actor.Int64
	}
	if readAt.Valid {
		it.ReadAt = &readAt.Int64
	}
	if invalidated.Valid {
		it.InvalidatedAt = &invalidated.Int64
	}
	return &it, nil
}

// ListNotifications 返回个人收件箱中仍有效的通知。
func (s *Store) ListNotifications(ctx context.Context, userID int64) ([]model.Notification, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+notificationColumns+`
		FROM inbox_notifications n LEFT JOIN users au ON au.id = n.actor_id
		WHERE n.user_id = ? AND n.invalidated_at IS NULL
		ORDER BY n.read_at IS NOT NULL, n.created_at DESC, n.id DESC
		LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Notification, 0)
	for rows.Next() {
		it, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

// MarkNotificationRead 标记单条通知已读。
func (s *Store) MarkNotificationRead(ctx context.Context, userID, id int64) error {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE inbox_notifications SET read_at = ? WHERE id = ? AND user_id = ? AND read_at IS NULL`,
		time.Now().Unix(), id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var exists int
		if err := s.DB.QueryRowContext(ctx,
			`SELECT COUNT(1) FROM inbox_notifications WHERE id = ? AND user_id = ?`, id, userID).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}

// MarkAllNotificationsRead 将个人未读通知全部标记为已读。
func (s *Store) MarkAllNotificationsRead(ctx context.Context, userID int64) (int64, error) {
	res, err := s.DB.ExecContext(ctx, `UPDATE inbox_notifications SET read_at = ?
		WHERE user_id = ? AND read_at IS NULL AND invalidated_at IS NULL`, time.Now().Unix(), userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
