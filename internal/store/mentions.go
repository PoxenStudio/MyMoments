package store

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"mymoments/internal/model"
)

// mentionSync 描述一次提及通知同步：保留当前 mentions 对应的通知，失效已移除的。
type mentionSync struct {
	prefix     string
	sourceType string
	sourceID   int64
	actorID    int64
	summary    string
	userIDs    []int64
}

// syncMentions 写入提及通知，并将同一内容下已被移除的提及通知标记为失效。
func syncMentions(ctx context.Context, tx *sql.Tx, m mentionSync) error {
	now := time.Now().Unix()
	keep := make(map[int64]bool, len(m.userIDs))
	for _, uid := range m.userIDs {
		if uid == 0 {
			continue
		}
		keep[uid] = true
		if uid == m.actorID {
			continue
		}
		key := m.prefix + strconv.FormatInt(uid, 10)
		if _, err := tx.ExecContext(ctx, `INSERT INTO inbox_notifications
			(user_id, source_type, source_id, summary, actor_id, created_at, idempotency_key)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(idempotency_key) DO UPDATE SET invalidated_at = NULL, summary = excluded.summary`,
			uid, m.sourceType, m.sourceID, m.summary, m.actorID, now, key); err != nil {
			return err
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, user_id FROM inbox_notifications
		WHERE idempotency_key LIKE ? AND invalidated_at IS NULL`, m.prefix+"%")
	if err != nil {
		return err
	}
	defer rows.Close()
	stale := make([]int64, 0)
	for rows.Next() {
		var id, uid int64
		if err := rows.Scan(&id, &uid); err != nil {
			return err
		}
		if !keep[uid] {
			stale = append(stale, id)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range stale {
		if _, err := tx.ExecContext(ctx,
			`UPDATE inbox_notifications SET invalidated_at = ? WHERE id = ?`, now, id); err != nil {
			return err
		}
	}
	return nil
}

// replaceMentions 重写某内容的提及记录（多态目标：user / contact）。
func replaceMentions(ctx context.Context, tx *sql.Tx, contentType string, contentID int64, targets []model.MentionTarget) error {
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM mentions WHERE content_type = ? AND content_id = ?`, contentType, contentID); err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, t := range targets {
		if t.ID == 0 || (t.Type != "user" && t.Type != "contact") {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO mentions (content_type, content_id, target_type, target_id, created_at) VALUES (?, ?, ?, ?, ?)`,
			contentType, contentID, t.Type, t.ID, now); err != nil {
			return err
		}
	}
	return nil
}

// replaceAudiences 重写「指定成员可见」的受众。
func replaceAudiences(ctx context.Context, tx *sql.Tx, contentType string, contentID int64, visibility string, userIDs []int64) error {
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM content_audiences WHERE content_type = ? AND content_id = ?`, contentType, contentID); err != nil {
		return err
	}
	if visibility != "selected" {
		return nil
	}
	for _, uid := range userIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO content_audiences (content_type, content_id, user_id) VALUES (?, ?, ?)`,
			contentType, contentID, uid); err != nil {
			return err
		}
	}
	return nil
}

// loadAudiences 读取指定内容的受众 ID。
func loadAudiences(ctx context.Context, q queryer, contentType string, contentID int64) ([]int64, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT user_id FROM content_audiences WHERE content_type = ? AND content_id = ? ORDER BY user_id`,
		contentType, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]int64, 0)
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		out = append(out, uid)
	}
	return out, rows.Err()
}

// loadMentionTargets 读取指定内容的提及目标（含姓名与头像）。
func loadMentionTargets(ctx context.Context, q queryer, contentType string, contentID int64) ([]model.MentionTarget, error) {
	rows, err := q.QueryContext(ctx, `SELECT m.target_type, m.target_id,
		CASE WHEN m.target_type = 'user' THEN COALESCE(u.display_name, '') ELSE COALESCE(c.name, '') END,
		CASE WHEN m.target_type = 'user' THEN u.avatar_key ELSE c.avatar_key END
		FROM mentions m
		LEFT JOIN users u ON m.target_type = 'user' AND u.id = m.target_id
		LEFT JOIN contacts c ON m.target_type = 'contact' AND c.id = m.target_id
		WHERE m.content_type = ? AND m.content_id = ? ORDER BY m.id`, contentType, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.MentionTarget, 0)
	for rows.Next() {
		var t model.MentionTarget
		var avatar sql.NullString
		if err := rows.Scan(&t.Type, &t.ID, &t.Name, &avatar); err != nil {
			return nil, err
		}
		if avatar.Valid {
			t.AvatarKey = &avatar.String
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// mentionUserIDs 提取多态提及目标中的成员 ID（联系人无账号，不生成通知）。
func mentionUserIDs(targets []model.MentionTarget) []int64 {
	out := make([]int64, 0, len(targets))
	for _, t := range targets {
		if t.Type == "user" {
			out = append(out, t.ID)
		}
	}
	return out
}

// mentionRecipients 按可见范围筛选可接收提及通知的成员。
func mentionRecipients(mentionIDs []int64, visibility string, audienceIDs []int64) []int64 {
	switch visibility {
	case "private":
		return nil
	case "selected":
		allowed := make(map[int64]bool, len(audienceIDs))
		for _, uid := range audienceIDs {
			allowed[uid] = true
		}
		out := make([]int64, 0, len(mentionIDs))
		for _, uid := range mentionIDs {
			if allowed[uid] {
				out = append(out, uid)
			}
		}
		return out
	default:
		return mentionIDs
	}
}

// queryer 抽象 *sql.DB 与 *sql.Tx 的查询能力。
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
