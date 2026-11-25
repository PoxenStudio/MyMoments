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

// ErrNotFound 表示查询对象不存在。
var ErrNotFound = errors.New("not found")

// GetHousehold 返回家庭设置；未初始化时返回 ErrNotFound。
func (s *Store) GetHousehold(ctx context.Context) (*model.Household, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, name, timezone, week_start, date_format, default_weather_city, default_weather_lat,
		default_weather_lon, default_reminder_offset, default_dashboard, style, created_at, updated_at FROM household_settings WHERE id = 1`)
	var h model.Household
	var city, dashboard, style sql.NullString
	var lat, lon sql.NullFloat64
	if err := row.Scan(&h.ID, &h.Name, &h.Timezone, &h.WeekStart, &h.DateFormat, &city, &lat, &lon, &h.DefaultReminderOffset,
		&dashboard, &style, &h.CreatedAt, &h.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if city.Valid {
		h.DefaultWeatherCity = &city.String
	}
	if lat.Valid {
		h.DefaultWeatherLat = &lat.Float64
	}
	if lon.Valid {
		h.DefaultWeatherLon = &lon.Float64
	}
	if dashboard.Valid {
		h.DefaultDashboard = &dashboard.String
	}
	if style.Valid {
		h.Style = &style.String
	}
	return &h, nil
}

// CreateHouseholdIfAbsent 在事务中初始化家庭；返回 false 表示已被其他实例初始化。
func CreateHouseholdIfAbsent(ctx context.Context, tx *sql.Tx, name, timezone string) (bool, error) {
	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `INSERT INTO household_settings
		(id, name, timezone, week_start, default_reminder_offset, created_at, updated_at)
		VALUES (1, ?, ?, 1, 0, ?, ?) ON CONFLICT(id) DO NOTHING`, name, timezone, now, now)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// UpdateHousehold 更新家庭设置。
func (s *Store) UpdateHousehold(ctx context.Context, h *model.Household) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE household_settings SET name=?, timezone=?, week_start=?, date_format=?,
		default_weather_city=?, default_weather_lat=?, default_weather_lon=?, default_reminder_offset=?,
		default_dashboard=?, style=?, updated_at=? WHERE id = 1`,
		h.Name, h.Timezone, h.WeekStart, h.DateFormat, ns(h.DefaultWeatherCity), nf(h.DefaultWeatherLat), nf(h.DefaultWeatherLon),
		h.DefaultReminderOffset, ns(h.DefaultDashboard), ns(h.Style), time.Now().Unix())
	return err
}

// CountUsers 返回成员数量。
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&n)
	return n, err
}

// CountActiveAdmins 返回启用中的管理员数量，可排除指定用户。
func (s *Store) CountActiveAdmins(ctx context.Context, excludeID int64) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM users WHERE role='admin' AND status='active' AND id <> ?`, excludeID).Scan(&n)
	return n, err
}

const userColumns = `id, username, display_name, password_hash, avatar_key, role, status, must_change_password,
	relation, sort_order, created_at, updated_at`

func scanUser(scanner interface{ Scan(...any) error }) (*model.User, error) {
	var u model.User
	var avatar, relation sql.NullString
	if err := scanner.Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &avatar, &u.Role, &u.Status,
		&u.MustChangePassword, &relation, &u.SortOrder, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if avatar.Valid {
		u.AvatarKey = &avatar.String
	}
	if relation.Valid {
		u.Relation = &relation.String
	}
	return &u, nil
}

// GetUserByUsername 按小写用户名查找。
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE username = ?`, strings.ToLower(strings.TrimSpace(username)))
	return scanUser(row)
}

// GetUserByID 按 ID 查找。
func (s *Store) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id)
	return scanUser(row)
}

// ListUsers 返回全部成员，按排序与创建时间排列。
func (s *Store) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+userColumns+` FROM users ORDER BY sort_order, created_at, id`)
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

// CreateUserTx 在事务中创建成员并初始化偏好。
func CreateUserTx(ctx context.Context, tx *sql.Tx, u *model.User) (int64, error) {
	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `INSERT INTO users
		(username, display_name, password_hash, avatar_key, role, status, must_change_password, relation, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.ToLower(u.Username), u.DisplayName, u.PasswordHash, ns(u.AvatarKey), u.Role, u.Status,
		u.MustChangePassword, ns(u.Relation), u.SortOrder, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return 0, fmt.Errorf("username exists")
		}
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_preferences (user_id, locale, date_format, updated_at) VALUES (?, 'zh-CN', '24h', ?)`, id, now); err != nil {
		return 0, err
	}
	return id, nil
}

// UpdateUser 更新可编辑字段。
func (s *Store) UpdateUser(ctx context.Context, u *model.User) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET display_name=?, avatar_key=?, role=?, status=?,
		must_change_password=?, relation=?, sort_order=?, updated_at=? WHERE id=?`,
		u.DisplayName, ns(u.AvatarKey), u.Role, u.Status, u.MustChangePassword, ns(u.Relation), u.SortOrder, time.Now().Unix(), u.ID)
	return err
}

// UpdatePassword 更新口令哈希与首次改密标记。
func (s *Store) UpdatePassword(ctx context.Context, userID int64, hash string, mustChange bool) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET password_hash=?, must_change_password=?, updated_at=? WHERE id=?`,
		hash, mustChange, time.Now().Unix(), userID)
	return err
}

// GetPreferences 返回个人偏好，缺失时创建默认值。
func (s *Store) GetPreferences(ctx context.Context, userID int64) (*model.Preferences, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT user_id, locale, date_format, dashboard, weather_city, weather_lat,
		weather_lon, weather_auto, notification_prefs, style, updated_at FROM user_preferences WHERE user_id = ?`, userID)
	var p model.Preferences
	var dashboard, city, notif, style sql.NullString
	var lat, lon sql.NullFloat64
	err := row.Scan(&p.UserID, &p.Locale, &p.DateFormat, &dashboard, &city, &lat, &lon, &p.WeatherAuto, &notif, &style, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		now := time.Now().Unix()
		if _, err := s.DB.ExecContext(ctx, `INSERT INTO user_preferences (user_id, locale, date_format, updated_at) VALUES (?, 'zh-CN', '24h', ?) ON CONFLICT(user_id) DO NOTHING`, userID, now); err != nil {
			return nil, err
		}
		p = model.Preferences{UserID: userID, Locale: "zh-CN", DateFormat: "24h", UpdatedAt: now}
		return &p, nil
	}
	if err != nil {
		return nil, err
	}
	if dashboard.Valid {
		p.Dashboard = &dashboard.String
	}
	if city.Valid {
		p.WeatherCity = &city.String
	}
	if lat.Valid {
		p.WeatherLat = &lat.Float64
	}
	if lon.Valid {
		p.WeatherLon = &lon.Float64
	}
	if notif.Valid {
		p.NotificationPrefs = &notif.String
	}
	if style.Valid {
		p.Style = &style.String
	}
	return &p, nil
}

// UpdatePreferences 更新个人偏好。
func (s *Store) UpdatePreferences(ctx context.Context, p *model.Preferences) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE user_preferences SET locale=?, date_format=?, dashboard=?, weather_city=?,
		weather_lat=?, weather_lon=?, weather_auto=?, notification_prefs=?, style=?, updated_at=? WHERE user_id=?`,
		p.Locale, p.DateFormat, ns(p.Dashboard), ns(p.WeatherCity), nf(p.WeatherLat), nf(p.WeatherLon),
		p.WeatherAuto, ns(p.NotificationPrefs), ns(p.Style), time.Now().Unix(), p.UserID)
	return err
}

func ns(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

func nf(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}
