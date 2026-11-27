<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { RouterLink, useRoute, useRouter } from 'vue-router';
import { greeting, todayISO } from '@/utils/chrono';
import { useAuthStore } from '@/stores/auth';
import { useDashboardStore } from '@/stores/dashboard';
import { useToastStore } from '@/stores/toast';
import { api, ApiError } from '@/api/client';
import { cardRegistry } from '@/cards/registry';
import type { OverviewData } from '@/cards/types';
import type { DashboardLayoutItem, WeatherData } from '@/types';
import AppButton from '@/components/base/AppButton.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';
import AppPanel from '@/components/base/AppPanel.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import TaskFormDialog from '@/components/business/TaskFormDialog.vue';
import WeatherSettingsDialog from '@/components/business/WeatherSettingsDialog.vue';
import WeatherMiniCard from '@/components/business/WeatherMiniCard.vue';

/** 栅格参数：12 列、40px 行高，与后端保持一致的通用看板布局。 */
const GRID_COLUMNS = 12;
const GRID_ROW_HEIGHT = 40;
const GRID_GAP = 12;
const CARD_MIN_W = 3;
const CARD_MIN_H = 5;
const CARD_MAX_H = 60;
const GRID_MAX_ROWS = 200;
const STACK_BREAKPOINT = 900;

type DragMode = 'move' | 'resize';

interface DragState {
  id: string;
  pointerId: number;
  mode: DragMode;
  startX: number;
  startY: number;
  origin: DashboardLayoutItem;
  preview: DashboardLayoutItem;
}

const auth = useAuthStore();
const dashboard = useDashboardStore();
const route = useRoute();
const router = useRouter();
const toast = useToastStore();
const weather = ref<WeatherData | null>(null);

const formOpen = ref(false);
const weatherOpen = ref(false);
const resetOpen = ref(false);
const editing = ref(false);
const selectedId = ref('');
const gridRef = ref<HTMLElement | null>(null);
const homeRef = ref<HTMLElement | null>(null);
const isFullscreen = ref(false);
const drag = ref<DragState | null>(null);
let timer: number | undefined;
let saveQueue: Promise<void> = Promise.resolve();

const greetingText = computed(() => greeting(auth.user?.displayName ?? '家人'));
const dateText = computed(() => {
  const now = new Date();
  const date = new Intl.DateTimeFormat('zh-CN', { month: 'long', day: 'numeric' }).format(now);
  const weekday = new Intl.DateTimeFormat('zh-CN', { weekday: 'long' }).format(now);
  return `${date} · ${weekday}`;
});

const overview = computed<OverviewData | null>(() => {
  const data = dashboard.cardMap.overview?.data;
  if (!data || typeof data !== 'object') return null;
  return data as OverviewData;
});

const summary = computed(() => {
  if (editing.value) return '拖动卡片标题可移动位置，拖动右下角可调整大小，点击卡片设置底色。';
  if (dashboard.loading && !dashboard.cards.length) return '正在加载首页…';
  const counts = overview.value?.counts;
  if (!counts) return '看看今天家里有什么事。';
  if (counts.overdue > 0) return `有 ${counts.overdue} 件已经逾期。`;
  if (counts.today > 0) return `今天有 ${counts.today} 件待办。`;
  if (counts.open > 0) return `还有 ${counts.open} 件待办未完成。`;
  return '今天没有待办。';
});

/** 展示用卡片：普通模式只看可见卡片，编辑模式展示全部以便重新显示。 */
const displayItems = computed(() => {
  const items = dashboard.layout.filter((item) => !!cardRegistry[item.id]);
  const picked = editing.value ? items : items.filter((item) => item.visible);
  return [...picked].sort((a, b) => a.y - b.y || a.x - b.x);
});

/** 后端默认布局（internal/httpapi/handlers_dashboard.go 的紧凑排列），首次无缓存时用于占位。 */
const DEFAULT_LAYOUT: DashboardLayoutItem[] = [
  { id: 'overview', visible: true, background: 'default', x: 0, y: 0, w: 8, h: 6 },
  { id: 'tasks', visible: true, background: 'default', x: 8, y: 0, w: 4, h: 6 },
  { id: 'calendar', visible: true, background: 'default', x: 0, y: 6, w: 4, h: 6 },
  { id: 'birthdays', visible: true, background: 'default', x: 4, y: 6, w: 4, h: 6 },
  { id: 'reminders', visible: true, background: 'default', x: 8, y: 6, w: 4, h: 6 },
];

