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
	ID                     int64    `json:"-"`
	Name                   string   `json:"name"`
	Timezone               string   `json:"timezone"`
	WeekStart              int      `json:"weekStart"`
	DateFormat             string   `json:"dateFormat"`
	DefaultWeatherCity     *string  `json:"defaultWeatherCity"`
	DefaultWeatherLat      *float64 `json:"defaultWeatherLat"`
	DefaultWeatherLon      *float64 `json:"defaultWeatherLon"`
	DefaultWeatherProvider *string  `json:"defaultWeatherProvider"`
	DefaultReminderOffset  int      `json:"defaultReminderOffset"`
	DefaultDashboard       *string  `json:"defaultDashboard"`
	Style                  *string  `json:"style"`
	CreatedAt              int64    `json:"-"`
	UpdatedAt              int64    `json:"-"`
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
	Style             *string  `json:"style"`
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
	Mentions        []MentionTarget `json:"mentions"`
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

// PhotoAlbum 是首页照片墙的一张照片，文件存储键不对外暴露。
type PhotoAlbum struct {
	ID           int64   `json:"id"`
	StorageKey   string  `json:"-"`
	OriginalName *string `json:"originalName"`
	MimeType     string  `json:"mimeType"`
	SizeBytes    int64   `json:"sizeBytes"`
	Fit          string  `json:"fit"`
	CreatedBy    int64   `json:"createdBy"`
	CreatedAt    int64   `json:"createdAt"`
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

// CalendarEvent 是日历事件；重复系列以单条记录存储，occurrenceDate 表示展开后的发生日期。
type CalendarEvent struct {
	ID              int64   `json:"id"`
	Title           string  `json:"title"`
	Description     *string `json:"description"`
	StartLocal      string  `json:"startLocal"`
	EndLocal        *string `json:"endLocal"`
	AllDay          bool    `json:"allDay"`
	Location        *string `json:"location"`
	Color           *string `json:"color"`
	Icon            *string `json:"icon"`
	RecurrenceRule  *string `json:"recurrenceRule"`
	SeriesID        *string `json:"seriesId"`
	OccurrenceDate  *string `json:"occurrenceDate"`
	Visibility      string  `json:"visibility"`
	Source          string  `json:"source"`
	SourceRef       *string `json:"-"`
	CreatedBy       int64   `json:"createdBy"`
	CreatedAt       int64   `json:"createdAt"`
	UpdatedAt       int64   `json:"updatedAt"`
	ReminderOffsets []int   `json:"reminderOffsets"`
	Assignees       []User  `json:"assignees"`
}

// Contact 是家庭通讯录联系人。
type Contact struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	Organization   *string           `json:"organization"`
	Title          *string           `json:"title"`
	Note           *string           `json:"note"`
	BirthdayMonth  *int              `json:"birthdayMonth"`
	BirthdayDay    *int              `json:"birthdayDay"`
	BirthYear      *int              `json:"birthYear"`
	ShowInCalendar bool              `json:"showInCalendar"`
	Pinned         bool              `json:"pinned"`
	Category       *string           `json:"category"`
	AvatarKey      *string           `json:"avatarKey"`
	Wechat         *string           `json:"wechat"`
	OfficePhone    *string           `json:"officePhone"`
	OwnerScope     string            `json:"ownerScope"`
	CreatedBy      int64             `json:"createdBy"`
	Items          []ContactItem     `json:"items"`
	Relations      []ContactRelation `json:"relations"`
	Reminders      []ContactReminder `json:"reminders"`
	CreatedAt      int64             `json:"createdAt"`
	UpdatedAt      int64             `json:"updatedAt"`
}

// ContactItem 是联系人的电话/邮箱/地址等多项字段。
type ContactItem struct {
	ID            int64   `json:"id"`
	ContactID     int64   `json:"contactId,omitempty"`
	Kind          string  `json:"kind"`
	Label         *string `json:"label"`
	Value         string  `json:"value"`
	Position      int     `json:"position,omitempty"`
	IsPrimaryCall bool    `json:"isPrimaryCall"`
	IsPrimarySms  bool    `json:"isPrimarySms"`
}

// ContactRelation 是联系人与其他联系人的关联（如紧急联络）。
type ContactRelation struct {
	ContactID        int64  `json:"contactId,omitempty"`
	RelatedContactID int64  `json:"relatedContactId"`
	Relation         string `json:"relation"`
	RelatedName      string `json:"relatedName,omitempty"`
}

// ContactReminder 是联系人的问候/提醒偏好。
type ContactReminder struct {
	ID             int64   `json:"id"`
	ContactID      int64   `json:"contactId,omitempty"`
	Kind           string  `json:"kind"`
	Title          string  `json:"title"`
	Frequency      string  `json:"frequency"`
	DueDate        *string `json:"dueDate"`
	RecurrenceRule *string `json:"recurrenceRule"`
	Color          *string `json:"color"`
	Enabled        bool    `json:"enabled"`
	CreatedAt      int64   `json:"createdAt"`
	UpdatedAt      int64   `json:"updatedAt"`
}

