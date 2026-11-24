package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"mymoments/internal/model"
)

const templateColumns = `id, name, title, description, icon, color, priority, default_assignee_ids, default_due_time,
	recurrence_rule, reminder_offset, subtask_template, enabled, sort_order, created_by, created_at`

func scanTemplate(scanner interface{ Scan(...any) error }) (*model.TaskTemplate, error) {
	var t model.TaskTemplate
	var desc, icon, color, priority, dueTime, rule, assignees, subtasks sql.NullString
	var offset sql.NullInt64
	if err := scanner.Scan(&t.ID, &t.Name, &t.Title, &desc, &icon, &color, &priority, &assignees, &dueTime, &rule,
		&offset, &subtasks, &t.Enabled, &t.SortOrder, &t.CreatedBy, &t.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t.Description = ps(desc)
	t.Icon = ps(icon)
	t.Color = ps(color)
	t.Priority = ps(priority)
	t.DefaultDueTime = ps(dueTime)
	t.RecurrenceRule = ps(rule)
	if offset.Valid {
		v := int(offset.Int64)
		t.ReminderOffset = &v
	}
	t.DefaultAssigneeIDs = []int64{}
	if assignees.Valid && assignees.String != "" {
		_ = json.Unmarshal([]byte(assignees.String), &t.DefaultAssigneeIDs)
	}
	t.SubtaskTemplate = []string{}
	if subtasks.Valid && subtasks.String != "" {
		_ = json.Unmarshal([]byte(subtasks.String), &t.SubtaskTemplate)
	}
	return &t, nil
}

// ListTemplates 返回全部模板。
func (s *Store) ListTemplates(ctx context.Context) ([]model.TaskTemplate, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+templateColumns+` FROM task_templates ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.TaskTemplate, 0)
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// GetTemplate 按 ID 读取模板。
func (s *Store) GetTemplate(ctx context.Context, id int64) (*model.TaskTemplate, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+templateColumns+` FROM task_templates WHERE id=?`, id)
	return scanTemplate(row)
}

// CreateTemplate 创建模板。
func (s *Store) CreateTemplate(ctx context.Context, t *model.TaskTemplate) (int64, error) {
	now := time.Now().Unix()
	assignees, _ := json.Marshal(t.DefaultAssigneeIDs)
	subtasks, _ := json.Marshal(t.SubtaskTemplate)
	offset := any(nil)
	if t.ReminderOffset != nil {
		offset = *t.ReminderOffset
	}
	res, err := s.DB.ExecContext(ctx, `INSERT INTO task_templates
		(name, title, description, icon, color, priority, default_assignee_ids, default_due_time, recurrence_rule,
		reminder_offset, subtask_template, enabled, sort_order, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Name, t.Title, ps2(t.Description), ps2(t.Icon), ps2(t.Color), ps2(t.Priority), string(assignees),
		ps2(t.DefaultDueTime), ps2(t.RecurrenceRule), offset, string(subtasks), t.Enabled, t.SortOrder, t.CreatedBy, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateTemplate 更新模板。
func (s *Store) UpdateTemplate(ctx context.Context, t *model.TaskTemplate) error {
	assignees, _ := json.Marshal(t.DefaultAssigneeIDs)
	subtasks, _ := json.Marshal(t.SubtaskTemplate)
	offset := any(nil)
	if t.ReminderOffset != nil {
		offset = *t.ReminderOffset
	}
	res, err := s.DB.ExecContext(ctx, `UPDATE task_templates SET name=?, title=?, description=?, icon=?, color=?, priority=?,
		default_assignee_ids=?, default_due_time=?, recurrence_rule=?, reminder_offset=?, subtask_template=?, enabled=?, sort_order=?
		WHERE id=?`,
		t.Name, t.Title, ps2(t.Description), ps2(t.Icon), ps2(t.Color), ps2(t.Priority), string(assignees),
		ps2(t.DefaultDueTime), ps2(t.RecurrenceRule), offset, string(subtasks), t.Enabled, t.SortOrder, t.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteTemplate 删除模板，既有待办保留。
func (s *Store) DeleteTemplate(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM task_templates WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