/** 加载中且无数据时展示占位卡片：优先用缓存布局，保证与真实位置一致。 */
const showSkeleton = computed(() => !dashboard.cards.length && !dashboard.error);

const placeholderItems = computed<DashboardLayoutItem[]>(() => {
  const cached = displayItems.value.length
    ? displayItems.value
    : dashboard.layout.filter((item) => item.visible && !!cardRegistry[item.id]);
  const items = cached.length ? cached : DEFAULT_LAYOUT;
  return [...items].sort((a, b) => a.y - b.y || a.x - b.x);
});

const gridRows = computed(() => {
  const items = displayItems.value.length ? displayItems.value : placeholderItems.value;
  let max = 0;
  for (const item of items) max = Math.max(max, item.y + item.h);
  return Math.max(max, editing.value ? 8 : 3);
});

const gridStyle = computed<Record<string, string>>(() => ({
  '--grid-cols': String(GRID_COLUMNS),
  '--grid-row': `${GRID_ROW_HEIGHT}px`,
  '--grid-gap': `${GRID_GAP}px`,
  minHeight: `${gridRows.value * (GRID_ROW_HEIGHT + GRID_GAP)}px`,
}));

const selectedItem = computed(
  () => dashboard.layout.find((item) => item.id === selectedId.value) ?? null,
);
const selectedDef = computed(() => (selectedItem.value ? cardRegistry[selectedItem.value.id] : null));
const selectedPos = computed(() => {
  const item = selectedItem.value;
  if (!item) return null;
  const active = drag.value;
  return active && active.id === item.id ? active.preview : item;
});

const backgrounds = [
  { value: 'default', label: '米白' },
  { value: 'warm', label: '暖阳' },
  { value: 'sage', label: '青绿' },
  { value: 'sky', label: '天空' },
  { value: 'sand', label: '沙色' },
  { value: 'gradient', label: '渐变' },
];

function cardOf(id: string) {
  return dashboard.cardMap[id];
}

/** 卡片数据仍在独立加载中时展示骨架屏。 */
function cardLoading(id: string): boolean {
  return cardOf(id)?.loading === true;
}

/** 天气卡片有独立加载的数据时，忽略首页接口返回的卡片级错误。 */
function cardError(id: string): string | null {
  if (id === 'weather' && weather.value) return null;
  return cardOf(id)?.error ?? null;
}

function cardData(id: string): unknown {
  if (id === 'weather') return weather.value ?? cardOf(id)?.data ?? null;
  return cardOf(id)?.data ?? null;
}

function itemStyle(item: DashboardLayoutItem): Record<string, string> {
  const pos = drag.value?.id === item.id ? drag.value.preview : item;
  return {
    gridColumn: `${pos.x + 1} / span ${pos.w}`,
    gridRow: `${pos.y + 1} / span ${pos.h}`,
  };
}

function enterEdit(): void {
  editing.value = true;
  selectedId.value = displayItems.value[0]?.id ?? '';
}

function exitEdit(): void {
  editing.value = false;
  selectedId.value = '';
  drag.value = null;
  removeDragListeners();
}

