import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { api, request } from '@/api/client';
import type { Birthday, CalendarEvent } from '@/types';

export interface EventPayload {
  title?: string;
  description?: string | null;
  startLocal?: string;
  endLocal?: string | null;
  allDay?: boolean;
  location?: string | null;
  color?: string | null;
  icon?: string | null;
  recurrenceRule?: string | null;
  visibility?: string;
  assigneeIds?: number[];
  reminderOffsets?: number[];
  scope?: 'all' | 'this' | 'following';
  occurrenceDate?: string;
}

export interface BirthdayPayload {
  name?: string;
  month?: number;
  day?: number;
  yearUnknown?: boolean;
  birthYear?: number | null;
  note?: string | null;
  avatarKey?: string | null;
  contactId?: number | null;
  userId?: number | null;
  reminderOffset?: number | null;
}

export const useCalendarStore = defineStore('calendar', () => {
  const events = ref<CalendarEvent[]>([]);
  const birthdays = ref<Birthday[]>([]);
  const range = ref({ from: '', to: '' });
  const loading = ref(false);
  const error = ref<unknown>(null);
  const birthdaysLoaded = ref(false);

  const birthdayCount = computed(() => birthdays.value.length);

  async function loadRange(from: string, to: string): Promise<CalendarEvent[]> {
    loading.value = true;
    error.value = null;
    try {
      const res = await api.get<{ items: CalendarEvent[]; from: string; to: string }>('/events', { from, to });
      events.value = res.items;
      range.value = { from: res.from, to: res.to };
      return res.items;
    } catch (err) {
      error.value = err;
      throw err;
    } finally {
      loading.value = false;
    }
  }

  async function loadBirthdays(force = false): Promise<Birthday[]> {
    if (birthdaysLoaded.value && !force) return birthdays.value;
    const res = await api.get<{ items: Birthday[] }>('/birthdays');
    birthdays.value = res.items;
    birthdaysLoaded.value = true;
    return res.items;
  }

  async function getEvent(id: number): Promise<CalendarEvent> {
    const res = await api.get<{ event: CalendarEvent }>(`/events/${id}`);
    return res.event;
  }

  async function createEvent(payload: EventPayload): Promise<CalendarEvent> {
    const res = await api.post<{ event: CalendarEvent }>('/events', payload);
    return res.event;
  }

  async function updateEvent(id: number, payload: EventPayload): Promise<CalendarEvent> {
    const res = await api.patch<{ event: CalendarEvent }>(`/events/${id}`, payload);
    return res.event;
  }

  async function deleteEvent(id: number, scope: 'all' | 'this' | 'following' = 'all', occurrenceDate?: string): Promise<void> {
    await request(`/events/${id}`, {
      method: 'DELETE',
      query: { scope, occurrence: occurrenceDate },
    });
  }

  async function createBirthday(payload: BirthdayPayload): Promise<Birthday> {
    const res = await api.post<{ birthday: Birthday }>('/birthdays', payload);
    birthdays.value = sortBirthdays([...birthdays.value, res.birthday]);
    return res.birthday;
  }

  async function updateBirthday(id: number, payload: BirthdayPayload): Promise<Birthday> {
    const res = await api.patch<{ birthday: Birthday }>(`/birthdays/${id}`, payload);
    birthdays.value = sortBirthdays(birthdays.value.map((b) => (b.id === id ? res.birthday : b)));
    return res.birthday;
  }

  async function deleteBirthday(id: number): Promise<void> {
    await api.del(`/birthdays/${id}`);
    birthdays.value = birthdays.value.filter((b) => b.id !== id);
  }

  function sortBirthdays(items: Birthday[]): Birthday[] {
    return [...items].sort((a, b) => (a.daysUntil ?? 999) - (b.daysUntil ?? 999) || a.name.localeCompare(b.name, 'zh-CN'));
  }

  function reset(): void {
    events.value = [];
    birthdays.value = [];
    birthdaysLoaded.value = false;
    error.value = null;
  }

  return {
    events,
    birthdays,
    range,
    loading,
    error,
    birthdayCount,
    loadRange,
    loadBirthdays,
    getEvent,
    createEvent,
    updateEvent,
    deleteEvent,
    createBirthday,
    updateBirthday,
    deleteBirthday,
    reset,
  };
});
