package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"mymoments/internal/model"
)

const birthdayColumns = `b.id, b.name, b.note, b.avatar_key, b.contact_id, b.user_id,
	b.created_by, b.created_at, b.updated_at`

const birthdayDateColumns = `d.id, d.birthday_id, d.calendar, d.month, d.day, d.year_unknown,
	d.birth_year, d.reminder_offset, d.calendar_event_id, d.created_at, d.updated_at`

func scanBirthday(scanner interface{ Scan(...any) error }) (*model.Birthday, error) {
	var b model.Birthday
	var contactID, userID sql.NullInt64
	var note, avatar sql.NullString
	if err := scanner.Scan(&b.ID, &b.Name, &note, &avatar, &contactID, &userID,
		&b.CreatedBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	b.Note = ps(note)
	b.AvatarKey = ps(avatar)
	if contactID.Valid {
		b.ContactID = &contactID.Int64
	}
	if userID.Valid {
		b.UserID = &userID.Int64
	}
	b.Dates = []model.BirthdayDate{}
	return &b, nil
}

func scanBirthdayDate(scanner interface{ Scan(...any) error }) (*model.BirthdayDate, error) {
	var d model.BirthdayDate
	var birthdayID int64
	var birthYear, reminder, eventID sql.NullInt64
	if err := scanner.Scan(&d.ID, &birthdayID, &d.Calendar, &d.Month, &d.Day, &d.YearUnknown,
		&birthYear, &reminder, &eventID, &d.CreatedAt, &d.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if birthYear.Valid {
		v := int(birthYear.Int64)
		d.BirthYear = &v
	}
	if reminder.Valid {
		v := int(reminder.Int64)
		d.ReminderOffset = &v
	}
	if eventID.Valid {
		d.CalendarEventID = &eventID.Int64
	}
	return &d, nil
}

// BirthdayFilter 描述生日列表筛选。
type BirthdayFilter struct {
	Query  string
	Source string
}

// ListBirthdays 返回生日档案及其日期。
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
		strings.Join(where, " AND ")+` ORDER BY b.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Birthday, 0)
	ids := make([]int64, 0)
	for rows.Next() {
		b, err := scanBirthday(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
		ids = append(ids, b.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	dates, err := s.birthdayDatesByProfile(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if list, ok := dates[out[i].ID]; ok {
			out[i].Dates = list
		}
	}
	return out, nil
}

// GetBirthday 读取单个生日档案及日期。
func (s *Store) GetBirthday(ctx context.Context, id int64) (*model.Birthday, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+birthdayColumns+` FROM birthdays b WHERE b.id = ?`, id)
	b, err := scanBirthday(row)
	if err != nil {
		return nil, err
	}
	dates, err := s.loadBirthdayDates(ctx, id)
	if err != nil {
		return nil, err
	}
	b.Dates = dates
	return b, nil
}

// CreateBirthday 创建生日档案和全部日期。
func (s *Store) CreateBirthday(ctx context.Context, b *model.Birthday) (int64, error) {
	now := time.Now().Unix()
	err := s.Tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `INSERT INTO birthdays
			(name, note, avatar_key, contact_id, user_id, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			b.Name, ps2(b.Note), ps2(b.AvatarKey), nid(b.ContactID), nid(b.UserID),
			b.CreatedBy, now, now)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		b.ID = id
		b.CreatedAt = now
		b.UpdatedAt = now
		for i := range b.Dates {
			if err := insertBirthdayDate(ctx, tx, id, &b.Dates[i], now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return b.ID, nil
}

// UpdateBirthday 更新生日档案并用给定日期整体替换子表。
func (s *Store) UpdateBirthday(ctx context.Context, b *model.Birthday) error {
	now := time.Now().Unix()
	return s.Tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE birthdays SET name=?, note=?, avatar_key=?,
			contact_id=?, user_id=?, updated_at=? WHERE id=?`,
			b.Name, ps2(b.Note), ps2(b.AvatarKey), nid(b.ContactID), nid(b.UserID), now, b.ID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return ErrNotFound
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM birthday_dates WHERE birthday_id = ?`, b.ID); err != nil {
			return err
		}
		for i := range b.Dates {
			if err := insertBirthdayDate(ctx, tx, b.ID, &b.Dates[i], now); err != nil {
				return err
			}
		}
		b.UpdatedAt = now
		return nil
	})
}