function selectItem(id: string): void {
  if (!editing.value) return;
  selectedId.value = id;
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

function isStackedViewport(): boolean {
  return typeof window !== 'undefined' && window.innerWidth <= STACK_BREAKPOINT;
}

function startDrag(event: PointerEvent, item: DashboardLayoutItem, mode: DragMode): void {
  if (!editing.value || isStackedViewport()) return;
  if (event.pointerType === 'mouse' && event.button !== 0) return;
  event.preventDefault();
  selectedId.value = item.id;
  drag.value = {
    id: item.id,
    pointerId: event.pointerId,
    mode,
    startX: event.clientX,
    startY: event.clientY,
    origin: { ...item },
    preview: { ...item },
  };
  window.addEventListener('pointermove', onPointerMove);
  window.addEventListener('pointerup', onPointerUp);
  window.addEventListener('pointercancel', onPointerUp);
}

function onPointerMove(event: PointerEvent): void {
  const state = drag.value;
  const grid = gridRef.value;
  if (!state || !grid || event.pointerId !== state.pointerId) return;
  const columnWidth = Math.max(
    (grid.clientWidth - GRID_GAP * (GRID_COLUMNS - 1)) / GRID_COLUMNS,
    1,
  );
  const stepX = columnWidth + GRID_GAP;
  const stepY = GRID_ROW_HEIGHT + GRID_GAP;
  const deltaCol = Math.round((event.clientX - state.startX) / stepX);
  const deltaRow = Math.round((event.clientY - state.startY) / stepY);
  const origin = state.origin;
  let preview: DashboardLayoutItem;
  if (state.mode === 'move') {
    preview = {
      ...origin,
      x: clamp(origin.x + deltaCol, 0, GRID_COLUMNS - origin.w),
      y: clamp(origin.y + deltaRow, 0, GRID_MAX_ROWS - origin.h),
    };
  } else {
    preview = {
      ...origin,
      w: clamp(origin.w + deltaCol, CARD_MIN_W, GRID_COLUMNS - origin.x),
      h: clamp(origin.h + deltaRow, CARD_MIN_H, CARD_MAX_H),
    };
  }
  drag.value = { ...state, preview };
}

function onPointerUp(event: PointerEvent): void {
  const state = drag.value;
  if (!state || event.pointerId !== state.pointerId) return;
  removeDragListeners();
  drag.value = null;
  const preview = state.preview;
  if (
    preview.x === state.origin.x &&
    preview.y === state.origin.y &&
    preview.w === state.origin.w &&
    preview.h === state.origin.h
  ) {
    return;
  }
  const next = dashboard.layout.map((item) =>
    item.id === state.id
      ? { ...item, x: preview.x, y: preview.y, w: preview.w, h: preview.h }
      : { ...item },
  );
  void persist(next);
}

function removeDragListeners(): void {
  window.removeEventListener('pointermove', onPointerMove);
  window.removeEventListener('pointerup', onPointerUp);
  window.removeEventListener('pointercancel', onPointerUp);
}

function overlaps(a: DashboardLayoutItem, b: DashboardLayoutItem): boolean {
  return a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y;
}

function byPosition(a: DashboardLayoutItem, b: DashboardLayoutItem): number {
  return a.y - b.y || a.x - b.x;
}

/**
 * 可见卡片按原顺序向上紧凑排布，消除大的垂直空白；
 * 隐藏卡片统一紧贴在可见区域之后，避免占住首屏位置。
 */
function reflowLayout(items: DashboardLayoutItem[]): DashboardLayoutItem[] {
  const result = items.map((item) => ({ ...item }));
  const visible = result.filter((item) => item.visible).sort(byPosition);
  const hidden = result.filter((item) => !item.visible).sort(byPosition);

  const placedVisible: DashboardLayoutItem[] = [];
  for (const item of visible) {
    item.y = 0;
    while (placedVisible.some((entry) => overlaps(item, entry))) item.y += 1;
    placedVisible.push(item);
  }

  const visibleBottom = placedVisible.reduce((max, item) => Math.max(max, item.y + item.h), 0);
  const placedHidden: DashboardLayoutItem[] = [];
  for (const item of hidden) {
    item.y = visibleBottom;
    while (
      placedVisible.some((entry) => overlaps(item, entry)) ||
      placedHidden.some((entry) => overlaps(item, entry))
    ) {
      item.y += 1;
    }
    placedHidden.push(item);
  }
  return result;
}

/** 串行保存，避免连续拖拽时请求乱序覆盖。 */
function persist(next: DashboardLayoutItem[]): Promise<void> {
  const snapshot = reflowLayout(next);
  dashboard.layout = snapshot.map((item) => ({ ...item }));
  saveQueue = saveQueue.then(async () => {
    try {
      await dashboard.saveLayout(snapshot);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '保存首页布局失败');
      try {
        await dashboard.load(true);
      } catch {
        /* 忽略刷新失败，等待下次加载 */
      }
    }
  });
  return saveQueue;
}

function toggleCard(id: string): void {
  if (id === 'overview') return;
  const next = dashboard.layout.map((item) =>
    item.id === id ? { ...item, visible: !item.visible } : { ...item },
  );
  void persist(next);
}

function setBackground(id: string, background: string): void {
  const next = dashboard.layout.map((item) =>
    item.id === id ? { ...item, background } : { ...item },
  );
  void persist(next);
}

/** 天气单独加载并设超时，失败时保留上次数据，不影响首页其他卡片。 */
let weatherAbort: AbortController | null = null;

async function loadWeather(): Promise<void> {
  weatherAbort?.abort();
  const controller = new AbortController();
  weatherAbort = controller;
  const timeout = window.setTimeout(() => controller.abort(), 10000);
  try {
    weather.value = await api.get<WeatherData>('/weather', undefined, controller.signal);
  } catch {
    // 天气加载失败或超时：保留已有数据，静默忽略
  } finally {
    window.clearTimeout(timeout);
    if (weatherAbort === controller) weatherAbort = null;
  }
}

