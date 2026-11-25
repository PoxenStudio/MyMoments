export type Role = 'admin' | 'member';
export type Visibility = 'household' | 'selected' | 'private';
export type TaskStatus = 'todo' | 'in_progress' | 'completed';
export type Priority = 'low' | 'normal' | 'high' | 'urgent';

export interface User {
  id: number;
  username: string;
  displayName: string;
  avatarKey: string | null;
  role: Role;
  status: 'active' | 'inactive';
  mustChangePassword: boolean;
  relation: string | null;
  sortOrder: number;
  createdAt?: number;
}

export interface Household {
  name: string;
  timezone: string;
  weekStart: number;
  dateFormat: '24h' | '12h';
  defaultWeatherCity: string | null;
  defaultWeatherLat: number | null;
  defaultWeatherLon: number | null;
  defaultWeatherProvider: string | null;
  defaultReminderOffset: number;
  defaultDashboard: string | null;
  style: string | null;
}

export type ThemeTone =
  | 'default'
  | 'warm-sun' | 'warm-sand' | 'warm-peach' | 'warm-cream' | 'warm-rose'
  | 'cartoon-macaron' | 'cartoon-sky' | 'cartoon-mint' | 'cartoon-berry' | 'cartoon-lemon';

export type CardRadius = 'sm' | 'md' | 'lg' | 'xl';

export type ThemeScheme = 'light' | 'dark';

export type CardTone = 'default' | 'soft' | 'sand' | 'mint' | 'peach' | 'lavender' | 'sky' | 'rose';

export interface ThemeBackground {
  kind: 'color' | 'image';
  color?: string;
  image?: string;
  blur?: number;
}

export interface ThemeCorner {
  image: string | null;
  opacity: number;
}

export interface ThemeConfig {
  useDefault?: boolean;
  scheme?: ThemeScheme;
  cardTone?: CardTone;
  tone: ThemeTone;
  cardRadius: CardRadius;
  pageBackground: ThemeBackground;
  cardCorner: ThemeCorner;
}

export type ThemePatch = Partial<Omit<ThemeConfig, 'pageBackground' | 'cardCorner'>> & {
  pageBackground?: Partial<ThemeBackground>;
  cardCorner?: Partial<ThemeCorner>;
};

export interface Preferences {
  locale: string;
  dateFormat: '24h' | '12h';
  dashboard: string | null;
  weatherCity: string | null;
  weatherLat: number | null;
  weatherLon: number | null;
  weatherAuto: boolean;
  notificationPrefs: string | null;
  style: string | null;
  updatedAt: number;
}

export interface Bootstrap {
  needsSetup: boolean;
  household?: Household;
  user?: User | null;
}

export interface Task {
  id: number;
  title: string;
  description: string | null;
  status: TaskStatus;
  priority: Priority;
  startDate: string | null;
  dueDate: string | null;
  dueTime: string | null;
  templateId: number | null;
  recurrenceRule: string | null;
  seriesId: string | null;
  occurrenceDate: string | null;
  color: string | null;
  icon: string | null;
  visibility: Visibility;
  completedAt: number | null;
  archivedAt: number | null;
  createdBy: number;
  createdAt: number;
  updatedAt: number;
  assignees: User[];
  subtasks: Subtask[];
  mediaCount?: number;
  completion?: Completion | null;
  reminderOffsets?: number[];
  mentions?: MentionTarget[];
}

export interface Subtask {
  id: number;
  title: string;
  done: boolean;
  position: number;
}

export interface Completion {
  id: number;
  taskId: number;
  occurrenceDate: string;
  doneByUserId: number | null;
  actualUserId: number | null;
  completedAt: number;
  doneByName?: string;
  actualName?: string;
}

export interface TaskTemplate {
  id: number;
  name: string;
  title: string;
  description: string | null;
  icon: string | null;
  color: string | null;
  priority: Priority | null;
  defaultAssigneeIds: number[];
  defaultDueTime: string | null;
  recurrenceRule: string | null;
  reminderOffset: number | null;
  subtaskTemplate: string[];
  enabled: boolean;
  sortOrder: number;
}

export interface CalendarEvent {
  id: number;
  title: string;
  description: string | null;
  startLocal: string;
  endLocal: string | null;
  allDay: boolean;
  location: string | null;
  color: string | null;
  icon: string | null;
  recurrenceRule: string | null;
  seriesId: string | null;
  occurrenceDate?: string;
  visibility: Visibility;
  source: 'manual' | 'birthday';
  createdBy: number;
  createdAt: number;
  updatedAt: number;
  assignees: User[];
  reminderOffsets?: number[];
}

export interface Birthday {
  id: number;
  name: string;
  month: number;
  day: number;
  yearUnknown: boolean;
  birthYear: number | null;
  note: string | null;
  avatarKey: string | null;
  contactId: number | null;
  userId: number | null;
  reminderOffset: number | null;
  createdBy?: number;
  daysUntil?: number;
  age?: number | null;
}

export interface ContactItem {
  id?: number;
  kind: 'phone' | 'email' | 'address';
  label: string | null;
  value: string;
  isPrimaryCall?: boolean;
  isPrimarySms?: boolean;
}

export interface ContactRelation {
  contactId?: number;
  relatedContactId: number;
  relation: string;
  relatedName?: string;
}

