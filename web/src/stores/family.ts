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

  async function createMember(payload: {
    username: string;
    displayName: string;
    role: string;
    relation: string;
    password: string;
  }): Promise<{ user: User; temporaryPassword: string }> {
    const res = await api.post<{ user: User; temporaryPassword: string }>('/family/members', payload);
    members.value = [...members.value, res.user];
    return res;
  }

  async function updateMember(id: number, payload: Partial<Pick<User, 'displayName' | 'relation' | 'role' | 'status'>>): Promise<User> {
    const res = await api.patch<{ user: User }>(`/family/members/${id}`, payload);
    members.value = members.value.map((m) => (m.id === id ? res.user : m));
    return res.user;
  }

  async function resetMemberPassword(id: number, password = ''): Promise<string> {
    const res = await api.post<{ temporaryPassword: string }>(`/family/members/${id}/reset-password`, { password });
    members.value = members.value.map((m) => (m.id === id ? { ...m, mustChangePassword: true } : m));
    return res.temporaryPassword;
  }

  function reset(): void {
    members.value = [];
    loaded.value = false;
  }

  return { members, activeMembers, loaded, load, memberName, createMember, updateMember, resetMemberPassword, reset };
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
