<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { RouterLink } from 'vue-router';
import { greeting, todayISO } from '@/utils/chrono';
import { useAuthStore } from '@/stores/auth';
import { useDashboardStore } from '@/stores/dashboard';
import { useToastStore } from '@/stores/toast';
import { ApiError } from '@/api/client';
import { cardRegistry } from '@/cards/registry';
import type { OverviewData } from '@/cards/types';
import type { DashboardLayoutItem } from '@/types';
import AppButton from '@/components/base/AppButton.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';
import AppPanel from '@/components/base/AppPanel.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import TaskFormDialog from '@/components/business/TaskFormDialog.vue';
import DashboardCustomizer from '@/components/business/DashboardCustomizer.vue';
import WeatherSettingsDialog from '@/components/business/WeatherSettingsDialog.vue';

const auth = useAuthStore();
const dashboard = useDashboardStore();
const toast = useToastStore();

const formOpen = ref(false);
const customizerOpen = ref(false);
const weatherOpen = ref(false);
const resetOpen = ref(false);
const collapsed = ref<Set<string>>(new Set());
let timer: number | undefined;

const collapseKey = computed(() => `mymoments:dashboard:collapsed:${auth.user?.id ?? 'anon'}`);

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
  if (dashboard.loading && !dashboard.cards.length) return '正在加载首页…';
  const counts = overview.value?.counts;
  if (!counts) return '看看今天家里有什么事。';
  if (counts.overdue > 0) return `有 ${counts.overdue} 件已经逾期。`;
  if (counts.today > 0) return `今天有 ${counts.today} 件待办。`;
  if (counts.open > 0) return `还有 ${counts.open} 件待办未完成。`;
  return '今天没有待办。';
});

const visibleItems = computed(() => dashboard.layout.filter((item) => item.visible && cardRegistry[item.id]));

function cardOf(id: string) {
  return dashboard.cardMap[id];
}

function isCollapsed(id: string): boolean {
  return collapsed.value.has(id);
}

function toggleCollapse(id: string): void {
  const next = new Set(collapsed.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  collapsed.value = next;
}

function canMove(id: string, dir: -1 | 1): boolean {
  const index = dashboard.layout.findIndex((item) => item.id === id);
  const target = index + dir;
  return index > 0 && target > 0 && target < dashboard.layout.length;
}

async function moveCard(id: string, dir: -1 | 1): Promise<void> {
  const index = dashboard.layout.findIndex((item) => item.id === id);
  const target = index + dir;
  if (index <= 0 || target <= 0 || target >= dashboard.layout.length) return;
  const next = dashboard.layout.map((item) => ({ ...item }));
  [next[index], next[target]] = [next[target], next[index]];
  await persist(next);
}

function toggleCard(id: string): void {
  if (id === 'overview') return;
  const next = dashboard.layout.map((item) => (item.id === id ? { ...item, visible: !item.visible } : { ...item }));
  void persist(next);
}

function resizeCard(id: string, size: DashboardLayoutItem['size']): void {
  const next = dashboard.layout.map((item) => (item.id === id ? { ...item, size } : { ...item }));
  void persist(next);
}

function backgroundCard(id: string, background: string): void {
  const next = dashboard.layout.map((item) => (item.id === id ? { ...item, background } : { ...item }));
  void persist(next);
}

async function persist(items: DashboardLayoutItem[]): Promise<void> {
  try {
    await dashboard.saveLayout(items);
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '保存首页布局失败');
  }
}

async function load(silent = false): Promise<void> {
  try {
    await dashboard.load(silent);
  } catch {
    if (!silent) toast.error('首页加载失败，请稍后重试');
  }
}

async function refresh(): Promise<void> {
  await load(true);
}

async function resetLayout(): Promise<void> {
  resetOpen.value = false;
  try {
    await dashboard.resetLayout();
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

function loadCollapsed(): void {
  try {
    const raw = localStorage.getItem(collapseKey.value);
    const ids = raw ? (JSON.parse(raw) as string[]) : [];
    collapsed.value = new Set(Array.isArray(ids) ? ids : []);
  } catch {
    collapsed.value = new Set();
  }
}

watch(collapsed, (value) => {
  try {
    localStorage.setItem(collapseKey.value, JSON.stringify([...value]));
  } catch {
    /* 忽略存储失败 */
  }
});

watch(collapseKey, () => loadCollapsed());

onMounted(async () => {
  loadCollapsed();
  await load();
  timer = window.setInterval(() => {
    if (document.hidden) return;
    void load(todayISO() !== dashboard.today ? false : true);
  }, 5 * 60 * 1000);
});

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer);
});
</script>

