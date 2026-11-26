import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { api, ApiError } from '@/api/client';
import type {
  DashboardCardPayload,
  DashboardCardState,
  DashboardLayoutItem,
  DashboardLayoutPayload,
  DashboardLayoutSource,
} from '@/types';

interface LayoutResponse {
  layout: DashboardLayoutItem[];
  layoutSource: DashboardLayoutSource;
}

/** 上次成功加载的布局，用于数据到达前按真实位置渲染占位卡片。 */
const LAYOUT_CACHE_KEY = 'mymoments.dashboard.layout';

function readCachedLayout(): DashboardLayoutItem[] {
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(LAYOUT_CACHE_KEY) ?? '[]');
    return Array.isArray(parsed) ? (parsed as DashboardLayoutItem[]) : [];
  } catch {
    return [];
  }
}

function writeCachedLayout(items: DashboardLayoutItem[]): void {
  try {
    localStorage.setItem(LAYOUT_CACHE_KEY, JSON.stringify(items));
  } catch {
    // 隐私模式或配额不足时忽略
  }
}

export const useDashboardStore = defineStore('dashboard', () => {
  const layout = ref<DashboardLayoutItem[]>(readCachedLayout());
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
      const res = await api.get<DashboardLayoutPayload>('/dashboard/layout');
      if (mine !== token) return;
      layout.value = res.layout;
      writeCachedLayout(res.layout);
      layoutSource.value = res.layoutSource ?? 'default';
      today.value = res.today ?? '';
      // 先按布局占位，每张卡片各自独立请求，谁先返回谁先展示。
      // 刷新时保留已有数据，避免整版卡片闪回骨架屏。
      const previous = cardMap.value;
      cards.value = res.layout
        .filter((item) => item.visible)
        .map((item) => {
          const prev = previous[item.id];
          return {
            ...item,
            error: prev?.error ?? null,
            data: prev?.data ?? null,
            loading: !prev?.data,
          };
        });
      for (const item of res.layout) {
        if (item.visible) void loadCard(item.id, mine);
      }
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

  async function loadCard(id: string, mine: number): Promise<void> {
    try {
      const res = await api.get<DashboardCardPayload>(`/dashboard/cards/${id}`);
      if (mine !== token) return;
      if (res.error) {
        // 已有数据时静默保留，避免单张卡片刷新失败把已展示内容替换成错误。
        if (cardMap.value[id]?.data != null) {
          updateCard(id, { error: null, loading: false });
          return;
        }
        updateCard(id, { data: null, error: res.error, loading: false });
        return;
      }
      updateCard(id, { data: res.data ?? null, error: null, loading: false });
    } catch (err) {
      if (mine !== token) return;
      if (cardMap.value[id]?.data != null) {
        updateCard(id, { error: null, loading: false });
        return;
      }
      updateCard(id, {
        error: err instanceof ApiError ? err.message : '内容加载失败',
        loading: false,
      });
    }
  }

  function updateCard(id: string, patch: Partial<DashboardCardState>): void {
    const index = cards.value.findIndex((card) => card.id === id);
    if (index < 0) return;
    cards.value[index] = { ...cards.value[index], ...patch };
  }

  async function saveLayout(items: DashboardLayoutItem[]): Promise<void> {
    saving.value = true;
    try {
      const res = await api.put<LayoutResponse>('/dashboard/layout', { layout: items });
      layout.value = res.layout;
      writeCachedLayout(res.layout);
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
      writeCachedLayout(res.layout);
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
    writeCachedLayout([]);
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
