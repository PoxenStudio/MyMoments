import { defineStore } from 'pinia';
import { ref } from 'vue';
import { api } from '@/api/client';
import type { MessagePost, MessageReply, MentionTarget, Visibility } from '@/types';

export interface MessageQuery {
  q?: string;
  filter?: 'mentions' | 'mine';
  pinned?: boolean;
}

export interface MessagePayload {
  body?: string;
  occurrenceDate?: string | null;
  visibility?: Visibility;
  audienceIds?: number[];
  mentions?: MentionTarget[];
  pinned?: boolean;
}

export const useMessagesStore = defineStore('messages', () => {
  const items = ref<MessagePost[]>([]);
  const loading = ref(false);
  const loaded = ref(false);
  const error = ref<unknown>(null);
  let lastQuery = '';

  async function load(query: MessageQuery = {}, force = false): Promise<MessagePost[]> {
    const key = JSON.stringify(query);
    if (loaded.value && !force && key === lastQuery) return items.value;
    loading.value = true;
    error.value = null;
    try {
      const res = await api.get<{ items: MessagePost[] }>('/messages', {
        q: query.q,
        filter: query.filter,
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

  async function create(payload: MessagePayload): Promise<MessagePost> {
    const res = await api.post<{ message: MessagePost }>('/messages', payload);
    items.value = [res.message, ...items.value];
    return res.message;
  }

  async function update(id: number, payload: MessagePayload): Promise<MessagePost> {
    const res = await api.patch<{ message: MessagePost }>(`/messages/${id}`, payload);
    items.value = items.value.map((m) => (m.id === id ? res.message : m));
    return res.message;
  }

  async function remove(id: number): Promise<void> {
    await api.del(`/messages/${id}`);
    items.value = items.value.filter((m) => m.id !== id);
  }

  async function reply(id: number, body: string, mentions: MentionTarget[]): Promise<MessageReply> {
    const res = await api.post<{ reply: MessageReply }>(`/messages/${id}/replies`, { body, mentions });
    items.value = items.value.map((m) => (m.id === id
      ? { ...m, replies: [...m.replies, res.reply], replyCount: (m.replyCount ?? m.replies.length) + 1 }
      : m));
    return res.reply;
  }

  async function setPinned(id: number, pinned: boolean): Promise<MessagePost> {
    return update(id, { pinned });
  }

  function reset(): void {
    items.value = [];
    loaded.value = false;
    lastQuery = '';
    error.value = null;
  }

  return { items, loading, loaded, error, load, create, update, remove, reply, setPinned, reset };
});