export interface ContactReminder {
  id?: number;
  kind: 'spring_festival' | 'traditional' | 'western' | 'custom';
  title?: string;
  frequency: '' | 'monthly' | 'weekly' | 'once';
  dueDate: string | null;
  recurrenceRule?: string | null;
  color?: string | null;
  enabled: boolean;
}

export interface Contact {
  id: number;
  name: string;
  organization: string | null;
  title: string | null;
  note: string | null;
  birthdayMonth: number | null;
  birthdayDay: number | null;
  birthYear: number | null;
  showInCalendar: boolean;
  pinned: boolean;
  category: string | null;
  avatarKey: string | null;
  wechat: string | null;
  officePhone: string | null;
  ownerScope: 'household' | 'private';
  createdBy: number;
  items: ContactItem[];
  relations: ContactRelation[];
  reminders: ContactReminder[];
  createdAt: number;
  updatedAt: number;
}

export interface Note {
  id: number;
  title: string | null;
  body: string | null;
  color: string | null;
  pinned: boolean;
  category: string | null;
  authorId: number;
  authorName?: string;
  memoryDate: string | null;
  visibility: Visibility;
  audienceIds: number[];
  mentions: MentionTarget[];
  closedAt: number | null;
  createdAt: number;
  updatedAt: number;
  items: NoteItem[];
}

export interface NoteItem {
  id: number;
  noteId?: number;
  text: string;
  checked: boolean;
  position: number;
}

export interface MentionTarget {
  type: 'user' | 'contact';
  id: number;
  name?: string;
  avatarKey?: string | null;
}

export interface MessageReply {
  id: number;
  postId: number;
  authorId: number;
  authorName?: string;
  body: string;
  createdAt: number;
}

export interface MessagePost {
  id: number;
  body: string;
  authorId: number;
  authorName?: string;
  occurrenceDate: string | null;
  pinned: boolean;
  visibility: Visibility;
  editedAt: number | null;
  closedAt: number | null;
  createdAt: number;
  updatedAt: number;
  audienceIds: number[];
  mentions: MentionTarget[];
  replies: MessageReply[];
  replyCount?: number;
}

export interface Reminder {
  id: number;
  userId: number;
  sourceType: 'task' | 'event' | 'birthday';
  sourceId: number;
  occurrenceDate: string | null;
  title: string | null;
  remindAt: number;
  status: 'pending' | 'delivered' | 'dismissed';
  deliveredAt: number | null;
}

export interface NotificationItem {
  id: number;
  userId: number;
  sourceType: string;
  sourceId: number;
  summary: string | null;
  actorId: number | null;
  actorName?: string;
  occurrenceDate: string | null;
  createdAt: number;
  readAt: number | null;
  invalidatedAt: number | null;
}

export interface TimelineEntry {
  sourceType: 'task' | 'task_completion' | 'event' | 'birthday' | 'message' | 'note';
  sourceId: number;
  occurrenceKey: string;
  occurredOn: string;
  occurredAt: number | null;
  title: string;
  summary: string | null;
  memberIds: number[];
  members: User[];
  status: string | null;
  mediaPreview: MediaItem[];
  detailTarget: { view: string; open: number; date: string };
}

export interface MediaItem {
  id: number;
  taskId: number;
  occurrenceDate: string | null;
  mediaType: 'image' | 'audio' | 'video';
  mimeType: string;
  sizeBytes: number;
  originalName: string | null;
  durationMs: number | null;
  createdAt: number;
}

export type TaskMedia = MediaItem;

export interface DashboardCardState {
  id: string;
  title: string;
  visible: boolean;
  size: 'standard' | 'wide' | 'tall';
  background: string;
  error: string | null;
  data: unknown;
}

export interface DashboardLayoutItem {
  id: string;
  visible: boolean;
  size: 'standard' | 'wide' | 'tall';
  background: string;
}

export type DashboardLayoutSource = 'personal' | 'household' | 'default';

export interface DashboardPayload {
  layout: DashboardLayoutItem[];
  layoutSource?: DashboardLayoutSource;
  cards: DashboardCardState[];
  today: string;
}

export interface WeatherForecastDay {
  date: string;
  max: number;
  min: number;
  code: number;
  description: string;
}

export interface WeatherData {
  source: string;
  city: string;
  temperature: number;
  apparent: number | null;
  description: string;
  code: number;
  high: number | null;
  low: number | null;
  updatedAt: number;
  stale: boolean;
  forecast: WeatherForecastDay[];
}

export interface WeatherSource {
  id: string;
  name: string;
  note: string;
  available: boolean;
  default: boolean;
}

export interface CalendarDayMeta {
  date: string;
  lunar: string;
  lunarMonth: number;
  lunarDay: number;
  lunarLeap: boolean;
  festival?: string;
  holiday?: string;
  offDay: boolean;
  workday: boolean;
}

export interface ApiToken {
  id: number;
  name: string;
  prefix: string;
  scopes: Record<string, string[]>;
  expiresAt: number | null;
  lastUsedAt: number | null;
  createdAt: number;
  revokedAt: number | null;
}

export interface SearchResult {
  type: 'task' | 'event' | 'contact' | 'message' | 'note';
  id: number;
  title: string;
  summary: string | null;
  date: string | null;
}
