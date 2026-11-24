import { defineStore } from 'pinia';
import { ref } from 'vue';
import { api } from '@/api/client';
import type { Contact, ContactItem } from '@/types';

export interface ContactPayload {
  name?: string;
  organization?: string | null;
  title?: string | null;
  note?: string | null;
  birthdayOn?: string | null;
  category?: string | null;
  avatarKey?: string | null;
  ownerScope?: 'household' | 'private';
  items?: ContactItem[];
}

export const useContactsStore = defineStore('contacts', () => {
  const items = ref<Contact[]>([]);
  const loading = ref(false);
  const loaded = ref(false);
  const error = ref<unknown>(null);

  async function load(query: { q?: string; category?: string } = {}, force = false): Promise<Contact[]> {
    if (loaded.value && !force && !query.q && !query.category) return items.value;
    loading.value = true;
    error.value = null;
    try {
      const res = await api.get<{ items: Contact[] }>('/contacts', { q: query.q, category: query.category });
      items.value = res.items;
      loaded.value = true;
      return res.items;
    } catch (err) {
      error.value = err;
      throw err;
    } finally {
      loading.value = false;
    }
  }

  async function create(payload: ContactPayload): Promise<Contact> {
    const res = await api.post<{ contact: Contact }>('/contacts', payload);
    items.value = [...items.value, res.contact].sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'));
    return res.contact;
  }

  async function update(id: number, payload: ContactPayload): Promise<Contact> {
    const res = await api.patch<{ contact: Contact }>(`/contacts/${id}`, payload);
    items.value = items.value
      .map((c) => (c.id === id ? res.contact : c))
      .sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'));
    return res.contact;
  }

  async function remove(id: number): Promise<void> {
    await api.del(`/contacts/${id}`);
    items.value = items.value.filter((c) => c.id !== id);
  }

  function reset(): void {
    items.value = [];
    loaded.value = false;
    error.value = null;
  }

  return { items, loading, loaded, error, load, create, update, remove, reset };
});
