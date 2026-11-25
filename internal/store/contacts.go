package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"mymoments/internal/model"
)

const contactColumns = `c.id, c.name, c.organization, c.title, c.note, c.category, c.avatar_key,
	c.birthday_month, c.birthday_day, c.birth_year, c.show_in_calendar, c.wechat, c.office_phone,
	c.owner_scope, c.created_by, c.created_at, c.updated_at, c.pinned`

func scanContact(scanner interface{ Scan(...any) error }) (*model.Contact, error) {
	var c model.Contact
	var org, title, note, category, avatar, wechat, office sql.NullString
	var bMonth, bDay, bYear sql.NullInt64
	var pinned int
	if err := scanner.Scan(&c.ID, &c.Name, &org, &title, &note, &category, &avatar,
		&bMonth, &bDay, &bYear, &c.ShowInCalendar, &wechat, &office,
		&c.OwnerScope, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &pinned); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.Organization = ps(org)
	c.Title = ps(title)
	c.Note = ps(note)
	c.Category = ps(category)
	c.AvatarKey = ps(avatar)
	c.Wechat = ps(wechat)
	c.OfficePhone = ps(office)
	c.BirthdayMonth = intPtr(bMonth)
	c.BirthdayDay = intPtr(bDay)
	c.BirthYear = intPtr(bYear)
	c.Pinned = pinned != 0
	c.Items = []model.ContactItem{}
	c.Relations = []model.ContactRelation{}
	c.Reminders = []model.ContactReminder{}
	return &c, nil
}

func intPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	i := int(v.Int64)
	return &i
}

// ContactFilter 描述通讯录筛选。
type ContactFilter struct {
	ViewerID int64
	Query    string
	Category string
}

// ListContacts 返回可见联系人。
func (s *Store) ListContacts(ctx context.Context, f ContactFilter) ([]model.Contact, error) {
	where := []string{"(c.owner_scope = 'household' OR c.created_by = ?)"}
	args := []any{f.ViewerID}
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, "(c.name LIKE ? OR c.organization LIKE ? OR c.note LIKE ? OR c.wechat LIKE ? OR c.office_phone LIKE ? OR EXISTS (SELECT 1 FROM contact_items ci WHERE ci.contact_id = c.id AND ci.value LIKE ?))")
		like := "%" + q + "%"
		args = append(args, like, like, like, like, like, like)
	}
	if f.Category != "" {
		where = append(where, "c.category = ?")
		args = append(args, f.Category)
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+contactColumns+` FROM contacts c WHERE `+
		strings.Join(where, " AND ")+` ORDER BY c.pinned DESC, c.name, c.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Contact, 0)
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := s.loadContactRelations(ctx, &out[i]); err != nil {
			return nil, err
		}
		if err := s.loadContactItems(ctx, &out[i]); err != nil {
			return nil, err
		}
		if err := s.loadContactReminders(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// GetContact 读取单个联系人。
func (s *Store) GetContact(ctx context.Context, id int64) (*model.Contact, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+contactColumns+` FROM contacts c WHERE c.id = ?`, id)
	c, err := scanContact(row)
	if err != nil {
		return nil, err
	}
	if err := s.loadContactRelations(ctx, c); err != nil {
		return nil, err
	}
	if err := s.loadContactItems(ctx, c); err != nil {
		return nil, err
	}
	if err := s.loadContactReminders(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) loadContactItems(ctx context.Context, c *model.Contact) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, contact_id, kind, label, value, position, is_primary_call, is_primary_sms
		FROM contact_items WHERE contact_id = ? ORDER BY position, id`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var it model.ContactItem
		var label sql.NullString
		if err := rows.Scan(&it.ID, &it.ContactID, &it.Kind, &label, &it.Value, &it.Position, &it.IsPrimaryCall, &it.IsPrimarySms); err != nil {
			return err
		}
		it.Label = ps(label)
		c.Items = append(c.Items, it)
	}
	return rows.Err()
}

