import type {
  Birthday,
  CalendarEvent,
  Contact,
  MessagePost,
  Note,
  Reminder,
  Task,
  TimelineEntry,
  User,
} from '@/types';

export interface TaskCounts {
  open: number;
  completed: number;
  archived: number;
  today: number;
  overdue: number;
}

export interface OverviewData {
  counts: TaskCounts;
  eventsToday: number;
  nextEvent: CalendarEvent | null;
  birthdays: Birthday[];
  remindersPending: number;
}

export interface TasksData {
  items: Task[];
  counts: TaskCounts;
}

export interface CalendarData {
  items: CalendarEvent[];
  from: string;
  to: string;
}

export interface TimelineData {
  items: TimelineEntry[];
}

export interface BirthdaysData {
  items: Birthday[];
}

export interface RemindersData {
  items: Reminder[];
}

export interface MessagesData {
  items: MessagePost[];
}

export interface NotesData {
  items: Note[];
}

export interface MembersData {
  items: User[];
}

export interface ContactsData {
  items: Contact[];
}

export interface PhotoAlbumItem {
  id: number;
  originalName: string | null;
  mimeType: string;
  sizeBytes: number;
  fit: 'cover' | 'contain';
  createdBy: number;
  createdAt: number;
}

export interface PhotoAlbumData {
  items: PhotoAlbumItem[];
}

export function asData<T>(raw: unknown): T | null {
  if (!raw || typeof raw !== 'object') return null;
  return raw as T;
}
