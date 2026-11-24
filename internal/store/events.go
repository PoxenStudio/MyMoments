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

const eventColumns = `e.id, e.title, e.description, e.start_local, e.end_local, e.all_day, e.location, e.color,
	e.icon, e.recurrence_rule, e.series_id, e.visibility, e.source, e.source_ref, e.created_by, e.created_at,
	e.updated_at, e.reminder_offsets`

func scanEvent(scanner interface{ Scan(...any) error }) (*model.CalendarEvent, error) {
	var ev model.CalendarEvent
	var desc, endLocal, location, color, icon, rule, series, sourceRef, offsets sql.NullString
	if err := scanner.Scan(&ev.ID, &ev.Title, &desc, &ev.StartLocal, &endLocal, &ev.AllDay, &location, &color,
		&icon, &rule, &series, &ev.Visibility, &ev.Source, &sourceRef, &ev.CreatedBy, &ev.CreatedAt,
		&ev.UpdatedAt, &offsets); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	ev.Description = ps(desc)
	ev.EndLocal = ps(endLocal)
	ev.Location = ps(location)
	ev.Color = ps(color)
	ev.Icon = ps(icon)
	ev.RecurrenceRule = ps(rule)
	ev.SeriesID = ps(series)
	ev.SourceRef = ps(sourceRef)
	ev.ReminderOffsets = parseOffsets(offsets.String)
	ev.Assignees = []model.User{}
	date := strings.TrimSpace(strings.SplitN(ev.StartLocal, " ", 2)[0])
	if date != "" {
		ev.OccurrenceDate = &date
	}
	return &ev, nil
}

func (s *Store) visibleEventClause() string {
	return "(e.visibility = 'household' OR e.created_by = ? OR (e.visibility = 'selected' AND EXISTS (SELECT 1 FROM event_assignees ea WHERE ea.event_id = e.id AND ea.user_id = ?)))"
}

// ListEvents 返回范围内的可见事件基记录；重复系列需由调用方展开。
func (s *Store) ListEvents(ctx context.Context, viewerID int64, from, to string, includeBirthday bool) ([]model.CalendarEvent, error) {
	include := 0
	if includeBirthday {
		include = 1
	}
	query := `SELECT ` + eventColumns + ` FROM calendar_events e WHERE ` + s.visibleEventClause() + `
		AND (e.source = 'manual' OR ? = 1)
		AND (
			(COALESCE(e.recurrence_rule, '') = '' AND substr(e.start_local, 1, 10) BETWEEN ? AND ?)
			OR (COALESCE(e.recurrence_rule, '') <> '' AND substr(e.start_local, 1, 10) <= ?)
		)
		ORDER BY e.start_local, e.id`
	rows, err := s.DB.QueryContext(ctx, query, viewerID, viewerID, include, from, to, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.CalendarEvent, 0)
	for rows.Next() {
		ev, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := s.loadEventRelations(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// GetEvent 读取单个事件及参与成员。
func (s *Store) GetEvent(ctx context.Context, id int64) (*model.CalendarEvent, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM calendar_events e WHERE e.id = ?`, id)
	ev, err := scanEvent(row)
	if err != nil {
		return nil, err
	}
	if err := s.loadEventRelations(ctx, ev); err != nil {
		return nil, err
	}
	return ev, nil
}

func (s *Store) loadEventRelations(ctx context.Context, ev *model.CalendarEvent) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+userColumns+` FROM users u
		JOIN event_assignees ea ON ea.user_id = u.id WHERE ea.event_id = ? ORDER BY u.sort_order, u.id`, ev.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return err
		}
		ev.Assignees = append(ev.Assignees, u.Public())
	}
	return rows.Err()
}

// CreateEvent 创建事件并保存参与成员。
func (s *Store) CreateEvent(ctx context.Context, ev *model.CalendarEvent) (int64, error) {
	var id int64
	err := s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		res, err := tx.ExecContext(ctx, `INSERT INTO calendar_events
			(title, description, start_local, end_local, all_day, location, color, icon, recurrence_rule, series_id,
			visibility, source, source_ref, reminder_offsets, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ev.Title, ps2(ev.Description), ev.StartLocal, ps2(ev.EndLocal), ev.AllDay, ps2(ev.Location),
			ps2(ev.Color), ps2(ev.Icon), ps2(ev.RecurrenceRule), ps2(ev.SeriesID), ev.Visibility, ev.Source,
			ps2(ev.SourceRef), offsetsJSON(ev.ReminderOffsets), ev.CreatedBy, now, now)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return err
		}
		for _, uid := range assigneeIDs(ev.Assignees) {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO event_assignees (event_id, user_id) VALUES (?, ?)`, id, uid); err != nil {
				return err
			}
		}
		return nil
	})
	return id, err
}

// UpdateEvent 更新事件及参与成员。
func (s *Store) UpdateEvent(ctx context.Context, ev *model.CalendarEvent) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		if _, err := tx.ExecContext(ctx, `UPDATE calendar_events SET title=?, description=?, start_local=?, end_local=?,
			all_day=?, location=?, color=?, icon=?, recurrence_rule=?, series_id=?, visibility=?, reminder_offsets=?,
			updated_at=? WHERE id=?`,
			ev.Title, ps2(ev.Description), ev.StartLocal, ps2(ev.EndLocal), ev.AllDay, ps2(ev.Location),
			ps2(ev.Color), ps2(ev.Icon), ps2(ev.RecurrenceRule), ps2(ev.SeriesID), ev.Visibility,
			offsetsJSON(ev.ReminderOffsets), now, ev.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM event_assignees WHERE event_id = ?`, ev.ID); err != nil {
			return err
		}
		for _, uid := range assigneeIDs(ev.Assignees) {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO event_assignees (event_id, user_id) VALUES (?, ?)`, ev.ID, uid); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteEvent 删除事件；重复例外随系列一起清理。
func (s *Store) DeleteEvent(ctx context.Context, id int64) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		var series sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT series_id FROM calendar_events WHERE id = ?`, id).Scan(&series)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM calendar_events WHERE id = ?`, id); err != nil {
			return err
		}
		if series.Valid && series.String != "" {
			if _, err := tx.ExecContext(ctx, `DELETE FROM recurrence_exceptions WHERE series_id = ?`, series.String); err != nil {
				return err
			}
		}
		return nil
	})
}

// EventException 描述重复系列中某个发生项的调整。
type EventException struct {
	SeriesID       string
	OccurrenceDate string
	Action         string
	OverrideFields string
}

// UpsertEventException 写入或覆盖某发生项的例外。
func (s *Store) UpsertEventException(ctx context.Context, e EventException) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO recurrence_exceptions (series_id, occurrence_date, action, override_fields)
		VALUES (?, ?, ?, NULLIF(?, ''))
		ON CONFLICT(series_id, occurrence_date) DO UPDATE SET action = excluded.action, override_fields = excluded.override_fields`,
		e.SeriesID, e.OccurrenceDate, e.Action, e.OverrideFields)
	return err
}

// LoadEventExceptions 批量读取系列例外，按 series_id → occurrence_date 建索引。
func (s *Store) LoadEventExceptions(ctx context.Context, seriesIDs []string) (map[string]map[string]EventException, error) {
	out := map[string]map[string]EventException{}
	if len(seriesIDs) == 0 {
		return out, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(seriesIDs)), ",")
	args := make([]any, 0, len(seriesIDs))
	for _, id := range seriesIDs {
		args = append(args, id)
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT series_id, occurrence_date, action, COALESCE(override_fields, '')
		FROM recurrence_exceptions WHERE series_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e EventException
		if err := rows.Scan(&e.SeriesID, &e.OccurrenceDate, &e.Action, &e.OverrideFields); err != nil {
			return nil, err
		}
		if out[e.SeriesID] == nil {
			out[e.SeriesID] = map[string]EventException{}
		}
		out[e.SeriesID][e.OccurrenceDate] = e
	}
	return out, rows.Err()
}

