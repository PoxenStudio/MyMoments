package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mymoments/internal/model"
)

// ErrTaskCompleted 表示待办已完成，不能重复完成。
var ErrTaskCompleted = errors.New("task already completed")

// ErrNextTaskChanged 表示下一次实例已被修改，撤销时不删除。
var ErrNextTaskChanged = errors.New("next task changed")

// CompleteTask 写入完成记录并生成下一次重复实例。
func (s *Store) CompleteTask(ctx context.Context, task *model.Task, c *model.TaskCompletion, next *model.Task) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		var active int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM task_completions WHERE task_id = ? AND occurrence_date = ? AND undone_at IS NULL`,
			task.ID, c.OccurrenceDate).Scan(&active); err != nil {
			return err
		}
		if active > 0 {
			return ErrTaskCompleted
		}
		now := time.Now().Unix()
		res, err := tx.ExecContext(ctx, `INSERT INTO task_completions
			(task_id, occurrence_date, series_id, done_by_user_id, actual_user_id, completed_at) VALUES (?, ?, ?, ?, ?, ?)`,
			task.ID, c.OccurrenceDate, ps2(c.SeriesID), nid(c.DoneByUserID), nid(c.ActualUserID), now)
		if err != nil {
			return err
		}
		c.ID, _ = res.LastInsertId()
		c.CompletedAt = now
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET status='completed', completed_at=?, updated_at=? WHERE id=?`, now, now, task.ID); err != nil {
			return err
		}
		_, _ = tx.ExecContext(ctx, `DELETE FROM reminders WHERE source_type='task' AND source_id=? AND status='pending' AND (? IS NULL OR occurrence_date = ?)`,
			task.ID, ps2(task.OccurrenceDate), ps2(task.OccurrenceDate))
		if next == nil {
			return nil
		}
		if next.SeriesID == nil {
			next.SeriesID = task.SeriesID
		}
		nextID, err := insertTaskTx(ctx, tx, next, now)
		if err != nil {
			return err
		}
		next.ID = nextID
		if _, err := tx.ExecContext(ctx, `UPDATE task_completions SET next_task_id=? WHERE id=?`, nextID, c.ID); err != nil {
			return err
		}
		return nil
	})
}

// UndoCompletion 撤销完成记录，并尽量撤回本次生成的下一次实例。
func (s *Store) UndoCompletion(ctx context.Context, task *model.Task, completionID int64) (bool, error) {
	removed := false
	err := s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		var nextID sql.NullInt64
		if err := tx.QueryRowContext(ctx, `SELECT next_task_id FROM task_completions WHERE id=? AND task_id=?`,
			completionID, task.ID).Scan(&nextID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE task_completions SET undone_at=? WHERE id=?`, now, completionID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET status='todo', completed_at=NULL, updated_at=? WHERE id=?`, now, task.ID); err != nil {
			return err
		}
		if !nextID.Valid {
			return nil
		}
		ok, err := nextTaskRemovable(ctx, tx, nextID.Int64)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE id=?`, nextID.Int64); err != nil {
			return err
		}
		removed = true
		return nil
	})
	return removed, err
}

func nextTaskRemovable(ctx context.Context, tx *sql.Tx, id int64) (bool, error) {
	var status string
	var created, updated int64
	var media, doneSubs, completions int
	if err := tx.QueryRowContext(ctx, `SELECT status, created_at, updated_at FROM tasks WHERE id=?`, id).Scan(&status, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if status == "completed" || created != updated {
		return false, nil
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM task_media WHERE task_id=?`, id).Scan(&media); err != nil {
		return false, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM task_subtasks WHERE task_id=? AND done=1`, id).Scan(&doneSubs); err != nil {
		return false, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM task_completions WHERE task_id=?`, id).Scan(&completions); err != nil {
		return false, err
	}
	return media == 0 && doneSubs == 0 && completions == 0, nil
}

func insertTaskTx(ctx context.Context, tx *sql.Tx, t *model.Task, now int64) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO tasks
		(title, description, status, priority, start_date, due_date, due_time, template_id, recurrence_rule,
		series_id, occurrence_date, color, icon, visibility, reminder_offsets, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Title, ps2(t.Description), t.Status, t.Priority, ps2(t.StartDate), ps2(t.DueDate), ps2(t.DueTime),
		nid(t.TemplateID), ps2(t.RecurrenceRule), ps2(t.SeriesID), ps2(t.OccurrenceDate), ps2(t.Color), ps2(t.Icon),
		t.Visibility, offsetsJSON(t.ReminderOffsets), t.CreatedBy, now, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, uid := range assigneeIDs(t.Assignees) {
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO task_assignees (task_id, user_id) VALUES (?, ?)`, id, uid); err != nil {
			return 0, err
		}
	}
	for i, st := range t.Subtasks {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_subtasks (task_id, title, done, position, updated_at) VALUES (?, ?, 0, ?, ?)`,
			id, st.Title, i, now); err != nil {
			return 0, err
		}
	}
	return id, nil
}
