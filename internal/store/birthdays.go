package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"mymoments/internal/model"
)

const birthdayColumns = `b.id, b.name, b.month, b.day, b.year_unknown, b.birth_year, b.note, b.avatar_key,
	b.contact_id, b.user_id, b.reminder_offset, b.calendar_event_id, b.created_by, b.created_at, b.updated_at`

func scanBirthday(scanner interface{ Scan(...any) error }) (*model.Birthday, error) {
	var b model.Birthday
	var birthYear, contactID, userID, reminder, eventID sql.NullInt64
	var note, avatar sql.NullString
	if err := scanner.Scan(&b.ID, &b.Name, &b.Month, &b.Day, &b.YearUnknown, &birthYear, &note, &avatar,
		&contactID, &userID, &reminder, &eventID, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if birthYear.Valid {
		v := int(birthYear.Int64)
		b.BirthYear = &v
	}
	b.Note = ps(note)
	b.AvatarKey = ps(avatar)
	if contactID.Valid {
		b.ContactID = &contactID.Int64
	}
	if userID.Valid {
		b.UserID = &userID.Int64
	}
	if reminder.Valid {
		v := int(reminder.Int64)
		b.ReminderOffset = &v
	}
	if eventID.Valid {
		b.CalendarEventID = &eventID.Int64
	}
	return &b, nil
}

// BirthdayFilter 描述生日列表筛选。
type BirthdayFilter struct {
	Query  string
	Source string
}

// ListBirthdays 返回生日记录。
func (s *Store) ListBirthdays(ctx context.Context, f BirthdayFilter) ([]model.Birthday, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, "b.name LIKE ?")
		args = append(args, "%"+q+"%")
	}
	switch f.Source {
	case "member":
		where = append(where, "b.user_id IS NOT NULL")
	case "contact":
		where = append(where, "b.contact_id IS NOT NULL")
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+birthdayColumns+` FROM birthdays b WHERE `+
		strings.Join(where, " AND ")+` ORDER BY b.month, b.day, b.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Birthday, 0)
	for rows.Next() {
		b, err := scanBirthday(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

// GetBirthday 读取单个生日。
func (s *Store) GetBirthday(ctx context.Context, id int64) (*model.Birthday, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+birthdayColumns+` FROM birthdays b WHERE b.id = ?`, id)
	return scanBirthday(row)
}

// CreateBirthday 创建生日记录。
func (s *Store) CreateBirthday(ctx context.Context, b *model.Birthday) (int64, error) {
	now := time.Now().Unix()
	res, err := s.DB.ExecContext(ctx, `INSERT INTO birthdays
		(name, month, day, year_unknown, birth_year, note, avatar_key, contact_id, user_id, reminder_offset,
		calendar_event_id, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Name, b.Month, b.Day, b.YearUnknown, nidInt(b.BirthYear), ps2(b.Note), ps2(b.AvatarKey),
		nid(b.ContactID), nid(b.UserID), nidInt(b.ReminderOffset), nid(b.CalendarEventID), b.CreatedBy, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateBirthday 更新生日记录。
func (s *Store) UpdateBirthday(ctx context.Context, b *model.Birthday) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE birthdays SET name=?, month=?, day=?, year_unknown=?, birth_year=?,
		note=?, avatar_key=?, contact_id=?, user_id=?, reminder_offset=?, calendar_event_id=?, updated_at=? WHERE id=?`,
		b.Name, b.Month, b.Day, b.YearUnknown, nidInt(b.BirthYear), ps2(b.Note), ps2(b.AvatarKey),
		nid(b.ContactID), nid(b.UserID), nidInt(b.ReminderOffset), nid(b.CalendarEventID), time.Now().Unix(), b.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteBirthday 删除生日记录。
func (s *Store) DeleteBirthday(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM birthdays WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetBirthdayByUser 查找成员已绑定的生日。
func (s *Store) GetBirthdayByUser(ctx context.Context, userID int64) (*model.Birthday, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+birthdayColumns+` FROM birthdays b WHERE b.user_id = ? LIMIT 1`, userID)
	return scanBirthday(row)
}

// GetBirthdayByContact 查找联系人已绑定的生日。
func (s *Store) GetBirthdayByContact(ctx context.Context, contactID int64) (*model.Birthday, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+birthdayColumns+` FROM birthdays b WHERE b.contact_id = ? LIMIT 1`, contactID)
	return scanBirthday(row)
}

func nidInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}
