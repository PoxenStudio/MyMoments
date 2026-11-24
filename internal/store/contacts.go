package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"mymoments/internal/model"
)

const contactColumns = `c.id, c.name, c.organization, c.title, c.note, c.birthday_on, c.category, c.avatar_key,
	c.owner_scope, c.created_by, c.created_at, c.updated_at`

func scanContact(scanner interface{ Scan(...any) error }) (*model.Contact, error) {
	var c model.Contact
	var org, title, note, birthday, category, avatar sql.NullString
	if err := scanner.Scan(&c.ID, &c.Name, &org, &title, &note, &birthday, &category, &avatar, &c.OwnerScope,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.Organization = ps(org)
	c.Title = ps(title)
	c.Note = ps(note)
	c.BirthdayOn = ps(birthday)
	c.Category = ps(category)
	c.AvatarKey = ps(avatar)
	c.Items = []model.ContactItem{}
	return &c, nil
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
		where = append(where, "(c.name LIKE ? OR c.organization LIKE ? OR c.note LIKE ? OR EXISTS (SELECT 1 FROM contact_items ci WHERE ci.contact_id = c.id AND ci.value LIKE ?))")
		like := "%" + q + "%"
		args = append(args, like, like, like, like)
	}
	if f.Category != "" {
		where = append(where, "c.category = ?")
		args = append(args, f.Category)
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+contactColumns+` FROM contacts c WHERE `+
		strings.Join(where, " AND ")+` ORDER BY c.name, c.id`, args...)
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
		if err := s.loadContactItems(ctx, &out[i]); err != nil {
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
	if err := s.loadContactItems(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) loadContactItems(ctx context.Context, c *model.Contact) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, contact_id, kind, label, value, position
		FROM contact_items WHERE contact_id = ? ORDER BY position, id`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var it model.ContactItem
		var label sql.NullString
		if err := rows.Scan(&it.ID, &it.ContactID, &it.Kind, &label, &it.Value, &it.Position); err != nil {
			return err
		}
		it.Label = ps(label)
		c.Items = append(c.Items, it)
	}
	return rows.Err()
}

// CreateContact 创建联系人及多值字段。
func (s *Store) CreateContact(ctx context.Context, c *model.Contact) (int64, error) {
	var id int64
	err := s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		res, err := tx.ExecContext(ctx, `INSERT INTO contacts
			(name, organization, title, note, birthday_on, category, avatar_key, owner_scope, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.Name, ps2(c.Organization), ps2(c.Title), ps2(c.Note), ps2(c.BirthdayOn), ps2(c.Category),
			ps2(c.AvatarKey), c.OwnerScope, c.CreatedBy, now, now)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return err
		}
		return insertContactItems(ctx, tx, id, c.Items)
	})
	return id, err
}

// UpdateContact 更新联系人并替换多值字段。
func (s *Store) UpdateContact(ctx context.Context, c *model.Contact) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE contacts SET name=?, organization=?, title=?, note=?, birthday_on=?,
			category=?, avatar_key=?, owner_scope=?, updated_at=? WHERE id=?`,
			c.Name, ps2(c.Organization), ps2(c.Title), ps2(c.Note), ps2(c.BirthdayOn), ps2(c.Category),
			ps2(c.AvatarKey), c.OwnerScope, time.Now().Unix(), c.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM contact_items WHERE contact_id = ?`, c.ID); err != nil {
			return err
		}
		return insertContactItems(ctx, tx, c.ID, c.Items)
	})
}

func insertContactItems(ctx context.Context, tx *sql.Tx, contactID int64, items []model.ContactItem) error {
	for i, it := range items {
		if _, err := tx.ExecContext(ctx, `INSERT INTO contact_items (contact_id, kind, label, value, position)
			VALUES (?, ?, ?, ?, ?)`, contactID, it.Kind, ps2(it.Label), it.Value, i); err != nil {
			return err
		}
	}
	return nil
}

// DeleteContact 删除联系人；生日记录保留但解除关联。
func (s *Store) DeleteContact(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM contacts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