// BirthdayDate 是生日档案下的一个日期，公历或农历各一条。
type BirthdayDate struct {
	ID              int64   `json:"id"`
	Calendar        string  `json:"calendar"`
	Month           int     `json:"month"`
	Day             int     `json:"day"`
	YearUnknown     bool    `json:"yearUnknown"`
	BirthYear       *int    `json:"birthYear"`
	ReminderOffset  *int    `json:"reminderOffset"`
	CalendarEventID *int64  `json:"-"`
	CreatedAt       int64   `json:"createdAt"`
	UpdatedAt       int64   `json:"updatedAt"`
	NextDate        *string `json:"nextDate,omitempty"`
	DaysUntil       *int    `json:"daysUntil,omitempty"`
	Age             *int    `json:"age,omitempty"`
}

// BirthdayOccurrence 是某个日期范围内的一次生日发生。
type BirthdayOccurrence struct {
	Date     string `json:"date"`
	Calendar string `json:"calendar"`
	Age      *int   `json:"age,omitempty"`
}

// Birthday 是生日档案，可同时包含公历与农历日期。
type Birthday struct {
	ID           int64                `json:"id"`
	Name         string               `json:"name"`
	Note         *string              `json:"note"`
	AvatarKey    *string              `json:"avatarKey"`
	ContactID    *int64               `json:"contactId"`
	UserID       *int64               `json:"userId"`
	CreatedBy    int64                `json:"createdBy"`
	CreatedAt    int64                `json:"createdAt"`
	UpdatedAt    int64                `json:"updatedAt"`
	Dates        []BirthdayDate       `json:"dates"`
	NextDate     *string              `json:"nextDate,omitempty"`
	DaysUntil    *int                 `json:"daysUntil,omitempty"`
	Age          *int                 `json:"age,omitempty"`
	NextCalendar *string              `json:"nextCalendar,omitempty"`
	Occurrences  []BirthdayOccurrence `json:"occurrences,omitempty"`
}

// MentionTarget 是被 @ 的对象，可为家庭成员（user）或联系人（contact）。
type MentionTarget struct {
	Type      string  `json:"type"`
	ID        int64   `json:"id"`
	Name      string  `json:"name,omitempty"`
	AvatarKey *string `json:"avatarKey,omitempty"`
}

// MessagePost 是留言板主贴，含回复与提及信息。
type MessagePost struct {
	ID             int64           `json:"id"`
	Body           string          `json:"body"`
	AuthorID       int64           `json:"authorId"`
	AuthorName     string          `json:"authorName,omitempty"`
	OccurrenceDate *string         `json:"occurrenceDate"`
	Pinned         bool            `json:"pinned"`
	Visibility     string          `json:"visibility"`
	EditedAt       *int64          `json:"editedAt"`
	ClosedAt       *int64          `json:"closedAt"`
	CreatedAt      int64           `json:"createdAt"`
	UpdatedAt      int64           `json:"updatedAt"`
	AudienceIDs    []int64         `json:"audienceIds"`
	Mentions       []MentionTarget `json:"mentions"`
	Replies        []MessageReply  `json:"replies"`
	ReplyCount     int             `json:"replyCount"`
}

// MessageReply 是留言下的回复。
type MessageReply struct {
	ID         int64  `json:"id"`
	PostID     int64  `json:"postId"`
	AuthorID   int64  `json:"authorId"`
	AuthorName string `json:"authorName,omitempty"`
	Body       string `json:"body"`
	CreatedAt  int64  `json:"createdAt"`
}

// NoteItem 是便签的检查清单项。
type NoteItem struct {
	ID       int64  `json:"id"`
	NoteID   int64  `json:"noteId,omitempty"`
	Text     string `json:"text"`
	Checked  bool   `json:"checked"`
	Position int    `json:"position"`
}

// Note 是家庭便签。
type Note struct {
	ID          int64           `json:"id"`
	Title       *string         `json:"title"`
	Body        *string         `json:"body"`
	Color       *string         `json:"color"`
	Pinned      bool            `json:"pinned"`
	Category    *string         `json:"category"`
	AuthorID    int64           `json:"authorId"`
	AuthorName  string          `json:"authorName,omitempty"`
	MemoryDate  *string         `json:"memoryDate"`
	Visibility  string          `json:"visibility"`
	ClosedAt    *int64          `json:"closedAt"`
	AudienceIDs []int64         `json:"audienceIds"`
	Mentions    []MentionTarget `json:"mentions"`
	Items       []NoteItem      `json:"items"`
	CreatedAt   int64           `json:"createdAt"`
	UpdatedAt   int64           `json:"updatedAt"`
}

// Notification 是个人收件箱消息。
type Notification struct {
	ID             int64   `json:"id"`
	UserID         int64   `json:"userId"`
	SourceType     string  `json:"sourceType"`
	SourceID       int64   `json:"sourceId"`
	Summary        *string `json:"summary"`
	ActorID        *int64  `json:"actorId"`
	ActorName      string  `json:"actorName,omitempty"`
	OccurrenceDate *string `json:"occurrenceDate"`
	CreatedAt      int64   `json:"createdAt"`
	ReadAt         *int64  `json:"readAt"`
	InvalidatedAt  *int64  `json:"invalidatedAt"`
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
