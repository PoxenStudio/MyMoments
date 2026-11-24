import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { api } from '@/api/client';
import type { NotificationItem, User } from '@/types';

export const useFamilyStore = defineStore('family', () => {
  const members = ref<User[]>([]);
  const loaded = ref(false);

  const activeMembers = computed(() => members.value.filter((m) => m.status === 'active'));

  async function load(force = false): Promise<User[]> {
    if (loaded.value && !force) return members.value;
    const res = await api.get<{ items: User[] }>('/family/members');
    members.value = res.items;
    loaded.value = true;
    return members.value;
  }

  function memberName(id: number | null | undefined): string {
    if (!id) return '已停用成员';
    return members.value.find((m) => m.id === id)?.displayName ?? `成员 ${id}`;
  }

  function reset(): void {
    members.value = [];
    loaded.value = false;
  }

  return { members, activeMembers, loaded, load, memberName, reset };
});

export const useNotificationStore = defineStore('notification', () => {
  const items = ref<NotificationItem[]>([]);
  const unread = computed(() => items.value.filter((n) => !n.readAt && !n.invalidatedAt).length);
  const loaded = ref(false);

  async function load(): Promise<void> {
    const res = await api.get<{ items: NotificationItem[] }>('/notifications');
    items.value = res.items;
    loaded.value = true;
  }

  async function markRead(id: number): Promise<void> {
    await api.patch(`/notifications/${id}/read`);
    const item = items.value.find((n) => n.id === id);
    if (item) item.readAt = Math.floor(Date.now() / 1000);
  }

  async function markAllRead(): Promise<void> {
    await api.post('/notifications/read-all');
    const now = Math.floor(Date.now() / 1000);
    items.value = items.value.map((n) => (n.readAt ? n : { ...n, readAt: now }));
  }

  function reset(): void {
    items.value = [];
    loaded.value = false;
  }

  return { items, unread, loaded, load, markRead, markAllRead, reset };
});