async function load(silent = false): Promise<void> {
  void loadWeather();
  try {
    await dashboard.load(silent);
  } catch {
    if (!silent) toast.error('首页加载失败，请稍后重试');
  }
}

async function refresh(): Promise<void> {
  await load(true);
}

interface TauriInvoke {
  invoke<T = unknown>(cmd: string, args?: Record<string, unknown>): Promise<T>;
}

interface TauriWindow {
  isFullscreen(): Promise<boolean>;
  setFullscreen(value: boolean): Promise<void>;
}

interface TauriGlobal {
  core?: TauriInvoke;
  window?: { getCurrentWindow(): TauriWindow };
}

/** 在 Tauri 原生壳内时返回窗口对象；普通浏览器返回 undefined。 */
function tauriWindow(): TauriWindow | undefined {
  return (globalThis as { __TAURI__?: TauriGlobal }).__TAURI__?.window?.getCurrentWindow();
}

function fullscreenTarget(): HTMLElement | null {
  return homeRef.value?.closest('main') ?? null;
}

async function toggleFullscreen(): Promise<void> {
  const window = tauriWindow();
  if (window) {
    try {
      const next = !(await window.isFullscreen());
      await window.setFullscreen(next);
      isFullscreen.value = next;
    } catch {
      toast.error('切换全屏失败');
    }
    return;
  }
  if (!document.fullscreenEnabled) {
    toast.error('当前环境不支持全屏');
    return;
  }
  try {
    if (document.fullscreenElement) {
      await document.exitFullscreen();
      return;
    }
    const target = fullscreenTarget();
    if (!target) return;
    await target.requestFullscreen();
  } catch {
    toast.error('切换全屏失败');
  }
}

function onFullscreenChange(): void {
  isFullscreen.value =
    document.fullscreenElement !== null && document.fullscreenElement === fullscreenTarget();
}

let resizeTimer: number | undefined;

/** 窗口尺寸变化（含通过系统方式退出原生全屏）时同步全屏状态。 */
function onWindowResize(): void {
  window.clearTimeout(resizeTimer);
  resizeTimer = window.setTimeout(async () => {
    const current = tauriWindow();
    if (!current) return;
    try {
      isFullscreen.value = await current.isFullscreen();
    } catch {
      // 读取失败时保持现状
    }
  }, 200);
}

async function resetLayout(): Promise<void> {
  resetOpen.value = false;
  try {
    await dashboard.resetLayout();
    selectedId.value = dashboard.layout[0]?.id ?? '';
    toast.success('已恢复家庭默认布局');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '恢复失败');
  }
}

async function publishLayout(): Promise<void> {
  try {
    await dashboard.publishHouseholdLayout(dashboard.layout);
    toast.success('已发布为家庭默认布局');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '发布失败');
  }
}

function onSaved(): void {
  toast.success('待办已创建');
  void load(true);
}

watch(
  () => displayItems.value.length,
  () => {
    if (editing.value && !selectedId.value) selectedId.value = displayItems.value[0]?.id ?? '';
  },
);

onMounted(async () => {
  if (route.query.customize === '1') {
    editing.value = true;
    void router.replace({ query: {} });
  }
  await load();
  if (editing.value) selectedId.value = displayItems.value[0]?.id ?? '';
  document.addEventListener('fullscreenchange', onFullscreenChange);
  window.addEventListener('resize', onWindowResize);
  timer = window.setInterval(() => {
    if (document.hidden || editing.value) return;
    void load(todayISO() !== dashboard.today ? false : true);
  }, 5 * 60 * 1000);
});

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer);
  weatherAbort?.abort();
  document.removeEventListener('fullscreenchange', onFullscreenChange);
  window.removeEventListener('resize', onWindowResize);
  window.clearTimeout(resizeTimer);
  if (isFullscreen.value) void document.exitFullscreen().catch(() => undefined);
  removeDragListeners();
});
</script>

