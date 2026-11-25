<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import AppAvatar from '@/components/base/AppAvatar.vue';
import { avatarUrl } from '@/utils/avatar';
import AppButton from '@/components/base/AppButton.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppErrorState from '@/components/base/AppErrorState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import { useTimelineStore } from '@/stores/timeline';
import { useFamilyStore } from '@/stores/family';
import { useToastStore } from '@/stores/toast';
import type { IconName } from '@/components/base/icons';
import type { TimelineEntry } from '@/types';
import { formatDateCN, formatShortDate, formatTime, relativeDayLabel, todayISO } from '@/utils/chrono';

const store = useTimelineStore();
const family = useFamilyStore();
const toast = useToastStore();
const router = useRouter();
const route = useRoute();

const today = todayISO();
const anchor = ref(String(route.query.date ?? today));
const direction = ref<'past' | 'future'>('past');
const member = ref('0');
const search = ref('');
const allTypes = ['task', 'event', 'birthday', 'message', 'note'] as const;
const selectedTypes = ref<string[]>([...allTypes]);
const typeOptions: { key: string; label: string }[] = [
  { key: 'task', label: '待办' },
  { key: 'event', label: '日历' },
  { key: 'birthday', label: '生日' },
  { key: 'message', label: '留言' },
  { key: 'note', label: '便签' },
];
const memberOptions = computed(() => family.activeMembers.map((m) => ({ value: String(m.id), label: m.displayName })));
const groups = computed(() => {
  const map = new Map<string, TimelineEntry[]>();
  for (const entry of store.items) {
    const list = map.get(entry.occurredOn) ?? [];
    list.push(entry);
    map.set(entry.occurredOn, list);
  }
  return [...map.entries()].map(([date, items]) => ({ date, items }));
});

function toggleType(key: string): void {
  if (selectedTypes.value.includes(key)) {
    selectedTypes.value = selectedTypes.value.filter((t) => t !== key);
  } else {
    selectedTypes.value = [...selectedTypes.value, key];
  }
}

async function reload(): Promise<void> {
  try {
    await store.load({
      anchor: anchor.value,
      direction: direction.value,
      types: selectedTypes.value.length === allTypes.length ? undefined : selectedTypes.value,
      member: Number(member.value) || undefined,
      q: search.value.trim() || undefined,
    });
  } catch {
    toast.error('时光轴加载失败，请稍后重试');
  }
}

let timer: number | undefined;
watch([anchor, direction, member, selectedTypes, search], () => {
  window.clearTimeout(timer);
  timer = window.setTimeout(() => { void reload(); }, 250);
}, { deep: true });

onMounted(async () => {
  await family.load();
  await reload();
});

async function more(): Promise<void> {
  try {
    await store.loadMore();
  } catch {
    toast.error('加载更多失败，请稍后重试');
  }
}

function iconFor(type: TimelineEntry['sourceType']): IconName {
  switch (type) {
    case 'task_completion':
      return 'check';
    case 'task':
      return 'tasks';
    case 'event':
      return 'calendar';
    case 'birthday':
      return 'gift';
    case 'message':
      return 'messages';
    default:
      return 'notes';
  }
}

function statusLabel(status: string | null): string {
  switch (status) {
    case 'completed':
      return '已完成';
    case 'overdue':
      return '已逾期';
    case 'planned':
      return '计划中';
    case 'upcoming':
      return '即将到来';
    case 'record':
      return '旧时光';
    default:
      return '已过去';
  }
}

function openEntry(entry: TimelineEntry): void {
  const target = entry.detailTarget;
  if (entry.sourceType === 'birthday') {
    void router.push({ path: '/family', query: { tab: 'birthdays', open: String(target.open) } });
    return;
  }
  const pathMap: Record<string, string> = {
    tasks: '/tasks',
    calendar: '/calendar',
    messages: '/messages',
    notes: '/notes',
  };
  const path = pathMap[target.view];
  if (!path) return;
  const query: Record<string, string> = { open: String(target.open) };
  if (target.view === 'calendar') query.date = target.date;
  void router.push({ path, query });
}

function mediaURL(id: number): string {
  return `/api/v1/media/${id}/thumbnail`;
}
</script>

