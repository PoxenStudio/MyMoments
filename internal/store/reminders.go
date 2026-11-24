package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"mymoments/internal/model"
)

// ReminderSpec 描述一条待写入提醒。
type ReminderSpec struct {
	UserID   int64
	RemindAt int64
}

// ReplaceTaskReminders 替换待办未投递提醒。
func (s *Store) ReplaceTaskReminders(ctx context.Context, task *model.Task, specs []ReminderSpec) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM reminders WHERE source_type='task' AND source_id=? AND status='pending'`, task.ID); err != nil {
			return err
		}
		now := time.Now().Unix()
		for _, spec := range specs {
			if spec.RemindAt <= now {
				continue
			}
			key := fmt.Sprintf("task:%d:%s:%d:%d", task.ID, strOrEmpty(task.OccurrenceDate), spec.UserID, spec.RemindAt)
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO reminders
				(user_id, source_type, source_id, occurrence_date, title, remind_at, status, idempotency_key, created_at)
				VALUES (?, 'task', ?, ?, ?, ?, 'pending', ?, ?)`,
				spec.UserID, task.ID, ps2(task.OccurrenceDate), task.Title, spec.RemindAt, key, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func strOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// ListReminders 返回用户在指定状态下的提醒。
func (s *Store) ListReminders(ctx context.Context, userID int64, status string) ([]model.Reminder, error) {
	query := `SELECT id, user_id, source_type, source_id, occurrence_date, title, remind_at, status, delivered_at
		FROM reminders WHERE user_id = ?`
	args := []any{userID}
	if status != "" && status != "all" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY remind_at`
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Reminder, 0)
	for rows.Next() {
		var r model.Reminder
		var occ, title sql.NullString
		var delivered sql.NullInt64
		if err := rows.Scan(&r.ID, &r.UserID, &r.SourceType, &r.SourceID, &occ, &title, &r.RemindAt, &r.Status, &delivered); err != nil {
			return nil, err
		}
		r.OccurrenceDate = ps(occ)
		r.Title = ps(title)
		if delivered.Valid {
			r.DeliveredAt = &delivered.Int64
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateReminderStatus 更新本人提醒状态。
func (s *Store) UpdateReminderStatus(ctx context.Context, userID, id int64, status string) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE reminders SET status=?, delivered_at=CASE WHEN ?='delivered' THEN ? ELSE delivered_at END WHERE id=? AND user_id=?`,
		status, status, time.Now().Unix(), id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteTaskReminders 删除某任务发生项的所有提醒。
func (s *Store) DeleteTaskReminders(ctx context.Context, taskID int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM reminders WHERE source_type='task' AND source_id=?`, taskID)
	return err
}