// DeleteEventExceptionsFrom 删除某系列从指定日期（含）起的例外。
func (s *Store) DeleteEventExceptionsFrom(ctx context.Context, seriesID, from string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM recurrence_exceptions WHERE series_id = ? AND occurrence_date >= ?`, seriesID, from)
	return err
}

// FindBirthdayEvent 读取生日生成的日历事件。
func (s *Store) FindBirthdayEvent(ctx context.Context, birthdayID int64) (*model.CalendarEvent, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM calendar_events e
		WHERE e.source = 'birthday' AND e.source_ref = ? ORDER BY e.id DESC LIMIT 1`, itoa64(birthdayID))
	return scanEvent(row)
}

func itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// EventOccurrenceOverride 是单次发生项的字段覆盖。
type EventOccurrenceOverride struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	StartLocal  *string `json:"startLocal,omitempty"`
	EndLocal    *string `json:"endLocal,omitempty"`
	AllDay      *bool   `json:"allDay,omitempty"`
	Location    *string `json:"location,omitempty"`
	Color       *string `json:"color,omitempty"`
	Icon        *string `json:"icon,omitempty"`
}

// Apply 将覆盖字段写入事件副本。
func (o EventOccurrenceOverride) Apply(ev *model.CalendarEvent) {
	if o.Title != nil {
		ev.Title = *o.Title
	}
	if o.Description != nil {
		ev.Description = nonEmpty(o.Description)
	}
	if o.StartLocal != nil {
		ev.StartLocal = *o.StartLocal
	}
	if o.EndLocal != nil {
		ev.EndLocal = nonEmpty(o.EndLocal)
	}
	if o.AllDay != nil {
		ev.AllDay = *o.AllDay
	}
	if o.Location != nil {
		ev.Location = nonEmpty(o.Location)
	}
	if o.Color != nil {
		ev.Color = nonEmpty(o.Color)
	}
	if o.Icon != nil {
		ev.Icon = nonEmpty(o.Icon)
	}
}

// ParseOverride 解析例外中的覆盖字段。
func ParseOverride(raw string) EventOccurrenceOverride {
	var o EventOccurrenceOverride
	if raw == "" {
		return o
	}
	_ = json.Unmarshal([]byte(raw), &o)
	return o
}

func nonEmpty(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}