func (s *Store) loadContactRelations(ctx context.Context, c *model.Contact) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT r.contact_id, r.related_contact_id, r.relation, COALESCE(cc.name, '')
		FROM contact_relations r LEFT JOIN contacts cc ON cc.id = r.related_contact_id
		WHERE r.contact_id = ? ORDER BY cc.name, r.related_contact_id`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var r model.ContactRelation
		if err := rows.Scan(&r.ContactID, &r.RelatedContactID, &r.Relation, &r.RelatedName); err != nil {
			return err
		}
		c.Relations = append(c.Relations, r)
	}
	return rows.Err()
}

func (s *Store) loadContactReminders(ctx context.Context, c *model.Contact) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, contact_id, kind, title, frequency, due_date, recurrence_rule, color, enabled, created_at, updated_at
		FROM contact_reminders WHERE contact_id = ? ORDER BY id`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var r model.ContactReminder
		var due, rule, color sql.NullString
		if err := rows.Scan(&r.ID, &r.ContactID, &r.Kind, &r.Title, &r.Frequency, &due, &rule, &color, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return err
		}
		r.DueDate = ps(due)
		r.RecurrenceRule = ps(rule)
		r.Color = ps(color)
		c.Reminders = append(c.Reminders, r)
	}
	return rows.Err()
}

