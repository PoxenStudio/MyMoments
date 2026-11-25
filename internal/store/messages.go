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

const messageColumns = `m.id, m.body, m.author_id, au.display_name, m.occurrence_date, m.pinned,
	m.visibility, m.edited_at, m.closed_at, m.created_at, m.updated_at`

func scanMessage(scanner interface{ Scan(...any) error }) (*model.MessagePost, error) {
	var m model.MessagePost
	var edited, closed sql.NullInt64
	var occurrenceDate sql.NullString
	var pinned int
	if err := scanner.Scan(&m.ID, &m.Body, &m.AuthorID, &m.AuthorName, &occurrenceDate, &pinned,
		&m.Visibility, &edited, &closed, &m.CreatedAt, &m.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	m.OccurrenceDate = ps(occurrenceDate)
	m.Pinned = pinned != 0
	if edited.Valid {
		m.EditedAt = &edited.Int64
	}
	if closed.Valid {
		m.ClosedAt = &closed.Int64
	}
	m.AudienceIDs = []int64{}
	m.Mentions = []model.MentionTarget{}
	m.Replies = []model.MessageReply{}
	return &m, nil
}

// MessageFilter 描述留言列表筛选。
type MessageFilter struct {
	ViewerID     int64
	Query        string
	MentionsOnly bool
	MineOnly     bool
	PinnedOnly   bool
	Limit        int
	Offset       int
}

// ListMessages 返回当前用户可见的留言，置顶优先、按最近更新排序。
func (s *Store) ListMessages(ctx context.Context, f MessageFilter) ([]model.MessagePost, error) {
	where := []string{`(m.visibility = 'household' OR m.author_id = ? OR (m.visibility = 'selected'
		AND EXISTS (SELECT 1 FROM content_audiences ca
			WHERE ca.content_type = 'message' AND ca.content_id = m.id AND ca.user_id = ?)))`}
	args := []any{f.ViewerID, f.ViewerID}
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, "m.body LIKE ?")
		args = append(args, "%"+q+"%")
	}
	if f.MentionsOnly {
		where = append(where, `EXISTS (SELECT 1 FROM mentions mn
			WHERE mn.content_type = 'message' AND mn.content_id = m.id AND mn.target_type = 'user' AND mn.target_id = ?)`)
		args = append(args, f.ViewerID)
	}
	if f.MineOnly {
		where = append(where, "m.author_id = ?")
		args = append(args, f.ViewerID)
	}
	if f.PinnedOnly {
		where = append(where, "m.pinned = 1")
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
	rows, err := s.DB.QueryContext(ctx, `SELECT `+messageColumns+`
		FROM message_posts m JOIN users au ON au.id = m.author_id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY m.pinned DESC, m.updated_at DESC, m.id DESC
		LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.MessagePost, 0)
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := s.loadMessageDetail(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// GetMessage 读取单条留言详情。
func (s *Store) GetMessage(ctx context.Context, id int64) (*model.MessagePost, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+messageColumns+`
		FROM message_posts m JOIN users au ON au.id = m.author_id WHERE m.id = ?`, id)
	m, err := scanMessage(row)
	if err != nil {
		return nil, err
	}
	if err := s.loadMessageDetail(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) loadMessageDetail(ctx context.Context, m *model.MessagePost) error {
	replies, err := s.loadReplies(ctx, m.ID)
	if err != nil {
		return err
	}
	m.Replies = replies
	m.ReplyCount = len(replies)
	m.AudienceIDs, err = loadAudiences(ctx, s.DB, "message", m.ID)
	if err != nil {
		return err
	}
	m.Mentions, err = loadMentionTargets(ctx, s.DB, "message", m.ID)
	return err
}

func (s *Store) loadReplies(ctx context.Context, postID int64) ([]model.MessageReply, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT r.id, r.post_id, r.author_id, u.display_name, r.body, r.created_at
		FROM message_replies r JOIN users u ON u.id = r.author_id
		WHERE r.post_id = ? ORDER BY r.created_at, r.id`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.MessageReply, 0)
	for rows.Next() {
		var r model.MessageReply
		if err := rows.Scan(&r.ID, &r.PostID, &r.AuthorID, &r.AuthorName, &r.Body, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) displayName(ctx context.Context, q queryer, userID int64) (string, error) {
	var name string
	err := q.QueryRowContext(ctx, `SELECT display_name FROM users WHERE id = ?`, userID).Scan(&name)
	return name, err
}

// CreateMessage 创建留言、受众、提及与通知。
func (s *Store) CreateMessage(ctx context.Context, m *model.MessagePost, mentions []model.MentionTarget) (int64, error) {
	var id int64
	err := s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		res, err := tx.ExecContext(ctx, `INSERT INTO message_posts
			(body, author_id, occurrence_date, pinned, visibility, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			m.Body, m.AuthorID, ps2(m.OccurrenceDate), m.Pinned, m.Visibility, now, now)
		if err != nil {
			return err
		}
		if id, err = res.LastInsertId(); err != nil {
			return err
		}
		if err := replaceAudiences(ctx, tx, "message", id, m.Visibility, m.AudienceIDs); err != nil {
			return err
		}
		if err := replaceMentions(ctx, tx, "message", id, mentions); err != nil {
			return err
		}
		name, err := s.displayName(ctx, tx, m.AuthorID)
		if err != nil {
			return err
		}
		return syncMentions(ctx, tx, mentionSync{
			prefix:     fmt.Sprintf("mention:message:%d:", id),
			sourceType: "message",
			sourceID:   id,
			actorID:    m.AuthorID,
			summary:    name + " 在留言中提到了你",
			userIDs:    mentionRecipients(mentionUserIDs(mentions), m.Visibility, m.AudienceIDs),
		})
	})
	return id, err
}

// UpdateMessage 更新留言内容与可见范围。
func (s *Store) UpdateMessage(ctx context.Context, m *model.MessagePost, mentions []model.MentionTarget) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		if _, err := tx.ExecContext(ctx, `UPDATE message_posts SET body = ?, occurrence_date = ?, pinned = ?,
			visibility = ?, edited_at = ?, updated_at = ? WHERE id = ?`,
			m.Body, ps2(m.OccurrenceDate), m.Pinned, m.Visibility, now, now, m.ID); err != nil {
			return err
		}
		if err := replaceAudiences(ctx, tx, "message", m.ID, m.Visibility, m.AudienceIDs); err != nil {
			return err
		}
		if err := replaceMentions(ctx, tx, "message", m.ID, mentions); err != nil {
			return err
		}
		name, err := s.displayName(ctx, tx, m.AuthorID)
		if err != nil {
			return err
		}
		return syncMentions(ctx, tx, mentionSync{
			prefix:     fmt.Sprintf("mention:message:%d:", m.ID),
			sourceType: "message",
			sourceID:   m.ID,
			actorID:    m.AuthorID,
			summary:    name + " 在留言中提到了你",
			userIDs:    mentionRecipients(mentionUserIDs(mentions), m.Visibility, m.AudienceIDs),
		})
	})
}

// DeleteMessage 删除留言及其回复、提及与相关通知。
func (s *Store) DeleteMessage(ctx context.Context, id int64) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM mentions WHERE content_type = 'message_reply'
			AND content_id IN (SELECT id FROM message_replies WHERE post_id = ?)`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM mentions WHERE content_type = 'message' AND content_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM content_audiences WHERE content_type = 'message' AND content_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM inbox_notifications WHERE source_type = 'message' AND source_id = ?`, id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM message_posts WHERE id = ?`, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// CreateReply 创建回复并生成提及通知。
func (s *Store) CreateReply(ctx context.Context, r *model.MessageReply, mentions []model.MentionTarget, visibility string, audienceIDs []int64) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		res, err := tx.ExecContext(ctx,
			`INSERT INTO message_replies (post_id, author_id, body, created_at) VALUES (?, ?, ?, ?)`,
			r.PostID, r.AuthorID, r.Body, now)
		if err != nil {
			return err
		}
		if r.ID, err = res.LastInsertId(); err != nil {
			return err
		}
		r.CreatedAt = now
		if err := replaceMentions(ctx, tx, "message_reply", r.ID, mentions); err != nil {
			return err
		}
		name, err := s.displayName(ctx, tx, r.AuthorID)
		if err != nil {
			return err
		}
		r.AuthorName = name
		return syncMentions(ctx, tx, mentionSync{
			prefix:     fmt.Sprintf("mention:reply:%d:", r.ID),
			sourceType: "message",
			sourceID:   r.PostID,
			actorID:    r.AuthorID,
			summary:    name + " 在留言回复中提到了你",
			userIDs:    mentionRecipients(mentionUserIDs(mentions), visibility, audienceIDs),
		})
	})
}