<template>
  <div ref="homeRef" class="home" :class="{ 'home--editing': editing, 'home--fullscreen': isFullscreen }">
    <header class="hero">
      <div class="hero__text">
        <p class="at-caption-upper hero__date">{{ dateText }}</p>
        <div class="hero__headline">
          <h1 class="at-display hero__title">{{ editing ? '整理首页' : greetingText }}</h1>
          <WeatherMiniCard v-if="weather && !editing" :data="weather" @configure="weatherOpen = true" />
        </div>
        <p class="hero__sub">{{ summary }}</p>
      </div>
      <div class="hero__actions">
        <template v-if="editing">
          <AppButton variant="ghost" :loading="dashboard.saving" @click="resetOpen = true">
            恢复家庭默认
          </AppButton>
          <AppButton
            v-if="auth.isAdmin"
            variant="outline"
            :loading="dashboard.saving"
            @click="publishLayout"
          >
            发布为家庭默认
          </AppButton>
          <AppButton @click="exitEdit">完成</AppButton>
        </template>
        <template v-else>
          <AppButton @click="formOpen = true"><AppIcon name="plus" :size="16" />新建待办</AppButton>
          <AppButton variant="outline" @click="enterEdit">
            <AppIcon name="grid" :size="16" />整理首页
          </AppButton>
          <AppIconButton
            icon="refresh"
            label="刷新首页"
            :active="dashboard.refreshing"
            @click="refresh"
          />
          <AppIconButton
            :icon="isFullscreen ? 'fullscreen-exit' : 'fullscreen'"
            :label="isFullscreen ? '退出全屏' : '全屏显示首页'"
            :active="isFullscreen"
            @click="toggleFullscreen"
          />
        </template>
      </div>
    </header>

    <div v-if="editing" class="home__inspector">
      <div class="home__inspector-main">
        <div class="home__inspector-text">
          <span class="home__inspector-title">{{ selectedDef?.title ?? '选择一张卡片' }}</span>
          <span v-if="selectedPos" class="home__inspector-meta">
            {{ selectedPos.w }} 列 × {{ selectedPos.h }} 行 · 最小 {{ CARD_MIN_W }} × {{ CARD_MIN_H }}
          </span>
        </div>
        <AppButton
          v-if="selectedItem && selectedItem.id !== 'overview'"
          variant="ghost"
          size="sm"
          @click="toggleCard(selectedItem.id)"
        >
          {{ selectedItem.visible ? '隐藏卡片' : '显示卡片' }}
        </AppButton>
      </div>
      <div v-if="selectedItem" class="home__swatches" role="group" aria-label="卡片底色">
        <button
          v-for="bg in backgrounds"
          :key="bg.value"
          type="button"
          class="home__swatch"
          :class="[`home__swatch--${bg.value}`, { 'is-active': selectedItem.background === bg.value }]"
          :title="bg.label"
          :aria-label="bg.label"
          :aria-pressed="selectedItem.background === bg.value"
          @click="setBackground(selectedItem.id, bg.value)"
        />
      </div>
    </div>

    <div v-if="showSkeleton" class="home__grid" :style="gridStyle" aria-busy="true" aria-label="正在加载首页">
      <article
        v-for="item in placeholderItems"
        :key="item.id"
        class="dash dash--bg-default"
        :style="itemStyle(item)"
      >
        <AppPanel padding="sm">
          <template #header>
            <div class="dash__heading">
              <span class="dash__icon">
                <AppIcon :name="cardRegistry[item.id].icon" :size="16" />
              </span>
              <div class="dash__heading-text">
                <h2 class="dash__title">{{ cardRegistry[item.id].title }}</h2>
              </div>
            </div>
          </template>
          <div class="home__skeleton-lines">
            <span class="home__skeleton-line home__skeleton-line--short" />
            <span class="home__skeleton-line" />
            <span class="home__skeleton-line" />
          </div>
        </AppPanel>
      </article>
    </div>

    <div v-else-if="dashboard.error && !dashboard.cards.length" class="home__error">
      <p>首页暂时没能加载出来。</p>
      <AppButton variant="outline" size="sm" @click="load()">重试</AppButton>
    </div>

    <div v-else-if="!editing && !displayItems.length" class="home__empty">
      <AppEmptyState title="首页还没有卡片" description="挑选想看的卡片，放到每天打开就能看到的位置" icon="grid">
        <AppButton size="sm" variant="outline" @click="enterEdit">添加首页卡片</AppButton>
      </AppEmptyState>
    </div>

    <div v-else class="home__canvas" :class="{ 'home__canvas--editing': editing }">
      <div
        ref="gridRef"
        class="home__grid"
        :class="{ 'home__grid--editing': editing }"
        :style="gridStyle"
      >
        <article
          v-for="item in displayItems"
          :key="item.id"
          class="dash"
          :class="[
            `dash--bg-${item.background}`,
            {
              'is-editing': editing,
              'is-selected': editing && selectedId === item.id,
              'is-hidden': editing && !item.visible,
            },
          ]"
          :style="itemStyle(item)"
          @click="selectItem(item.id)"
        >
          <AppPanel padding="sm" :hoverable="!editing">
            <template #header>
              <div
                class="dash__heading"
                :class="{ 'dash__heading--draggable': editing }"
                :title="editing ? '拖动移动卡片' : undefined"
                @pointerdown="startDrag($event, item, 'move')"
              >
                <span class="dash__icon"><AppIcon :name="cardRegistry[item.id].icon" :size="16" /></span>
                <div class="dash__heading-text">
                  <h2 class="dash__title">{{ cardOf(item.id)?.title ?? cardRegistry[item.id].title }}</h2>
                </div>
              </div>
            </template>
            <template #actions>
              <div class="dash__actions">
                <AppIconButton
                  v-if="editing && item.id !== 'overview'"
                  :icon="item.visible ? 'eye' : 'eye-off'"
                  :label="item.visible ? '隐藏卡片' : '显示卡片'"
                  size="sm"
                  @click.stop="toggleCard(item.id)"
                />
                <AppIconButton
                  v-else-if="!editing && item.id === 'weather'"
                  icon="settings"
                  label="设置天气位置"
                  size="sm"
                  @click="weatherOpen = true"
                />
              </div>
            </template>

            <div v-if="editing && !item.visible" class="dash__placeholder">
              <AppIcon name="eye-off" :size="18" />
              <span>这张卡片已隐藏</span>
            </div>
            <div v-else class="dash__body">
              <div v-if="cardLoading(item.id)" class="home__skeleton-lines" aria-busy="true">
                <span class="home__skeleton-line home__skeleton-line--short" />
                <span class="home__skeleton-line" />
                <span class="home__skeleton-line" />
              </div>
              <div v-else-if="cardError(item.id)" class="dash__card-error">
                <AppIcon name="alert" :size="15" />
                <span>{{ cardError(item.id) }}</span>
                <button type="button" class="dash__card-retry" @click="refresh">重试</button>
              </div>
              <component
                :is="cardRegistry[item.id].component"
                v-else
                :data="cardData(item.id)"
                @refresh="refresh"
                @configure="weatherOpen = true"
              />
            </div>

            <template v-if="!editing && cardRegistry[item.id].path" #footer>
              <RouterLink :to="cardRegistry[item.id].path!" class="dash__more">
                查看全部<AppIcon name="arrow-right" :size="14" />
              </RouterLink>
            </template>
          </AppPanel>
          <span
            v-if="editing"
            class="dash__resize"
            aria-hidden="true"
            @pointerdown.stop.prevent="startDrag($event, item, 'resize')"
          />
        </article>
      </div>
    </div>

    <TaskFormDialog v-model="formOpen" @saved="onSaved" />
    <WeatherSettingsDialog v-model="weatherOpen" @saved="load(true)" />
    <AppConfirmDialog
      v-model="resetOpen"
      title="恢复家庭默认布局"
      message="会使用家庭管理员发布的默认卡片位置、大小与显示设置，你当前的自定义布局将被覆盖。"
      confirm-text="恢复默认"
      @confirm="resetLayout"
    />
  </div>
