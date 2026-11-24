import { defineStore } from 'pinia';
import { ref } from 'vue';
import { api } from '@/api/client';
import type { TimelineEntry } from '@/types';

export interface TimelineParams {
  anchor: string;
  direction: 'past' | 'future';
  types?: string[];
  member?: number;
  q?: string;
  limit?: number;
}

interface TimelineResponse {
  items: TimelineEntry[];
  nextCursor: string;
  hasMore: boolean;
  partial: string[];
}

export const useTimelineStore = defineStore('timeline', () => {
  const items = ref<TimelineEntry[]>([]);
  const loading = ref(false);
  const loadingMore = ref(false);
  const error = ref<unknown>(null);
  const nextCursor = ref('');
  const hasMore = ref(false);
  const partial = ref<string[]>([]);
  let lastParams: TimelineParams | null = null;
  let token = 0;

  function query(params: TimelineParams, cursor?: string): Record<string, string | number | undefined> {
    return {
      anchor: params.anchor,
      direction: params.direction,
      types: params.types && params.types.length ? params.types.join(',') : undefined,
      member: params.member || undefined,
      q: params.q || undefined,
      limit: params.limit,
      cursor: cursor || undefined,
    };
  }

  async function load(params: TimelineParams): Promise<void> {
    lastParams = params;
    const mine = ++token;
    loading.value = true;
    error.value = null;
    try {
      const res = await api.get<TimelineResponse>('/timeline', query(params));
      if (mine !== token) return;
      items.value = res.items;
      nextCursor.value = res.nextCursor;
      hasMore.value = res.hasMore;
      partial.value = res.partial ?? [];
    } catch (err) {
      if (mine === token) error.value = err;
      throw err;
    } finally {
      if (mine === token) loading.value = false;
    }
  }

  async function loadMore(): Promise<void> {
    if (!lastParams || !hasMore.value || loadingMore.value) return;
    const mine = token;
    loadingMore.value = true;
    try {
      const res = await api.get<TimelineResponse>('/timeline', query(lastParams, nextCursor.value));
      if (mine !== token) return;
      items.value = [...items.value, ...res.items];
      nextCursor.value = res.nextCursor;
      hasMore.value = res.hasMore;
      partial.value = res.partial ?? [];
    } catch (err) {
      if (mine === token) error.value = err;
      throw err;
    } finally {
      if (mine === token) loadingMore.value = false;
    }
  }

  function reset(): void {
    token += 1;
    items.value = [];
    nextCursor.value = '';
    hasMore.value = false;
    partial.value = [];
    error.value = null;
    loading.value = false;
    loadingMore.value = false;
    lastParams = null;
  }

  return { items, loading, loadingMore, error, nextCursor, hasMore, partial, load, loadMore, reset };
});
