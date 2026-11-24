package model

// User 是可登录的家庭成员。
type User struct {
	ID                 int64   `json:"id"`
	Username           string  `json:"username"`
	DisplayName        string  `json:"displayName"`
	PasswordHash       string  `json:"-"`
	AvatarKey          *string `json:"avatarKey"`
	Role               string  `json:"role"`
	Status             string  `json:"status"`
	MustChangePassword bool    `json:"mustChangePassword"`
	Relation           *string `json:"relation"`
	SortOrder          int     `json:"sortOrder"`
	CreatedAt          int64   `json:"createdAt"`
	UpdatedAt          int64   `json:"updatedAt"`
}

// Public 去掉敏感字段，用于返回给前端。
func (u User) Public() User {
	u.PasswordHash = ""
	return u
}

// Household 是家庭级设置。
type Household struct {
	ID                    int64    `json:"-"`
	Name                  string   `json:"name"`
	Timezone              string   `json:"timezone"`
	WeekStart             int      `json:"weekStart"`
	DefaultWeatherCity    *string  `json:"defaultWeatherCity"`
	DefaultWeatherLat     *float64 `json:"defaultWeatherLat"`
	DefaultWeatherLon     *float64 `json:"defaultWeatherLon"`
	DefaultReminderOffset int      `json:"defaultReminderOffset"`
	DefaultDashboard      *string  `json:"defaultDashboard"`
	CreatedAt             int64    `json:"-"`
	UpdatedAt             int64    `json:"-"`
}

// Preferences 是个人偏好。
type Preferences struct {
	UserID            int64    `json:"-"`
	Locale            string   `json:"locale"`
	DateFormat        string   `json:"dateFormat"`
	Dashboard         *string  `json:"dashboard"`
	WeatherCity       *string  `json:"weatherCity"`
	WeatherLat        *float64 `json:"weatherLat"`
	WeatherLon        *float64 `json:"weatherLon"`
	WeatherAuto       bool     `json:"weatherAuto"`
	NotificationPrefs *string  `json:"notificationPrefs"`
	UpdatedAt         int64    `json:"updatedAt"`
}

// Session 是服务端会话记录，ID 存令牌哈希。
type Session struct {
	ID         string
	UserID     int64
	ExpiresAt  int64
	CreatedAt  int64
	LastSeenAt int64
	UserAgent  string
	IP         string
}

// APIToken 是长期访问令牌，Hash 为令牌哈希。
type APIToken struct {
	ID         int64
	UserID     int64
	Name       string
	Hash       string
	Prefix     string
	Scopes     string
	ExpiresAt  *int64
	LastUsedAt *int64
	CreatedAt  int64
	RevokedAt  *int64
}