</template>

<style scoped>
.home {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-6);
  width: 100%;
  max-width: var(--at-content-max);
  margin: 0 auto;
  padding: var(--at-space-8) var(--at-space-6) var(--at-space-12);
}

/* 原生壳内窗口全屏后，让首页内容铺满整个屏幕，隐藏应用外壳与页面空白 */
.home--fullscreen {
  position: fixed;
  inset: 0;
  z-index: 100;
  max-width: none;
  overflow: auto;
  background: var(--at-bg-page);
  /* 内容不满一屏时垂直居中；safe 保证内容超高时从顶部开始、不被裁切 */
  justify-content: safe center;
}

.hero {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--at-space-6);
  padding: var(--at-space-4) 0 var(--at-space-2);
}

.hero__date {
  color: var(--at-color-muted);
}

.hero__title {
  margin-top: 0;
}

.hero__headline {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  flex-wrap: wrap;
  margin-top: var(--at-space-2);
}

.hero__sub {
  margin-top: var(--at-space-2);
  max-width: 52ch;
  font-size: var(--at-text-body-sm);
  color: var(--at-color-muted);
  line-height: 1.65;
}

.hero__actions {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  flex: none;
}

/* 编辑态工具条 */
.home__inspector {
  position: sticky;
  top: calc(var(--at-topbar-height) + var(--at-space-2));
  z-index: var(--at-z-sticky);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-4);
  flex-wrap: wrap;
  padding: var(--at-space-3) var(--at-space-4);
  background: color-mix(in srgb, var(--at-bg-surface) 92%, transparent);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-lg);
  box-shadow: var(--at-shadow-panel);
  backdrop-filter: blur(12px);
}