// CreateContact 创建联系人及多值字段。
func (s *Store) CreateContact(ctx context.Context, c *model.Contact) (int64, error) {
	var id int64
	err := s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		res, err := tx.ExecContext(ctx, `INSERT INTO contacts
			(name, organization, title, note, category, avatar_key, birthday_month, birthday_day, birth_year,
			show_in_calendar, wechat, office_phone, owner_scope, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.Name, ps2(c.Organization), ps2(c.Title), ps2(c.Note), ps2(c.Category), ps2(c.AvatarKey),
			nint(c.BirthdayMonth), nint(c.BirthdayDay), nint(c.BirthYear), boolInt(c.ShowInCalendar),
			ps2(c.Wechat), ps2(c.OfficePhone), c.OwnerScope, c.CreatedBy, now, now)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return err
		}
		if err := insertContactItems(ctx, tx, id, c.Items); err != nil {
			return err
		}
		if err := replaceContactRelations(ctx, tx, id, c.Relations); err != nil {
			return err
		}
		return replaceContactReminders(ctx, tx, id, c.Reminders)
	})
	return id, err
}

// UpdateContact 更新联系人并替换多值字段。
func (s *Store) UpdateContact(ctx context.Context, c *model.Contact) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE contacts SET name=?, organization=?, title=?, note=?, category=?, avatar_key=?,
			birthday_month=?, birthday_day=?, birth_year=?, show_in_calendar=?, wechat=?, office_phone=?, owner_scope=?, updated_at=? WHERE id=?`,
			c.Name, ps2(c.Organization), ps2(c.Title), ps2(c.Note), ps2(c.Category), ps2(c.AvatarKey),
			nint(c.BirthdayMonth), nint(c.BirthdayDay), nint(c.BirthYear), boolInt(c.ShowInCalendar),
			ps2(c.Wechat), ps2(c.OfficePhone), c.OwnerScope, time.Now().Unix(), c.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM contact_items WHERE contact_id = ?`, c.ID); err != nil {
			return err
		}
		if err := insertContactItems(ctx, tx, c.ID, c.Items); err != nil {
			return err
		}
		if err := replaceContactRelations(ctx, tx, c.ID, c.Relations); err != nil {
			return err
		}
		return replaceContactReminders(ctx, tx, c.ID, c.Reminders)
	})
}

// SetContactPinned 更新联系人的置顶状态。
func (s *Store) SetContactPinned(ctx context.Context, id int64, pinned bool) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE contacts SET pinned = ?, updated_at = ? WHERE id = ?`,
		boolInt(pinned), time.Now().Unix(), id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

func nint(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func insertContactItems(ctx context.Context, tx *sql.Tx, contactID int64, items []model.ContactItem) error {
	for i, it := range items {
		if _, err := tx.ExecContext(ctx, `INSERT INTO contact_items (contact_id, kind, label, value, position, is_primary_call, is_primary_sms)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, contactID, it.Kind, ps2(it.Label), it.Value, i, boolInt(it.IsPrimaryCall), boolInt(it.IsPrimarySms)); err != nil {
			return err
		}
	}
	return nil
}

func replaceContactRelations(ctx context.Context, tx *sql.Tx, contactID int64, relations []model.ContactRelation) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM contact_relations WHERE contact_id = ?`, contactID); err != nil {
		return err
	}
	for _, r := range relations {
		if r.RelatedContactID == 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO contact_relations (contact_id, related_contact_id, relation)
			VALUES (?, ?, ?)`, contactID, r.RelatedContactID, r.Relation); err != nil {
			return err
		}
	}
	return nil
}

func replaceContactReminders(ctx context.Context, tx *sql.Tx, contactID int64, reminders []model.ContactReminder) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM contact_reminders WHERE contact_id = ?`, contactID); err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, r := range reminders {
		if _, err := tx.ExecContext(ctx, `INSERT INTO contact_reminders
			(contact_id, kind, title, frequency, due_date, recurrence_rule, color, enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			contactID, r.Kind, r.Title, r.Frequency, ps2(r.DueDate), ps2(r.RecurrenceRule), ps2(r.Color), boolInt(r.Enabled), now, now); err != nil {
			return err
		}
	}
	return nil
}

// DeleteContact 删除联系人：解除其被 @ 的关联并收尾受影响内容，再删除联系人行。
// 被 @ 的待办/留言/便签不物理删除，仅收尾（待办完成、留言/便签关闭），时间线仍可查看。
func (s *Store) DeleteContact(ctx context.Context, id int64) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx,
			`SELECT content_type, content_id FROM mentions WHERE target_type = 'contact' AND target_id = ?`, id)
		if err != nil {
			return err
		}
		type contentRef struct {
			contentType string
			contentID   int64
		}
		affected := make([]contentRef, 0)
		for rows.Next() {
			var a contentRef
			if err := rows.Scan(&a.contentType, &a.contentID); err != nil {
				rows.Close()
				return err
			}
			affected = append(affected, a)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM mentions WHERE target_type = 'contact' AND target_id = ?`, id); err != nil {
			return err
		}
		seen := map[contentRef]bool{}
		for _, a := range affected {
			if seen[a] {
				continue
			}
			seen[a] = true
			if err := finalizeIfNoMentions(ctx, tx, a.contentType, a.contentID); err != nil {
				return err
			}
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM contacts WHERE id = ?`, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// finalizeIfNoMentions 当某内容已无任何 @ 关联时收尾：待办完成、留言/便签关闭。
func finalizeIfNoMentions(ctx context.Context, tx *sql.Tx, contentType string, contentID int64) error {
	var remaining int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM mentions WHERE content_type = ? AND content_id = ?`, contentType, contentID).Scan(&remaining); err != nil {
		return err
	}
	if remaining > 0 {
		return nil
	}
	now := time.Now().Unix()
	switch contentType {
	case "task":
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, contentID).Scan(&status); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		if status == "completed" {
			return nil
		}
		var occ, series sql.NullString
		if err := tx.QueryRowContext(ctx,
			`SELECT COALESCE(occurrence_date, due_date, ''), series_id FROM tasks WHERE id = ?`, contentID).Scan(&occ, &series); err != nil {
			return err
		}
		date := occ.String
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		var seriesPtr *string
		if series.Valid {
			seriesPtr = &series.String
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO task_completions (task_id, occurrence_date, series_id, completed_at) VALUES (?, ?, ?, ?)`,
			contentID, date, seriesPtr, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE tasks SET status='completed', completed_at=?, updated_at=? WHERE id=?`, now, now, contentID); err != nil {
			return err
		}
	case "message":
		if _, err := tx.ExecContext(ctx,
			`UPDATE message_posts SET closed_at=?, updated_at=? WHERE id=?`, now, now, contentID); err != nil {
			return err
		}
	case "note":
		if _, err := tx.ExecContext(ctx,
			`UPDATE notes SET closed_at=?, updated_at=? WHERE id=?`, now, now, contentID); err != nil {
			return err
		}
	}
	return nil
}

// ContactMentionCounts 汇总联系人被 @ 的关联数量（按内容类型归类）。
type ContactMentionCounts struct {
	Tasks    int `json:"tasks"`
	Messages int `json:"messages"`
	Notes    int `json:"notes"`
}

// Total 返回关联总数。
func (c ContactMentionCounts) Total() int { return c.Tasks + c.Messages + c.Notes }

// ContactMentionCounts 查询单个联系人被 @ 的关联数量。
func (s *Store) ContactMentionCounts(ctx context.Context, contactID int64) (ContactMentionCounts, error) {
	var c ContactMentionCounts
	rows, err := s.DB.QueryContext(ctx, `SELECT content_type, COUNT(1) FROM mentions
		WHERE target_type = 'contact' AND target_id = ? GROUP BY content_type`, contactID)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var ct string
		var n int
		if err := rows.Scan(&ct, &n); err != nil {
			return c, err
		}
		switch ct {
		case "task":
			c.Tasks += n
		case "message", "message_reply":
			c.Messages += n
		case "note":
			c.Notes += n
		}
	}
	return c, rows.Err()
}

// ListContactIDs 返回全部可见联系人 ID（用于紧急联络下拉与导入去重）。
func (s *Store) ListContactIDs(ctx context.Context, viewerID int64) ([]int64, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id FROM contacts WHERE owner_scope = 'household' OR created_by = ? ORDER BY id`, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
