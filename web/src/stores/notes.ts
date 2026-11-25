import { defineStore } from 'pinia';
import { ref } from 'vue';
import { api } from '@/api/client';
import type { MentionTarget, Note, NoteItem, Visibility } from '@/types';

export interface NoteQuery {
  q?: string;
  category?: string;
  pinned?: boolean;
}

export interface NotePayload {
  title?: string | null;
  body?: string | null;
  color?: string | null;
  category?: string | null;
  memoryDate?: string | null;
  visibility?: Visibility;
  audienceIds?: number[];
  mentions?: MentionTarget[];
  pinned?: boolean;
  items?: Pick<NoteItem, 'text' | 'checked'>[];
}

export const useNotesStore = defineStore('notes', () => {
  const items = ref<Note[]>([]);
  const loading = ref(false);
  const loaded = ref(false);
  const error = ref<unknown>(null);
  let lastQuery = '';

  function sort(notes: Note[]): Note[] {
    return [...notes].sort((a, b) => {
      if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
      return b.updatedAt - a.updatedAt;
    });
  }

  async function load(query: NoteQuery = {}, force = false): Promise<Note[]> {
    const key = JSON.stringify(query);
    if (loaded.value && !force && key === lastQuery) return items.value;
    loading.value = true;
    error.value = null;
    try {
      const res = await api.get<{ items: Note[] }>('/notes', {
        q: query.q,
        category: query.category,
        pinned: query.pinned ? 1 : undefined,
      });
      items.value = res.items;
      loaded.value = true;
      lastQuery = key;
      return res.items;
    } catch (err) {
      error.value = err;
      throw err;
    } finally {
      loading.value = false;
    }
  }

  async function create(payload: NotePayload): Promise<Note> {
    const res = await api.post<{ note: Note }>('/notes', payload);
    items.value = sort([res.note, ...items.value]);
    return res.note;
  }

  async function update(id: number, payload: NotePayload): Promise<Note> {
    const res = await api.patch<{ note: Note }>(`/notes/${id}`, payload);
    items.value = sort(items.value.map((n) => (n.id === id ? res.note : n)));
    return res.note;
  }

  async function remove(id: number): Promise<void> {
    await api.del(`/notes/${id}`);
    items.value = items.value.filter((n) => n.id !== id);
  }

  async function setPinned(id: number, pinned: boolean): Promise<Note> {
    return update(id, { pinned });
  }

  function reset(): void {
    items.value = [];
    loaded.value = false;
    lastQuery = '';
    error.value = null;
  }

  return { items, loading, loaded, error, load, create, update, remove, setPinned, reset };
});