<template>
  <div class="home">
    <header class="hero">
      <div class="hero__text">
        <p class="at-caption-upper hero__date">{{ dateText }}</p>
        <h1 class="at-display hero__title">{{ greetingText }}</h1>
        <p class="hero__sub">{{ summary }}</p>
      </div>
      <div class="hero__actions">
        <AppButton @click="formOpen = true"><AppIcon name="plus" :size="16" />新建待办</AppButton>
        <AppButton variant="outline" @click="customizerOpen = true">
          <AppIcon name="grid" :size="16" />整理首页
        </AppButton>
        <AppIconButton
          icon="refresh"
          label="刷新首页"
          :active="dashboard.refreshing"
          @click="refresh"
        />
      </div>
    </header>

    <div v-if="dashboard.loading && !dashboard.cards.length" class="home__grid" aria-busy="true" aria-label="正在加载首页">
      <div v-for="n in 3" :key="n" class="home__skeleton at-card" :class="{ 'is-wide': n === 1 }">
        <span class="home__skeleton-line home__skeleton-line--short" />
        <span class="home__skeleton-line" />
        <span class="home__skeleton-line" />
      </div>
    </div>

    <div v-else-if="dashboard.error && !dashboard.cards.length" class="home__error">
      <p>首页暂时没能加载出来。</p>
      <AppButton variant="outline" size="sm" @click="load()">重试</AppButton>
    </div>

    <div v-else-if="!visibleItems.length" class="home__empty">
      <AppEmptyState title="首页还没有卡片" description="挑选想看的卡片，放到每天打开就能看到的位置" icon="grid">
        <AppButton size="sm" variant="outline" @click="customizerOpen = true">添加首页卡片</AppButton>
      </AppEmptyState>
    </div>

    <div v-else class="home__grid">
      <article
        v-for="item in visibleItems"
        :key="item.id"
        class="dash"
        :class="[`dash--${item.size}`, `dash--bg-${item.background}`, { 'is-collapsed': isCollapsed(item.id) }]"
      >
        <AppPanel padding="sm" hoverable>
          <template #header>
            <div class="dash__heading">
              <span class="dash__icon"><AppIcon :name="cardRegistry[item.id].icon" :size="16" /></span>
              <div class="dash__heading-text">
                <h2 class="dash__title">{{ cardOf(item.id)?.title ?? cardRegistry[item.id].title }}</h2>
              </div>
            </div>
          </template>
          <template #actions>
            <div class="dash__actions">
              <AppIconButton
                v-if="item.id === 'weather'"
                icon="settings"
                label="设置天气位置"
                size="sm"
                @click="weatherOpen = true"
              />
              <AppIconButton
                icon="chevron-up"
                label="上移"
                size="sm"
                :disabled="!canMove(item.id, -1)"
                @click="moveCard(item.id, -1)"
              />
              <AppIconButton
                icon="chevron-down"
                label="下移"
                size="sm"
                :disabled="!canMove(item.id, 1)"
                @click="moveCard(item.id, 1)"
              />
              <AppIconButton
                :icon="isCollapsed(item.id) ? 'plus' : 'minus'"
                :label="isCollapsed(item.id) ? '展开卡片' : '折叠卡片'"
                size="sm"
                @click="toggleCollapse(item.id)"
              />
            </div>
          </template>

          <div v-show="!isCollapsed(item.id)" class="dash__body">
            <div v-if="cardOf(item.id)?.error" class="dash__card-error">
              <AppIcon name="alert" :size="15" />
              <span>{{ cardOf(item.id)?.error }}</span>
              <button type="button" class="dash__card-retry" @click="refresh">重试</button>
            </div>
            <component
              :is="cardRegistry[item.id].component"
              v-else
              :data="cardOf(item.id)?.data ?? null"
              @refresh="refresh"
              @configure="weatherOpen = true"
            />
          </div>

          <template v-if="cardRegistry[item.id].path" #footer>
            <RouterLink :to="cardRegistry[item.id].path!" class="dash__more">
              查看全部<AppIcon name="arrow-right" :size="14" />
            </RouterLink>
          </template>
        </AppPanel>
      </article>
    </div>

    <TaskFormDialog v-model="formOpen" @saved="onSaved" />
    <DashboardCustomizer
      v-model="customizerOpen"
      :layout="dashboard.layout"
      :layout-source="dashboard.layoutSource"
      :saving="dashboard.saving"
      :is-admin="auth.isAdmin"
      @toggle="toggleCard"
      @move="moveCard"
      @resize="resizeCard"
      @background="backgroundCard"
      @reset="resetOpen = true"
      @publish="publishLayout"
    />
    <WeatherSettingsDialog v-model="weatherOpen" @saved="load(true)" />
    <AppConfirmDialog
      v-model="resetOpen"
      title="恢复家庭默认布局"
      message="会使用家庭管理员发布的默认卡片顺序与显示设置，你当前的自定义布局将被覆盖。"
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
  margin-top: var(--at-space-2);
}

.hero__sub {
  margin-top: var(--at-space-2);
  max-width: 42ch;
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

.home__grid {
  display: grid;
  grid-template-columns: repeat(12, minmax(0, 1fr));
  gap: var(--at-space-4);
  align-items: start;
}

.dash {
  grid-column: span 4;
  min-width: 0;
}

.dash--wide {
  grid-column: span 8;
}

.dash--tall {
  grid-column: span 4;
  grid-row: span 2;
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
.dash__actions:focus-within {
  opacity: 1;
}

.dash.is-collapsed .dash__actions {
  opacity: 1;
}

.dash__body {
  min-width: 0;
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

.home__skeleton {
  grid-column: span 4;
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  padding: var(--at-space-5);
  min-height: 200px;
  border-radius: var(--at-radius-xl);
}

.home__skeleton.is-wide {
  grid-column: span 8;
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

  .dash,
  .dash--tall {
    grid-column: span 6;
  }

  .dash--wide {
    grid-column: span 12;
  }
}

@media (max-width: 767px) {
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

  .dash,
  .dash--wide,
  .dash--tall {
    grid-column: span 12;
  }

  .dash--tall {
    grid-row: auto;
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