// DeleteBirthday 删除生日档案与日期。
func (s *Store) DeleteBirthday(ctx context.Context, id int64) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM birthday_dates WHERE birthday_id = ?`, id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM birthdays WHERE id = ?`, id)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// SetBirthdayDateEvent 记录某个日期关联的日历事件。
func (s *Store) SetBirthdayDateEvent(ctx context.Context, dateID int64, eventID *int64) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE birthday_dates SET calendar_event_id=?, updated_at=? WHERE id=?`,
		nid(eventID), time.Now().Unix(), dateID)
	return err
}

// GetBirthdayByUser 查找成员已绑定的生日档案。
func (s *Store) GetBirthdayByUser(ctx context.Context, userID int64) (*model.Birthday, error) {
	var id int64
	err := s.DB.QueryRowContext(ctx, `SELECT id FROM birthdays WHERE user_id = ? LIMIT 1`, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.GetBirthday(ctx, id)
}

// GetBirthdayByContact 查找联系人已绑定的生日档案。
func (s *Store) GetBirthdayByContact(ctx context.Context, contactID int64) (*model.Birthday, error) {
	var id int64
	err := s.DB.QueryRowContext(ctx, `SELECT id FROM birthdays WHERE contact_id = ? LIMIT 1`, contactID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.GetBirthday(ctx, id)
}

func (s *Store) loadBirthdayDates(ctx context.Context, birthdayID int64) ([]model.BirthdayDate, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+birthdayDateColumns+` FROM birthday_dates d
		WHERE d.birthday_id = ? ORDER BY d.calendar DESC, d.id`, birthdayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.BirthdayDate, 0, 2)
	for rows.Next() {
		d, err := scanBirthdayDate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (s *Store) birthdayDatesByProfile(ctx context.Context, ids []int64) (map[int64][]model.BirthdayDate, error) {
	out := make(map[int64][]model.BirthdayDate, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+birthdayDateColumns+` FROM birthday_dates d
		WHERE d.birthday_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY d.birthday_id, d.calendar DESC, d.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var birthdayID int64
		// scanBirthdayDate 不返回 birthday_id，这里单独先读一列。
		var d model.BirthdayDate
		var birthYear, reminder, eventID sql.NullInt64
		if err := rows.Scan(&d.ID, &birthdayID, &d.Calendar, &d.Month, &d.Day, &d.YearUnknown,
			&birthYear, &reminder, &eventID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		if birthYear.Valid {
			v := int(birthYear.Int64)
			d.BirthYear = &v
		}
		if reminder.Valid {
			v := int(reminder.Int64)
			d.ReminderOffset = &v
		}
		if eventID.Valid {
			d.CalendarEventID = &eventID.Int64
		}
		out[birthdayID] = append(out[birthdayID], d)
	}
	return out, rows.Err()
}

func insertBirthdayDate(ctx context.Context, tx *sql.Tx, birthdayID int64, d *model.BirthdayDate, now int64) error {
	res, err := tx.ExecContext(ctx, `INSERT INTO birthday_dates
		(birthday_id, calendar, month, day, year_unknown, birth_year, reminder_offset,
		calendar_event_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		birthdayID, d.Calendar, d.Month, d.Day, boolInt(d.YearUnknown), nidInt(d.BirthYear),
		nidInt(d.ReminderOffset), nid(d.CalendarEventID), now, now)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	d.ID = id
	d.CreatedAt = now
	d.UpdatedAt = now
	return nil
}

func nidInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}