<template>
  <div class="at-page timeline">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">回望</p>
        <h1 class="at-page__title">时光轴</h1>
        <p class="at-page__subtitle">按时间顺序查看家里的待办、日程、便签和留言。</p>
      </div>
    </header>

    <section class="controls at-card">
      <div class="controls__row">
        <label class="date-jump">
          <AppIcon name="calendar" :size="16" />
          <input v-model="anchor" type="date" aria-label="锚点日期" />
        </label>
        <AppButton variant="ghost" size="sm" @click="anchor = today">回到今天</AppButton>
        <div class="segmented" role="tablist" aria-label="浏览方向">
          <button
            type="button"
            role="tab"
            :aria-selected="direction === 'past'"
            :class="{ 'is-active': direction === 'past' }"
            @click="direction = 'past'"
          >
            过去
          </button>
          <button
            type="button"
            role="tab"
            :aria-selected="direction === 'future'"
            :class="{ 'is-active': direction === 'future' }"
            @click="direction = 'future'"
          >
            未来
          </button>
        </div>
      </div>
      <div class="controls__row controls__row--wrap">
        <div class="type-chips" role="group" aria-label="内容类型">
          <button
            v-for="option in typeOptions"
            :key="option.key"
            type="button"
            class="chip"
            :class="{ 'is-active': selectedTypes.includes(option.key) }"
            :aria-pressed="selectedTypes.includes(option.key)"
            @click="toggleType(option.key)"
          >
            {{ option.label }}
          </button>
        </div>
        <label class="member-filter">
          <span class="at-caption-upper">成员</span>
          <select v-model="member" class="member-filter__select" aria-label="按成员筛选">
            <option value="0">全家</option>
            <option v-for="option in memberOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <label class="search">
          <AppIcon name="search" :size="16" />
          <input v-model="search" type="search" placeholder="搜索时光轴" aria-label="搜索时光轴" />
        </label>
      </div>
      <p v-if="store.partial.length" class="partial">
        部分内容暂时无法加载：{{ store.partial.join('、') }}
      </p>
    </section>

    <AppErrorState v-if="store.error" kind="server" compact @retry="reload" />
    <p v-else-if="store.loading && !store.items.length" class="at-faint loading">正在加载…</p>
    <AppEmptyState
      v-else-if="!store.items.length"
      icon="timeline"
      title="这段时间还没有记录"
      description="换个日期或方向看看，或者先去日历、待办里添加内容。"
    />

    <div v-else class="stream">
      <section v-for="group in groups" :key="group.date" class="day">
        <header class="day__head">
          <span class="day__dot" />
          <h2>{{ relativeDayLabel(group.date) }}</h2>
          <span class="day__date">{{ formatDateCN(group.date, { withYear: true, weekday: true }) }}</span>
        </header>
        <ul class="day__items">
          <li
            v-for="entry in group.items"
            :key="entry.occurrenceKey"
            class="entry at-card at-card--hover"
            tabindex="0"
            role="button"
            @click="openEntry(entry)"
            @keydown.enter="openEntry(entry)"
          >
            <span class="entry__icon" :class="`entry__icon--${entry.sourceType}`">
              <AppIcon :name="iconFor(entry.sourceType)" :size="16" />
            </span>
            <div class="entry__body">
              <div class="entry__title-row">
                <strong>{{ entry.title }}</strong>
                <span v-if="entry.status" class="badge" :class="`badge--${entry.status}`">{{ statusLabel(entry.status) }}</span>
              </div>
              <p v-if="entry.summary" class="entry__summary">{{ entry.summary }}</p>
              <div class="entry__meta">
                <span v-if="entry.sourceType === 'event' && entry.occurredAt">{{ formatTime(new Date(entry.occurredAt * 1000).toISOString()) }}</span>
                <span v-if="entry.sourceType === 'note' && entry.detailTarget.date">{{ formatShortDate(entry.detailTarget.date) }}</span>
                <span v-for="m in entry.members.slice(0, 4)" :key="m.id" class="entry__member">
                  <AppAvatar :name="m.displayName" :size="20" :src="avatarUrl(m.avatarKey)" />
                  <span>{{ m.displayName }}</span>
                </span>
              </div>
              <div v-if="entry.mediaPreview.length" class="entry__media">
                <img v-for="media in entry.mediaPreview" :key="media.id" :src="mediaURL(media.id)" :alt="media.originalName ?? '待办照片'" loading="lazy" />
              </div>
            </div>
            <AppIcon name="chevron-right" :size="16" class="entry__chevron" />
          </li>
        </ul>
      </section>

      <div class="more">
        <AppButton v-if="store.hasMore" :loading="store.loadingMore" @click="more">
          {{ direction === 'past' ? '加载更早的时光' : '加载更远的安排' }}
        </AppButton>
        <p v-else class="at-faint">
          {{ direction === 'past' ? '已经回到最早的记录' : '已经看到最新的安排' }}
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.controls {
  padding: var(--at-space-4);
  margin-bottom: var(--at-space-5);
  border-radius: var(--at-radius-xl);
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
}

.controls__row {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
}

.controls__row--wrap {
  flex-wrap: wrap;
}

.date-jump {
  display: inline-flex;
  align-items: center;
  gap: var(--at-space-2);
  height: var(--at-control-height);
  padding: 0 var(--at-space-3);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface);
  color: var(--at-color-muted);
}

.date-jump input {
  border: 0;
  background: transparent;
  outline: none;
  font-size: var(--at-text-caption);
  color: var(--at-text-primary);
}

