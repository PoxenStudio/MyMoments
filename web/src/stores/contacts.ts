import { defineStore } from 'pinia';
import { ref } from 'vue';
import { api } from '@/api/client';
import type { Contact, ContactItem, ContactRelation, ContactReminder } from '@/types';

export interface ContactPayload {
  name?: string;
  organization?: string | null;
  title?: string | null;
  note?: string | null;
  birthdayMonth?: number | null;
  birthdayDay?: number | null;
  birthYear?: number | null;
  showInCalendar?: boolean;
  category?: string | null;
  avatarKey?: string | null;
  wechat?: string | null;
  officePhone?: string | null;
  ownerScope?: 'household' | 'private';
  items?: ContactItem[];
  relations?: ContactRelation[];
  reminders?: ContactReminder[];
}

export interface ContactAssociations {
  tasks: number;
  messages: number;
  notes: number;
}

export interface BlockedContact {
  id: number;
  name: string;
  counts: ContactAssociations;
}

export interface CleanupResult {
  deleted: number;
  blocked: BlockedContact[];
}

function sortByName(list: Contact[]): Contact[] {
  return [...list].sort(
    (a, b) => Number(b.pinned) - Number(a.pinned) || a.name.localeCompare(b.name, 'zh-CN'),
  );
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
      items.value = sortByName(res.items);
      loaded.value = true;
      return items.value;
    } catch (err) {
      error.value = err;
      throw err;
    } finally {
      loading.value = false;
    }
  }

  async function get(id: number): Promise<Contact> {
    const res = await api.get<{ contact: Contact }>(`/contacts/${id}`);
    return res.contact;
  }

  async function create(payload: ContactPayload): Promise<Contact> {
    const res = await api.post<{ contact: Contact }>('/contacts', payload);
    items.value = sortByName([...items.value, res.contact]);
    return res.contact;
  }

  async function update(id: number, payload: ContactPayload): Promise<Contact> {
    const res = await api.patch<{ contact: Contact }>(`/contacts/${id}`, payload);
    items.value = sortByName(items.value.map((c) => (c.id === id ? res.contact : c)));
    return res.contact;
  }

  async function setPinned(id: number, pinned: boolean): Promise<Contact> {
    const res = await api.put<{ contact: Contact }>(`/contacts/${id}/pinned`, { pinned });
    items.value = sortByName(items.value.map((c) => (c.id === id ? res.contact : c)));
    return res.contact;
  }

  async function remove(id: number): Promise<void> {
    await api.del(`/contacts/${id}`);
    items.value = items.value.filter((c) => c.id !== id);
  }

  async function associations(id: number): Promise<ContactAssociations> {
    const res = await api.get<{ counts: ContactAssociations; hasAssociations: boolean }>(`/contacts/${id}/associations`);
    return res.counts;
  }

  async function cleanup(): Promise<CleanupResult> {
    const res = await api.post<CleanupResult>('/contacts/cleanup');
    if (res.deleted > 0) items.value = [];
    return res;
  }

  async function importFile(file: File): Promise<{ created: number }> {
    const form = new FormData();
    form.append('file', file);
    return await api.upload<{ created: number }>('/contacts/import', form);
  }

  function reset(): void {
    items.value = [];
    loaded.value = false;
    error.value = null;
  }

  return { items, loading, loaded, error, load, get, create, update, setPinned, remove, associations, cleanup, importFile, reset };
});
