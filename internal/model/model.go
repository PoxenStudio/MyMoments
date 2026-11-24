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

// Task 是待办，含关联数据。
type Task struct {
	ID              int64           `json:"id"`
	Title           string          `json:"title"`
	Description     *string         `json:"description"`
	Status          string          `json:"status"`
	Priority        string          `json:"priority"`
	StartDate       *string         `json:"startDate"`
	DueDate         *string         `json:"dueDate"`
	DueTime         *string         `json:"dueTime"`
	TemplateID      *int64          `json:"templateId"`
	RecurrenceRule  *string         `json:"recurrenceRule"`
	SeriesID        *string         `json:"seriesId"`
	OccurrenceDate  *string         `json:"occurrenceDate"`
	Color           *string         `json:"color"`
	Icon            *string         `json:"icon"`
	Visibility      string          `json:"visibility"`
	CompletedAt     *int64          `json:"completedAt"`
	ArchivedAt      *int64          `json:"archivedAt"`
	CreatedBy       int64           `json:"createdBy"`
	CreatedAt       int64           `json:"createdAt"`
	UpdatedAt       int64           `json:"updatedAt"`
	ReminderOffsets []int           `json:"reminderOffsets"`
	Assignees       []User          `json:"assignees"`
	Subtasks        []Subtask       `json:"subtasks"`
	MediaCount      int             `json:"mediaCount"`
	Completion      *TaskCompletion `json:"completion"`
}

// Subtask 是任务子项。
type Subtask struct {
	ID       int64  `json:"id"`
	TaskID   int64  `json:"taskId"`
	Title    string `json:"title"`
	Done     bool   `json:"done"`
	Position int    `json:"position"`
}

// TaskCompletion 是一次完成记录。
type TaskCompletion struct {
	ID             int64   `json:"id"`
	TaskID         int64   `json:"taskId"`
	OccurrenceDate string  `json:"occurrenceDate"`
	SeriesID       *string `json:"seriesId"`
	DoneByUserID   *int64  `json:"doneByUserId"`
	ActualUserID   *int64  `json:"actualUserId"`
	CompletedAt    int64   `json:"completedAt"`
	DoneByName     string  `json:"doneByName,omitempty"`
	ActualName     string  `json:"actualName,omitempty"`
	UndoneAt       *int64  `json:"-"`
	NextTaskID     *int64  `json:"-"`
}

// TaskMedia 是任务附件元数据，存储键不对外暴露。
type TaskMedia struct {
	ID             int64   `json:"id"`
	TaskID         int64   `json:"taskId"`
	OccurrenceDate *string `json:"occurrenceDate"`
	StorageKey     string  `json:"-"`
	OriginalName   *string `json:"originalName"`
	MediaType      string  `json:"mediaType"`
	MimeType       string  `json:"mimeType"`
	SizeBytes      int64   `json:"sizeBytes"`
	DurationMs     *int64  `json:"durationMs"`
	CreatedBy      int64   `json:"createdBy"`
	CreatedAt      int64   `json:"createdAt"`
}

// TaskTemplate 是待办模板。
type TaskTemplate struct {
	ID                 int64    `json:"id"`
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	Description        *string  `json:"description"`
	Icon               *string  `json:"icon"`
	Color              *string  `json:"color"`
	Priority           *string  `json:"priority"`
	DefaultAssigneeIDs []int64  `json:"defaultAssigneeIds"`
	DefaultDueTime     *string  `json:"defaultDueTime"`
	RecurrenceRule     *string  `json:"recurrenceRule"`
	ReminderOffset     *int     `json:"reminderOffset"`
	SubtaskTemplate    []string `json:"subtaskTemplate"`
	Enabled            bool     `json:"enabled"`
	SortOrder          int      `json:"sortOrder"`
	CreatedBy          int64    `json:"createdBy"`
	CreatedAt          int64    `json:"createdAt"`
}

// Reminder 是个人提醒。
type Reminder struct {
	ID             int64   `json:"id"`
	UserID         int64   `json:"userId"`
	SourceType     string  `json:"sourceType"`
	SourceID       int64   `json:"sourceId"`
	OccurrenceDate *string `json:"occurrenceDate"`
	Title          *string `json:"title"`
	RemindAt       int64   `json:"remindAt"`
	Status         string  `json:"status"`
	DeliveredAt    *int64  `json:"deliveredAt"`
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