.home__inspector-main {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  min-width: 0;
}

.home__inspector-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.home__inspector-title {
  font-size: var(--at-text-body-sm);
  font-weight: 500;
  color: var(--at-color-ink);
}

.home__inspector-meta {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.home__swatches {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.home__swatch {
  position: relative;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1px solid var(--at-color-hairline-strong);
  transition: transform var(--at-duration-fast) var(--at-ease), box-shadow var(--at-duration-fast) var(--at-ease);
}

.home__swatch::after {
  content: '';
  position: absolute;
  inset: -10px;
  border-radius: 50%;
}

.home__swatch:hover {
  transform: scale(1.08);
}

.home__swatch.is-active {
  box-shadow: 0 0 0 2px var(--at-bg-surface), 0 0 0 3px var(--at-color-ink);
}

.home__swatch--default { background: var(--at-bg-surface); }
.home__swatch--warm { background: var(--at-card-peach); }
.home__swatch--sage { background: var(--at-card-mint); }
.home__swatch--sky { background: var(--at-card-sky); }
.home__swatch--sand { background: var(--at-card-sand); }
.home__swatch--gradient {
  background: linear-gradient(135deg, var(--at-card-lavender) 0%, var(--at-card-peach) 52%, var(--at-card-mint) 100%);
}

/* 栅格画布 */
.home__canvas {
  min-width: 0;
}

.home__canvas--editing {
  padding: var(--at-space-3);
  background: var(--at-bg-canvas-soft);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-xxl);
}

.home__grid {
  display: grid;
  grid-template-columns: repeat(var(--grid-cols, 12), minmax(0, 1fr));
  grid-auto-rows: var(--grid-row, 40px);
  gap: var(--grid-gap, 12px);
  align-items: stretch;
  min-width: 0;
}

.home__grid--editing {
  background-image:
    linear-gradient(to right, var(--at-color-hairline-soft) 1px, transparent 1px),
    linear-gradient(to bottom, var(--at-color-hairline-soft) 1px, transparent 1px);
  background-size: calc(100% / 12) calc(var(--grid-row, 40px) + var(--grid-gap, 12px));
  border-radius: var(--at-radius-lg);
}

.dash {
  position: relative;
  display: flex;
  min-width: 0;
  min-height: 0;
}

.dash :deep(.at-panel) {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  height: 100%;
}

.dash :deep(.at-panel__header),
.dash :deep(.at-panel__footer) {
  flex: none;
}

.dash :deep(.at-panel__body) {
  flex: 1;
  min-height: 0;
  overflow: auto;
  overscroll-behavior: contain;
  /* 预留滚动条槽位但默认透明，悬停时再显示滑块，避免出现时内容跳动 */
  scrollbar-width: thin;
  scrollbar-color: transparent transparent;
}

.dash :deep(.at-panel__body)::-webkit-scrollbar {
  width: 6px;
  height: 6px;
  background: transparent;
}

.dash :deep(.at-panel__body)::-webkit-scrollbar-thumb {
  background: transparent;
  border: 0;
  border-radius: var(--at-radius-pill);
}

.dash :deep(.at-panel__body)::-webkit-scrollbar-track {
  background: transparent;
}

.dash:hover :deep(.at-panel__body),
.dash:focus-within :deep(.at-panel__body) {
  scrollbar-color: var(--at-color-hairline-strong) transparent;
}

.dash:hover :deep(.at-panel__body)::-webkit-scrollbar-thumb,
.dash:focus-within :deep(.at-panel__body)::-webkit-scrollbar-thumb {
  background: var(--at-color-hairline-strong);
}

.dash.is-editing :deep(.at-panel__body) {
  overflow: hidden;
}

.dash.is-editing .dash__body {
  height: 100%;
  pointer-events: none;
}

.dash.is-selected :deep(.at-panel) {
  box-shadow: 0 0 0 2px var(--at-color-ink), var(--at-shadow-card-hover);
}

.dash.is-hidden :deep(.at-panel) {
  opacity: 0.6;
  border-style: dashed;
  box-shadow: var(--at-shadow-panel);
}

.dash--bg-warm :deep(.at-panel) { background: var(--at-card-peach); }
.dash--bg-sage :deep(.at-panel) { background: var(--at-card-mint); }
.dash--bg-sky :deep(.at-panel) { background: var(--at-card-sky); }
.dash--bg-sand :deep(.at-panel) { background: var(--at-card-sand); }
.dash--bg-gradient :deep(.at-panel) {
  background: linear-gradient(135deg, var(--at-card-lavender) 0%, var(--at-card-peach) 52%, var(--at-card-mint) 100%);
}

.dash__heading {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  min-width: 0;
}

.dash__heading--draggable {
  cursor: grab;
  user-select: none;
  touch-action: none;
}

.dash__heading--draggable:active {
  cursor: grabbing;
}

.dash__icon {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 50%;
  background: color-mix(in srgb, var(--at-bg-surface-strong) 82%, transparent);
  color: var(--at-color-body);
}

.dash__heading-text {
  min-width: 0;
}

.dash__title {
  font-family: var(--at-font-display);
  font-size: var(--at-text-title-sm);
  font-weight: 400;
  line-height: 1.2;
  letter-spacing: 0.2px;
}

.dash__actions {
  display: flex;
  align-items: center;
  gap: 0;
  opacity: 0;
  transition: opacity var(--at-duration-fast) var(--at-ease);
}

.dash:hover .dash__actions,
.dash__actions:focus-within,
.dash.is-editing .dash__actions {
  opacity: 1;
}

.dash__body {
  min-width: 0;
}

/* 照片卡按卡片可用高度撑满，图片再按高度等比缩放 */
.dash__body:has(> .photo) {
  height: 100%;
}

.dash__placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--at-space-2);
  height: 100%;
  min-height: 96px;
  padding: var(--at-space-4);
  color: var(--at-color-muted);
  font-size: var(--at-text-caption);
  background: var(--at-bg-canvas-soft);
  border: 1px dashed var(--at-color-hairline-strong);
  border-radius: var(--at-radius-lg);
}

