package store

import (
	"context"
	"database/sql"

	"mymoments/internal/model"
)

// TimelinePost 是留言/回复在时光轴中的只读投影。
type TimelinePost struct {
	ID             int64
	PostID         int64
	AuthorID       int64
	AuthorName     string
	Body           string
	OccurrenceDate *string
	CreatedAt      int64
}

// TimelineNote 是便签在时光轴中的只读投影。
type TimelineNote struct {
	ID         int64
	AuthorID   int64
	AuthorName string
	Title      *string
	Body       *string
	MemoryDate *string
	CreatedAt  int64
}

// TimelineCompletion 是待办完成记录在时光轴中的只读投影。
type TimelineCompletion struct {
	CompletionID   int64
	OccurrenceDate string
	CompletedAt    int64
	DoneByName     string
	ActualUserID   *int64
	Task           model.Task
}

const visibleMessageClause = `(m.visibility = 'household' OR m.author_id = ? OR (m.visibility = 'selected'
	AND EXISTS (SELECT 1 FROM content_audiences ca
		WHERE ca.content_type = 'message' AND ca.content_id = m.id AND ca.user_id = ?)))`

// ListTimelineMessages 返回可见留言（不含回复）。
func (s *Store) ListTimelineMessages(ctx context.Context, viewerID int64) ([]TimelinePost, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT m.id, m.id, m.author_id, u.display_name, m.body, m.occurrence_date, m.created_at
		FROM message_posts m JOIN users u ON u.id = m.author_id
		WHERE `+visibleMessageClause+`
		ORDER BY m.created_at DESC LIMIT 1000`, viewerID, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TimelinePost, 0)
	for rows.Next() {
		var p TimelinePost
		var occurrence sql.NullString
		if err := rows.Scan(&p.ID, &p.PostID, &p.AuthorID, &p.AuthorName, &p.Body, &occurrence, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.OccurrenceDate = ps(occurrence)
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListTimelineReplies 返回可见留言下的回复。
func (s *Store) ListTimelineReplies(ctx context.Context, viewerID int64) ([]TimelinePost, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT r.id, r.post_id, r.author_id, u.display_name, r.body, r.created_at
		FROM message_replies r
		JOIN users u ON u.id = r.author_id
		JOIN message_posts m ON m.id = r.post_id
		WHERE `+visibleMessageClause+`
		ORDER BY r.created_at DESC LIMIT 1000`, viewerID, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TimelinePost, 0)
	for rows.Next() {
		var p TimelinePost
		if err := rows.Scan(&p.ID, &p.PostID, &p.AuthorID, &p.AuthorName, &p.Body, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListTimelineNotes 返回可见便签。
func (s *Store) ListTimelineNotes(ctx context.Context, viewerID int64) ([]TimelineNote, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT n.id, n.author_id, u.display_name, n.title, n.body, n.memory_date, n.created_at
		FROM notes n JOIN users u ON u.id = n.author_id
		WHERE (n.visibility = 'household' OR n.author_id = ? OR (n.visibility = 'selected'
			AND EXISTS (SELECT 1 FROM content_audiences ca
				WHERE ca.content_type = 'note' AND ca.content_id = n.id AND ca.user_id = ?)))
		ORDER BY n.created_at DESC LIMIT 1000`, viewerID, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TimelineNote, 0)
	for rows.Next() {
		var n TimelineNote
		var title, body, memory sql.NullString
		if err := rows.Scan(&n.ID, &n.AuthorID, &n.AuthorName, &title, &body, &memory, &n.CreatedAt); err != nil {
			return nil, err
		}
		n.Title = ps(title)
		n.Body = ps(body)
		n.MemoryDate = ps(memory)
		out = append(out, n)
	}
	return out, rows.Err()
}

// ListTimelineCompletions 返回时间窗内有效的待办完成记录。
func (s *Store) ListTimelineCompletions(ctx context.Context, viewerID, fromTS, toTS int64) ([]TimelineCompletion, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT c.id, c.occurrence_date, c.completed_at,
			COALESCE(du.display_name, ''), c.actual_user_id,
			t.id, t.title, t.description, t.status, t.priority, t.start_date, t.due_date, t.due_time,
			t.template_id, t.recurrence_rule, t.series_id, t.occurrence_date, t.color, t.icon,
			t.visibility, t.created_by, t.archived_at, t.created_at, t.updated_at
		FROM task_completions c
		JOIN tasks t ON t.id = c.task_id
		LEFT JOIN users du ON du.id = c.done_by_user_id
		WHERE c.undone_at IS NULL AND c.completed_at >= ? AND c.completed_at <= ?
			AND `+s.visibleTaskClause()+`
		ORDER BY c.completed_at DESC, c.id DESC LIMIT 500`, fromTS, toTS, viewerID, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TimelineCompletion, 0)
	for rows.Next() {
		var tc TimelineCompletion
		var actual, templateID, archived sql.NullInt64
		var desc, startDate, dueDate, dueTime, rule, series, occ, color, icon sql.NullString
		if err := rows.Scan(&tc.CompletionID, &tc.OccurrenceDate, &tc.CompletedAt, &tc.DoneByName, &actual,
			&tc.Task.ID, &tc.Task.Title, &desc, &tc.Task.Status, &tc.Task.Priority,
			&startDate, &dueDate, &dueTime, &templateID, &rule, &series, &occ, &color, &icon,
			&tc.Task.Visibility, &tc.Task.CreatedBy, &archived, &tc.Task.CreatedAt, &tc.Task.UpdatedAt); err != nil {
			return nil, err
		}
		tc.Task.Description = ps(desc)
		if actual.Valid {
			tc.ActualUserID = &actual.Int64
		}
		if templateID.Valid {
			tc.Task.TemplateID = &templateID.Int64
		}
		if archived.Valid {
			tc.Task.ArchivedAt = &archived.Int64
		}
		tc.Task.CompletedAt = &tc.CompletedAt
		tc.Task.StartDate = ps(startDate)
		tc.Task.DueDate = ps(dueDate)
		tc.Task.DueTime = ps(dueTime)
		tc.Task.RecurrenceRule = ps(rule)
		tc.Task.SeriesID = ps(series)
		tc.Task.OccurrenceDate = ps(occ)
		tc.Task.Color = ps(color)
		tc.Task.Icon = ps(icon)
		out = append(out, tc)
	}
	return out, rows.Err()
}

// TaskAssignees 读取单个待办的指派成员。
func (s *Store) TaskAssignees(ctx context.Context, taskID int64) ([]model.User, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+userColumns+` FROM users u
		JOIN task_assignees ta ON ta.user_id = u.id WHERE ta.task_id = ? ORDER BY u.sort_order, u.id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u.Public())
	}
	return out, rows.Err()
}