.segmented {
  display: inline-flex;
  padding: 3px;
  margin-left: auto;
  background: var(--at-bg-surface-strong);
  border-radius: var(--at-radius-pill);
}

.segmented button {
  padding: 6px 18px;
  border-radius: var(--at-radius-pill);
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.segmented button.is-active {
  background: var(--at-bg-surface);
  color: var(--at-text-primary);
  box-shadow: var(--at-shadow-panel);
}

.type-chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.chip {
  padding: 6px 13px;
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface);
  color: var(--at-color-muted);
  font-size: var(--at-text-micro);
  transition: background var(--at-duration-fast) var(--at-ease), color var(--at-duration-fast) var(--at-ease);
}

.chip.is-active {
  background: var(--at-color-ink);
  border-color: var(--at-color-ink);
  color: var(--at-text-on-primary);
}

.member-filter {
  display: inline-flex;
  align-items: center;
  gap: var(--at-space-2);
}

.member-filter__select {
  height: var(--at-control-height);
  padding: 0 var(--at-space-3);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface);
  color: var(--at-text-primary);
  font-size: var(--at-text-caption);
}

.search {
  display: inline-flex;
  align-items: center;
  gap: var(--at-space-2);
  margin-left: auto;
  padding: 0 var(--at-space-3);
  height: var(--at-control-height);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface);
  color: var(--at-color-muted);
}

.search input {
  border: 0;
  outline: none;
  background: transparent;
  font-size: var(--at-text-caption);
  min-width: 150px;
  color: var(--at-text-primary);
}

.partial {
  font-size: var(--at-text-micro);
  color: var(--at-color-warning, #8a5a00);
}

.loading {
  padding: var(--at-space-6) 0;
  text-align: center;
  font-size: var(--at-text-caption);
}

.stream {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-6);
}

.day__head {
  display: flex;
  align-items: baseline;
  gap: var(--at-space-2);
  margin-bottom: var(--at-space-3);
}

.day__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--at-color-ink);
}

.day__head h2 {
  font-family: var(--at-font-display);
  font-size: var(--at-text-title-sm);
  font-weight: 400;
}

.day__date {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.day__items {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  padding-left: var(--at-space-4);
  border-left: 1px solid var(--at-color-hairline);
  margin-left: 3px;
}

.entry {
  display: flex;
  align-items: flex-start;
  gap: var(--at-space-3);
  padding: var(--at-space-3) var(--at-space-4);
  cursor: pointer;
}

.entry:hover,
.entry:focus-visible {
  outline: none;
}

.entry__icon {
  flex: 0 0 auto;
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--at-bg-surface-strong);
  color: var(--at-color-body-strong);
}

.entry__icon--task_completion {
  background: var(--at-color-success-soft);
  color: var(--at-color-success);
}

.entry__icon--birthday {
  background: var(--at-color-warm-soft);
  color: var(--at-color-warm);
}

.entry__body {
  flex: 1;
  min-width: 0;
}

.entry__title-row {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  font-size: var(--at-text-body-sm);
}

.entry__title-row strong {
  font-weight: 500;
}

.badge {
  flex: 0 0 auto;
  padding: 1px 8px;
  border-radius: var(--at-radius-pill);
  font-size: 11px;
  background: var(--at-bg-surface-strong);
  color: var(--at-color-muted);
}

.badge--completed {
  background: var(--at-color-success-soft);
  color: var(--at-color-success);
}

.badge--overdue {
  background: var(--at-color-danger-soft);
  color: var(--at-color-danger);
}

.badge--upcoming {
  background: var(--at-color-warning-soft);
  color: var(--at-color-warning);
}

.entry__summary {
  margin-top: 3px;
  font-size: var(--at-text-caption);
  color: var(--at-text-secondary);
  line-height: 1.6;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.entry__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--at-space-3);
  margin-top: var(--at-space-2);
  font-size: 11px;
  color: var(--at-color-muted);
}

.entry__member {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.entry__media {
  display: flex;
  gap: var(--at-space-2);
  margin-top: var(--at-space-2);
}

.entry__media img {
  width: 64px;
  height: 64px;
  object-fit: cover;
  border-radius: var(--at-radius-md);
  border: 1px solid var(--at-color-hairline);
}

.entry__chevron {
  color: var(--at-color-muted-soft);
  align-self: center;
}

.more {
  display: flex;
  justify-content: center;
  padding: var(--at-space-4) 0 var(--at-space-2);
  font-size: var(--at-text-micro);
}

@media (max-width: 639px) {
  .controls__row {
    flex-wrap: wrap;
  }

  .segmented {
    margin-left: 0;
  }

  .search {
    flex: 1;
    margin-left: 0;
  }

  .search input {
    width: 100%;
    min-width: 0;
  }

  .day__items {
    padding-left: var(--at-space-3);
  }

  .entry {
    padding: var(--at-space-3);
  }

  .entry__media img {
    width: 52px;
    height: 52px;
  }
}
</style>
