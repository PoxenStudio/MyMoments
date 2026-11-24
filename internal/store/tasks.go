package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"mymoments/internal/model"
)

const taskColumns = `t.id, t.title, t.description, t.status, t.priority, t.start_date, t.due_date, t.due_time,
	t.template_id, t.recurrence_rule, t.series_id, t.occurrence_date, t.color, t.icon, t.visibility,
	t.completed_at, t.archived_at, t.created_by, t.created_at, t.updated_at, t.reminder_offsets`

type rowQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func scanTask(scanner interface{ Scan(...any) error }) (*model.Task, error) {
	var t model.Task
	var desc, startDate, dueDate, dueTime, rule, series, occ, color, icon, offsets sql.NullString
	var templateID, completedAt, archivedAt sql.NullInt64
	if err := scanner.Scan(&t.ID, &t.Title, &desc, &t.Status, &t.Priority, &startDate, &dueDate, &dueTime,
		&templateID, &rule, &series, &occ, &color, &icon, &t.Visibility, &completedAt, &archivedAt,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &offsets); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t.Description = ps(desc)
	t.StartDate = ps(startDate)
	t.DueDate = ps(dueDate)
	t.DueTime = ps(dueTime)
	t.RecurrenceRule = ps(rule)
	t.SeriesID = ps(series)
	t.OccurrenceDate = ps(occ)
	t.Color = ps(color)
	t.Icon = ps(icon)
	if templateID.Valid {
		t.TemplateID = &templateID.Int64
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Int64
	}
	if archivedAt.Valid {
		t.ArchivedAt = &archivedAt.Int64
	}
	t.ReminderOffsets = parseOffsets(offsets.String)
	t.Assignees = []model.User{}
	t.Subtasks = []model.Subtask{}
	return &t, nil
}

func parseOffsets(raw string) []int {
	out := []int{}
	if raw == "" {
		return out
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []int{}
	}
	return out
}

// TaskFilter 描述待办列表筛选。
type TaskFilter struct {
	ViewerID   int64
	Scope      string
	AssigneeID int64
	Priority   string
	TemplateID int64
	Query      string
	From       string
	To         string
	Limit      int
	Offset     int
	Order      string
}

func scopeClause(scope string) string {
	switch scope {
	case "completed":
		return "t.archived_at IS NULL AND t.status = 'completed'"
	case "archived":
		return "t.archived_at IS NOT NULL"
	case "all":
		return "1 = 1"
	default:
		return "t.archived_at IS NULL AND t.status <> 'completed'"
	}
}

func (s *Store) visibleTaskClause() string {
	return "(t.visibility = 'household' OR t.created_by = ? OR (t.visibility = 'selected' AND EXISTS (SELECT 1 FROM task_assignees ta WHERE ta.task_id = t.id AND ta.user_id = ?)))"
}