.dash__resize {
  position: absolute;
  right: 2px;
  bottom: 2px;
  z-index: 2;
  width: 20px;
  height: 20px;
  cursor: nwse-resize;
  touch-action: none;
  opacity: 0.5;
  background: linear-gradient(135deg, transparent 50%, var(--at-color-ink) 50%);
  border-bottom-right-radius: var(--at-radius-card, var(--at-radius-xl));
  transition: opacity var(--at-duration-fast) var(--at-ease);
}

.dash__resize:hover {
  opacity: 1;
}

.dash__card-error {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  min-height: 72px;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.dash__card-retry {
  margin-left: auto;
  padding: 0;
  border: 0;
  background: none;
  font-size: var(--at-text-caption);
  color: var(--at-color-ink);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.dash__more {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  transition: color var(--at-duration-fast) var(--at-ease);
}

.dash__more:hover {
  color: var(--at-color-ink);
}

.home__skeleton-lines {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  padding-top: var(--at-space-2);
}

.home__skeleton-line {
  display: block;
  height: 12px;
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface-strong);
  animation: home-pulse 1.6s var(--at-ease) infinite;
}

.home__skeleton-line--short {
  width: 40%;
}

.home__error {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--at-space-3);
  padding: var(--at-space-6);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-xl);
  color: var(--at-color-muted);
}

.home__empty {
  border: 1px dashed var(--at-color-hairline-strong);
  border-radius: var(--at-radius-xl);
  background: var(--at-bg-surface);
}

@keyframes home-pulse {
  0%, 100% { opacity: 0.55; }
  50% { opacity: 1; }
}

@media (max-width: 1100px) {
  .home {
    padding: var(--at-space-6) var(--at-space-4) var(--at-space-12);
  }
}

@media (max-width: 900px) {
  .home {
    padding: var(--at-space-5) var(--at-space-4) 96px;
  }

  .hero {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--at-space-4);
  }

  .hero__actions {
    flex-wrap: wrap;
  }

  .home__inspector {
    position: static;
    flex-direction: column;
    align-items: stretch;
  }

  .home__inspector-main {
    justify-content: space-between;
  }

  .home__canvas--editing {
    padding: var(--at-space-2);
  }

  .home__grid,
  .home__grid--editing {
    grid-template-columns: minmax(0, 1fr);
    grid-auto-rows: auto;
    min-height: 0 !important;
    background-image: none;
  }

  .dash {
    grid-column: 1 / -1 !important;
    grid-row: auto !important;
  }

  .dash :deep(.at-panel) {
    height: auto;
  }

  .dash :deep(.at-panel__body) {
    overflow: visible;
  }

  .dash__heading--draggable {
    cursor: default;
  }

  .dash__resize {
    display: none;
  }

  .dash__actions {
    opacity: 1;
  }
}

@media (hover: none) {
  .dash__actions {
    opacity: 1;
  }
}
</style>
