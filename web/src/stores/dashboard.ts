import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { api } from '@/api/client';
import type {
  DashboardCardState,
  DashboardLayoutItem,
  DashboardLayoutSource,
  DashboardPayload,
} from '@/types';

interface LayoutResponse {
  layout: DashboardLayoutItem[];
  layoutSource: DashboardLayoutSource;
}

export const useDashboardStore = defineStore('dashboard', () => {
  const layout = ref<DashboardLayoutItem[]>([]);
  const layoutSource = ref<DashboardLayoutSource>('default');
  const cards = ref<DashboardCardState[]>([]);
  const today = ref('');
  const loading = ref(false);
  const refreshing = ref(false);
  const saving = ref(false);
  const error = ref<unknown>(null);
  let token = 0;

  const cardMap = computed<Record<string, DashboardCardState>>(() => {
    const map: Record<string, DashboardCardState> = {};
    for (const card of cards.value) map[card.id] = card;
    return map;
  });

  const visibleItems = computed(() => layout.value.filter((item) => item.visible));

  async function load(silent = false): Promise<void> {
    const mine = ++token;
    if (silent) refreshing.value = true;
    else loading.value = true;
    error.value = null;
    try {
      const res = await api.get<DashboardPayload>('/dashboard');
      if (mine !== token) return;
      layout.value = res.layout;
      layoutSource.value = res.layoutSource ?? 'default';
      cards.value = res.cards;
      today.value = res.today;
    } catch (err) {
      if (mine === token) error.value = err;
      throw err;
    } finally {
      if (mine === token) {
        loading.value = false;
        refreshing.value = false;
      }
    }
  }

  async function saveLayout(items: DashboardLayoutItem[]): Promise<void> {
    saving.value = true;
    try {
      const res = await api.put<LayoutResponse>('/dashboard/layout', { layout: items });
      layout.value = res.layout;
      layoutSource.value = res.layoutSource;
      await load(true);
    } catch (err) {
      error.value = err;
      throw err;
    } finally {
      saving.value = false;
    }
  }

  async function resetLayout(): Promise<void> {
    saving.value = true;
    try {
      const res = await api.put<LayoutResponse>('/dashboard/layout', { reset: true });
      layout.value = res.layout;
      layoutSource.value = res.layoutSource;
      await load(true);
    } catch (err) {
      error.value = err;
      throw err;
    } finally {
      saving.value = false;
    }
  }

  async function publishHouseholdLayout(items: DashboardLayoutItem[]): Promise<void> {
    saving.value = true;
    try {
      await api.put<LayoutResponse>('/household/default-layout', { layout: items });
    } catch (err) {
      error.value = err;
      throw err;
    } finally {
      saving.value = false;
    }
  }

  function reset(): void {
    token += 1;
    layout.value = [];
    cards.value = [];
    today.value = '';
    error.value = null;
    loading.value = false;
    refreshing.value = false;
  }

  return {
    layout,
    layoutSource,
    cards,
    today,
    loading,
    refreshing,
    saving,
    error,
    cardMap,
    visibleItems,
    load,
    saveLayout,
    resetLayout,
    publishHouseholdLayout,
    reset,
  };
});
