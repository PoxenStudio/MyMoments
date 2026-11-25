package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"mymoments/internal/model"
)

const noteColumns = `n.id, n.title, n.body, n.color, n.pinned, n.category, n.author_id, u.display_name,
	n.memory_date, n.visibility, n.closed_at, n.created_at, n.updated_at`

func scanNote(scanner interface{ Scan(...any) error }) (*model.Note, error) {
	var n model.Note
	var title, body, color, category, memory sql.NullString
	var closed sql.NullInt64
	var pinned int
	if err := scanner.Scan(&n.ID, &title, &body, &color, &pinned, &category, &n.AuthorID, &n.AuthorName,
		&memory, &n.Visibility, &closed, &n.CreatedAt, &n.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	n.Title = ps(title)
	n.Body = ps(body)
	n.Color = ps(color)
	n.Category = ps(category)
	n.MemoryDate = ps(memory)
	if closed.Valid {
		n.ClosedAt = &closed.Int64
	}
	n.Pinned = pinned != 0
	n.AudienceIDs = []int64{}
	n.Mentions = []model.MentionTarget{}
	n.Items = []model.NoteItem{}
	return &n, nil
}

// NoteFilter 描述便签列表筛选。
type NoteFilter struct {
	ViewerID   int64
	Query      string
	Category   string
	PinnedOnly bool
	Limit      int
	Offset     int
}

// ListNotes 返回当前用户可见便签，置顶优先、按最近编辑排序。
func (s *Store) ListNotes(ctx context.Context, f NoteFilter) ([]model.Note, error) {
	where := []string{`(n.visibility = 'household' OR n.author_id = ? OR (n.visibility = 'selected'
		AND EXISTS (SELECT 1 FROM content_audiences ca
			WHERE ca.content_type = 'note' AND ca.content_id = n.id AND ca.user_id = ?)))`}
	args := []any{f.ViewerID, f.ViewerID}
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, "(n.title LIKE ? OR n.body LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	if f.Category != "" {
		where = append(where, "n.category = ?")
		args = append(args, f.Category)
	}
	if f.PinnedOnly {
		where = append(where, "n.pinned = 1")
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := s.DB.QueryContext(ctx, `SELECT `+noteColumns+`
		FROM notes n JOIN users u ON u.id = n.author_id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY n.pinned DESC, n.updated_at DESC, n.id DESC
		LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Note, 0)
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := s.loadNoteDetail(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// GetNote 读取单条便签。
func (s *Store) GetNote(ctx context.Context, id int64) (*model.Note, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+noteColumns+`
		FROM notes n JOIN users u ON u.id = n.author_id WHERE n.id = ?`, id)
	n, err := scanNote(row)
	if err != nil {
		return nil, err
	}
	if err := s.loadNoteDetail(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *Store) loadNoteDetail(ctx context.Context, n *model.Note) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, note_id, text, checked, position
		FROM note_items WHERE note_id = ? ORDER BY position, id`, n.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var it model.NoteItem
		var checked int
		if err := rows.Scan(&it.ID, &it.NoteID, &it.Text, &checked, &it.Position); err != nil {
			return err
		}
		it.Checked = checked != 0
		n.Items = append(n.Items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	n.AudienceIDs, err = loadAudiences(ctx, s.DB, "note", n.ID)
	if err != nil {
		return err
	}
	n.Mentions, err = loadMentionTargets(ctx, s.DB, "note", n.ID)
	return err
}

// CreateNote 创建便签。
func (s *Store) CreateNote(ctx context.Context, n *model.Note, mentions []model.MentionTarget) (int64, error) {
	var id int64
	err := s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		res, err := tx.ExecContext(ctx, `INSERT INTO notes
			(title, body, color, pinned, category, author_id, memory_date, visibility, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ps2(n.Title), ps2(n.Body), ps2(n.Color), n.Pinned, ps2(n.Category), n.AuthorID,
			ps2(n.MemoryDate), n.Visibility, now, now)
		if err != nil {
			return err
		}
		if id, err = res.LastInsertId(); err != nil {
			return err
		}
		if err := insertNoteItems(ctx, tx, id, n.Items); err != nil {
			return err
		}
		if err := replaceAudiences(ctx, tx, "note", id, n.Visibility, n.AudienceIDs); err != nil {
			return err
		}
		if err := replaceMentions(ctx, tx, "note", id, mentions); err != nil {
			return err
		}
		name, err := s.displayName(ctx, tx, n.AuthorID)
		if err != nil {
			return err
		}
		return syncMentions(ctx, tx, mentionSync{
			prefix:     fmt.Sprintf("mention:note:%d:", id),
			sourceType: "note",
			sourceID:   id,
			actorID:    n.AuthorID,
			summary:    name + " 在便签中提到了你",
			userIDs:    mentionRecipients(mentionUserIDs(mentions), n.Visibility, n.AudienceIDs),
		})
	})
	return id, err
}

// UpdateNote 更新便签并替换检查清单。
func (s *Store) UpdateNote(ctx context.Context, n *model.Note, mentions []model.MentionTarget) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		if _, err := tx.ExecContext(ctx, `UPDATE notes SET title = ?, body = ?, color = ?, pinned = ?, category = ?,
			memory_date = ?, visibility = ?, updated_at = ? WHERE id = ?`,
			ps2(n.Title), ps2(n.Body), ps2(n.Color), n.Pinned, ps2(n.Category),
			ps2(n.MemoryDate), n.Visibility, now, n.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM note_items WHERE note_id = ?`, n.ID); err != nil {
			return err
		}
		if err := insertNoteItems(ctx, tx, n.ID, n.Items); err != nil {
			return err
		}
		if err := replaceAudiences(ctx, tx, "note", n.ID, n.Visibility, n.AudienceIDs); err != nil {
			return err
		}
		if err := replaceMentions(ctx, tx, "note", n.ID, mentions); err != nil {
			return err
		}
		name, err := s.displayName(ctx, tx, n.AuthorID)
		if err != nil {
			return err
		}
		return syncMentions(ctx, tx, mentionSync{
			prefix:     fmt.Sprintf("mention:note:%d:", n.ID),
			sourceType: "note",
			sourceID:   n.ID,
			actorID:    n.AuthorID,
			summary:    name + " 在便签中提到了你",
			userIDs:    mentionRecipients(mentionUserIDs(mentions), n.Visibility, n.AudienceIDs),
		})
	})
}

func insertNoteItems(ctx context.Context, tx *sql.Tx, noteID int64, items []model.NoteItem) error {
	now := time.Now().Unix()
	for i, it := range items {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO note_items (note_id, text, checked, position, updated_at) VALUES (?, ?, ?, ?, ?)`,
			noteID, it.Text, it.Checked, i, now); err != nil {
			return err
		}
	}
	return nil
}

// DeleteNote 删除便签及其关联数据。
func (s *Store) DeleteNote(ctx context.Context, id int64) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM mentions WHERE content_type = 'note' AND content_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM content_audiences WHERE content_type = 'note' AND content_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM inbox_notifications WHERE source_type = 'note' AND source_id = ?`, id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM notes WHERE id = ?`, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		return nil
	})
}