// ListTasks 返回可见待办。
func (s *Store) ListTasks(ctx context.Context, f TaskFilter) ([]model.Task, error) {
	where := []string{scopeClause(f.Scope), s.visibleTaskClause()}
	args := []any{f.ViewerID, f.ViewerID}
	if f.AssigneeID > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM task_assignees ta2 WHERE ta2.task_id = t.id AND ta2.user_id = ?)")
		args = append(args, f.AssigneeID)
	}
	if f.Priority != "" {
		where = append(where, "t.priority = ?")
		args = append(args, f.Priority)
	}
	if f.TemplateID > 0 {
		where = append(where, "t.template_id = ?")
		args = append(args, f.TemplateID)
	}
	if f.From != "" {
		where = append(where, "t.due_date >= ?")
		args = append(args, f.From)
	}
	if f.To != "" {
		where = append(where, "t.due_date <= ?")
		args = append(args, f.To)
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, "(t.title LIKE ? OR t.description LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	order := "CASE WHEN t.due_date IS NULL THEN 1 ELSE 0 END, t.due_date, t.due_time, t.created_at"
	if f.Order == "recent" {
		order = "t.updated_at DESC"
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	args = append(args, limit, f.Offset)
	query := `SELECT ` + taskColumns + ` FROM tasks t WHERE ` + strings.Join(where, " AND ") + ` ORDER BY ` + order + ` LIMIT ? OFFSET ?`
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Task, 0)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := s.loadTaskRelations(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// TaskCounts 是待办分栏计数。
type TaskCounts struct {
	Open      int `json:"open"`
	Completed int `json:"completed"`
	Archived  int `json:"archived"`
	Today     int `json:"today"`
	Overdue   int `json:"overdue"`
}

func (s *Store) CountTasks(ctx context.Context, viewerID int64, today string) (TaskCounts, error) {
	var c TaskCounts
	base := ` SELECT COUNT(1) FROM tasks t WHERE ` + s.visibleTaskClause()
	rows := []struct {
		where string
		args  []any
		dest  *int
	}{
		{scopeClause("open"), nil, &c.Open},
		{scopeClause("completed"), nil, &c.Completed},
		{scopeClause("archived"), nil, &c.Archived},
		{scopeClause("open") + ` AND t.due_date = ?`, []any{today}, &c.Today},
		{scopeClause("open") + ` AND t.due_date IS NOT NULL AND t.due_date < ?`, []any{today}, &c.Overdue},
	}
	for _, r := range rows {
		args := append([]any{viewerID, viewerID}, r.args...)
		if err := s.DB.QueryRowContext(ctx, base+` AND `+r.where, args...).Scan(r.dest); err != nil {
			return c, err
		}
	}
	return c, nil
}

func (s *Store) loadTaskRelations(ctx context.Context, t *model.Task) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+userColumns+` FROM users u
		JOIN task_assignees ta ON ta.user_id = u.id WHERE ta.task_id = ? ORDER BY u.sort_order, u.id`, t.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			rows.Close()
			return err
		}
		t.Assignees = append(t.Assignees, u.Public())
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	subs, err := s.DB.QueryContext(ctx, `SELECT id, task_id, title, done, position FROM task_subtasks WHERE task_id = ? ORDER BY position, id`, t.ID)
	if err != nil {
		return err
	}
	for subs.Next() {
		var st model.Subtask
		if err := subs.Scan(&st.ID, &st.TaskID, &st.Title, &st.Done, &st.Position); err != nil {
			subs.Close()
			return err
		}
		t.Subtasks = append(t.Subtasks, st)
	}
	subs.Close()
	if err := subs.Err(); err != nil {
		return err
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM task_media WHERE task_id = ? AND (? IS NULL OR occurrence_date = ? OR occurrence_date IS NULL)`,
		t.ID, t.OccurrenceDate, t.OccurrenceDate).Scan(&t.MediaCount); err != nil {
		return err
	}
	c, err := s.activeCompletion(ctx, t.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	t.Completion = c
	return nil
}

func (s *Store) activeCompletion(ctx context.Context, taskID int64) (*model.TaskCompletion, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT c.id, c.task_id, c.occurrence_date, c.series_id, c.done_by_user_id,
		c.actual_user_id, c.completed_at, c.undone_at, c.next_task_id,
		COALESCE(d.display_name, ''), COALESCE(a.display_name, '')
		FROM task_completions c
		LEFT JOIN users d ON d.id = c.done_by_user_id
		LEFT JOIN users a ON a.id = c.actual_user_id
		WHERE c.task_id = ? AND c.undone_at IS NULL ORDER BY c.completed_at DESC LIMIT 1`, taskID)
	var c model.TaskCompletion
	var series sql.NullString
	var doneBy, actual, undone, next sql.NullInt64
	if err := row.Scan(&c.ID, &c.TaskID, &c.OccurrenceDate, &series, &doneBy, &actual, &c.CompletedAt, &undone, &next,
		&c.DoneByName, &c.ActualName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.SeriesID = ps(series)
	if doneBy.Valid {
		c.DoneByUserID = &doneBy.Int64
	}
	if actual.Valid {
		c.ActualUserID = &actual.Int64
	}
	if undone.Valid {
		c.UndoneAt = &undone.Int64
	}
	if next.Valid {
		c.NextTaskID = &next.Int64
	}
	return &c, nil
}

// GetTask 读取单个待办及关联数据。
func (s *Store) GetTask(ctx context.Context, id int64) (*model.Task, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks t WHERE t.id = ?`, id)
	t, err := scanTask(row)
	if err != nil {
		return nil, err
	}
	if err := s.loadTaskRelations(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// CreateTask 在事务中创建待办、指派与子任务。
func (s *Store) CreateTask(ctx context.Context, t *model.Task) (int64, error) {
	var id int64
	err := s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		res, err := tx.ExecContext(ctx, `INSERT INTO tasks
			(title, description, status, priority, start_date, due_date, due_time, template_id, recurrence_rule,
			series_id, occurrence_date, color, icon, visibility, reminder_offsets, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.Title, ps2(t.Description), t.Status, t.Priority, ps2(t.StartDate), ps2(t.DueDate), ps2(t.DueTime),
			nid(t.TemplateID), ps2(t.RecurrenceRule), ps2(t.SeriesID), ps2(t.OccurrenceDate), ps2(t.Color), ps2(t.Icon),
			t.Visibility, offsetsJSON(t.ReminderOffsets), t.CreatedBy, now, now)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return err
		}
		for _, uid := range assigneeIDs(t.Assignees) {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO task_assignees (task_id, user_id) VALUES (?, ?)`, id, uid); err != nil {
				return err
			}
		}
		for i, st := range t.Subtasks {
			if _, err := tx.ExecContext(ctx, `INSERT INTO task_subtasks (task_id, title, done, position, updated_at) VALUES (?, ?, ?, ?, ?)`,
				id, st.Title, st.Done, i, now); err != nil {
				return err
			}
		}
		return nil
	})
	return id, err
}

// UpdateTask 更新待办及其指派、子任务。
func (s *Store) UpdateTask(ctx context.Context, t *model.Task) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET title=?, description=?, status=?, priority=?, start_date=?,
			due_date=?, due_time=?, template_id=?, recurrence_rule=?, series_id=?, occurrence_date=?, color=?, icon=?,
			visibility=?, completed_at=?, archived_at=?, reminder_offsets=?, updated_at=? WHERE id=?`,
			t.Title, ps2(t.Description), t.Status, t.Priority, ps2(t.StartDate), ps2(t.DueDate), ps2(t.DueTime),
			nid(t.TemplateID), ps2(t.RecurrenceRule), ps2(t.SeriesID), ps2(t.OccurrenceDate), ps2(t.Color), ps2(t.Icon),
			t.Visibility, nid(t.CompletedAt), nid(t.ArchivedAt), offsetsJSON(t.ReminderOffsets), now, t.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM task_assignees WHERE task_id = ?`, t.ID); err != nil {
			return err
		}
		for _, uid := range assigneeIDs(t.Assignees) {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO task_assignees (task_id, user_id) VALUES (?, ?)`, t.ID, uid); err != nil {
				return err
			}
		}
		existing := map[int64]bool{}
		for i, st := range t.Subtasks {
			done := 0
			if st.Done {
				done = 1
			}
			if st.ID > 0 {
				existing[st.ID] = true
				if _, err := tx.ExecContext(ctx, `UPDATE task_subtasks SET title=?, done=?, position=?, updated_at=? WHERE id=? AND task_id=?`,
					st.Title, done, i, now, st.ID, t.ID); err != nil {
					return err
				}
				continue
			}
			res, err := tx.ExecContext(ctx, `INSERT INTO task_subtasks (task_id, title, done, position, updated_at) VALUES (?, ?, ?, ?, ?)`,
				t.ID, st.Title, done, i, now)
			if err != nil {
				return err
			}
			newID, err := res.LastInsertId()
			if err != nil {
				return err
			}
			existing[newID] = true
		}
		rows, err := tx.QueryContext(ctx, `SELECT id FROM task_subtasks WHERE task_id = ?`, t.ID)
		if err != nil {
			return err
		}
		remove := []int64{}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			if !existing[id] {
				remove = append(remove, id)
			}
		}
		rows.Close()
		for _, id := range remove {
			if _, err := tx.ExecContext(ctx, `DELETE FROM task_subtasks WHERE id = ?`, id); err != nil {
				return err
			}
		}
		return nil
	})
}

// SetSubtaskDone 更新单个子任务状态。
func (s *Store) SetSubtaskDone(ctx context.Context, taskID, subtaskID int64, done bool) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE task_subtasks SET done=?, updated_at=? WHERE id=? AND task_id=?`,
		done, time.Now().Unix(), subtaskID, taskID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteTask 删除待办及其关联。
func (s *Store) DeleteTask(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func assigneeIDs(users []model.User) []int64 {
	out := make([]int64, 0, len(users))
	for _, u := range users {
		out = append(out, u.ID)
	}
	return out
}

func offsetsJSON(offsets []int) string {
	if len(offsets) == 0 {
		return ""
	}
	b, _ := json.Marshal(offsets)
	return string(b)
}

func ps(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func ps2(v *string) any {
	if v == nil || *v == "" {
		return nil
	}
	return *v
}

func nid(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}
