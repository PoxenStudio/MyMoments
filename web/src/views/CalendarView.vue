<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { Birthday, CalendarDayMeta, CalendarEvent, Task } from '@/types';
import { api, ApiError } from '@/api/client';
import { useCalendarStore } from '@/stores/calendar';
import { useFamilyStore } from '@/stores/family';
import { useToastStore } from '@/stores/toast';
import { todayISO } from '@/utils/chrono';
import { birthdayRoute } from '@/utils/birthday';
import {
  addDays, addMonths, dayLabel, isSameMonth, lunarDayLabel, monthGrid, monthLabel, monthRange,
  parseDate, startOfWeek, timeOf, weekRange, agendaRange, WEEKDAYS,
} from '@/utils/calendar';
import AppButton from '@/components/base/AppButton.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppTabs from '@/components/base/AppTabs.vue';
import EventDetailDialog from '@/components/business/EventDetailDialog.vue';
import EventFormDialog from '@/components/business/EventFormDialog.vue';

type CalendarViewMode = 'month' | 'week' | 'day' | 'agenda';
type LayerKey = 'events' | 'tasks' | 'birthdays';

interface DayEntry {
  key: string;
  kind: 'event' | 'task' | 'birthday';
  date: string;
  title: string;
  time: string;
  allDay: boolean;
  color: string;
  event?: CalendarEvent;
  task?: Task;
  birthday?: Birthday;
}

const route = useRoute();
const router = useRouter();
const calendar = useCalendarStore();
const family = useFamilyStore();
const toast = useToastStore();

const today = todayISO();
const view = ref<CalendarViewMode>('month');
const anchor = ref(today);
const selectedDate = ref(today);
const loading = ref(false);
const calendarTasks = ref<Task[]>([]);
const birthdaysInRange = ref<Birthday[]>([]);
const dayMeta = ref<Record<string, CalendarDayMeta>>({});
const formOpen = ref(false);
const editingEvent = ref<CalendarEvent | null>(null);
const detailOpen = ref(false);
const activeEvent = ref<CalendarEvent | null>(null);
const deleteTarget = ref<{ event: CalendarEvent; scope: 'this' | 'all' } | null>(null);
const deleting = ref(false);
const defaultHour = ref('09:00');

const storedLayers = (() => {
  try {
    return JSON.parse(localStorage.getItem('mm.calendar.layers') || '{}') as Partial<Record<LayerKey, boolean>>;
  } catch {
    return {};
  }
})();
const layers = reactive<Record<LayerKey, boolean>>({
  events: storedLayers.events ?? true,
  tasks: storedLayers.tasks ?? true,
  birthdays: storedLayers.birthdays ?? true,
});

const tabs = [
  { key: 'month', label: '月' },
  { key: 'week', label: '周' },
  { key: 'day', label: '日' },
  { key: 'agenda', label: '议程' },
];

const range = computed(() => {
  if (view.value === 'week') return weekRange(anchor.value);
  if (view.value === 'day') return { from: anchor.value, to: anchor.value };
  if (view.value === 'agenda') return agendaRange(anchor.value);
  const month = monthRange(anchor.value);
  const from = startOfWeek(month.from);
  return { from, to: addDays(from, 41) };
});

const grid = computed(() => monthGrid(anchor.value));
const weekDates = computed(() => {
  const from = startOfWeek(anchor.value);
  return Array.from({ length: 7 }, (_, index) => addDays(from, index));
});

const title = computed(() => {
  if (view.value === 'month') return monthLabel(anchor.value);
  if (view.value === 'week') {
    const r = range.value;
    return `${r.from.slice(5).replace('-', '/')} – ${r.to.slice(5).replace('-', '/')}`;
  }
  if (view.value === 'day') return dayLabel(anchor.value);
  return '接下来 45 天';
});

const layerCounts = computed(() => ({
  events: calendar.events.length,
  tasks: calendarTasks.value.length,
  birthdays: birthdayEntries.value.length,
}));

const birthdayEntries = computed<DayEntry[]>(() => {
  if (!layers.birthdays) return [];
  const out: DayEntry[] = [];
  for (const birthday of birthdaysInRange.value) {
    for (const occurrence of birthday.occurrences ?? []) {
      let title = `@${birthday.name} 的生日`;
      if (occurrence.calendar === 'both') title += '（公历/农历同日）';
      else if (occurrence.calendar === 'lunar') title += '（农历）';
      out.push({
        key: `birthday-${birthday.id}-${occurrence.date}-${occurrence.calendar}`,
        kind: 'birthday',
        date: occurrence.date,
        title,
        time: '全天',
        allDay: true,
        color: 'var(--at-color-warning)',
        birthday,
      });
    }
  }
  return out;
});

const entriesByDate = computed(() => {
  const map = new Map<string, DayEntry[]>();
  const push = (entry: DayEntry): void => {
    const list = map.get(entry.date) ?? [];
    list.push(entry);
    map.set(entry.date, list);
  };
  if (layers.events) {
    for (const event of calendar.events) {
      const date = event.occurrenceDate ?? event.startLocal.slice(0, 10);
      const time = event.allDay ? '全天' : timeOf(event.startLocal);
      push({
        key: `event-${event.id}-${date}`,
        kind: 'event',
        date,
        title: event.title,
        time,
        allDay: event.allDay,
        color: event.color || 'var(--at-color-ink)',
        event,
      });
    }
  }
  if (layers.tasks) {
    for (const task of calendarTasks.value) {
      if (!task.dueDate) continue;
      push({
        key: `task-${task.id}-${task.dueDate}`,
        kind: 'task',
        date: task.dueDate,
        title: task.title,
        time: task.dueTime || '待办',
        allDay: !task.dueTime,
        color: task.color || 'var(--at-color-brand)',
        task,
      });
    }
  }
  for (const entry of birthdayEntries.value) push(entry);
  for (const list of map.values()) {
    list.sort((a, b) => {
      if (a.allDay !== b.allDay) return a.allDay ? -1 : 1;
      if (a.time !== b.time) return a.time.localeCompare(b.time);
      return a.title.localeCompare(b.title, 'zh-CN');
    });
  }
  return map;
});

const selectedEntries = computed(() => entriesByDate.value.get(selectedDate.value) ?? []);
const agendaDays = computed(() => {
  const r = range.value;
  const days: { date: string; entries: DayEntry[] }[] = [];
  for (let date = r.from; date <= r.to; date = addDays(date, 1)) {
    const entries = entriesByDate.value.get(date) ?? [];
    if (!entries.length && date !== today) continue;
    days.push({ date, entries });
  }
  return days;
});
const isSelectedPast = computed(() => selectedDate.value < today);
const emptyDescription = computed(() => {
  if (!layers.events || !layers.tasks || !layers.birthdays) return '当前隐藏了部分图层，可打开上方图层查看。';
  return '把家人的安排记下来，日历会替你记着。';
});

watch([view, anchor], () => void reload());
watch(layers, (value) => {
  try {
    localStorage.setItem('mm.calendar.layers', JSON.stringify(value));
  } catch {
    // 本地存储不可用时仅保留当前会话内的图层偏好。
  }
}, { deep: true });
watch(view, (mode) => {
  if (mode === 'day') selectedDate.value = anchor.value;
});
watch(() => route.query.open, () => void handleOpenQuery());

/** 输入框、弹窗或带修饰键时不接管方向键，避免干扰正常编辑与系统快捷键。 */
function shouldIgnoreKey(event: KeyboardEvent): boolean {
  if (event.metaKey || event.ctrlKey || event.altKey) return true;
  if (formOpen.value || detailOpen.value || deleteTarget.value) return true;
  const el = event.target as HTMLElement | null;
  if (!el) return false;
  return el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName);
}

function onKeydown(event: KeyboardEvent): void {
  if (shouldIgnoreKey(event)) return;
  if (event.key === 'ArrowLeft' || event.key === 'PageUp') {
    event.preventDefault();
    navigate(-1);
  } else if (event.key === 'ArrowRight' || event.key === 'PageDown') {
    event.preventDefault();
    navigate(1);
  }
}

onMounted(async () => {
  const media = window.matchMedia('(max-width: 767px)');
  if (media.matches) view.value = 'agenda';
  void family.load().catch(() => undefined);
  await Promise.allSettled([reload()]);
  await handleOpenQuery();
  if (route.query.date) {
    selectedDate.value = String(route.query.date);
    anchor.value = String(route.query.date);
  }
  document.addEventListener('keydown', onKeydown);
});

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown);
});

async function reload(): Promise<void> {
  const r = range.value;
  loading.value = true;
  try {
    const [, taskRes, metaRes, birthdayRes] = await Promise.all([
      calendar.loadRange(r.from, r.to),
      api.get<{ items: Task[] }>('/tasks', { scope: 'open', from: r.from, to: r.to, limit: 200 }),
      api
        .get<{ days: CalendarDayMeta[] }>('/calendar/meta', { from: r.from, to: r.to })
        .catch(() => ({ days: [] as CalendarDayMeta[] })),
      api
        .get<{ items: Birthday[] }>('/birthdays', { from: r.from, to: r.to })
        .catch(() => ({ items: [] as Birthday[] })),
    ]);
    calendarTasks.value = taskRes.items;
    birthdaysInRange.value = birthdayRes.items;
    dayMeta.value = Object.fromEntries(metaRes.days.map((item) => [item.date, item]));
  } catch (err) {
    if (!(err instanceof ApiError)) toast.error('加载日历失败');
  } finally {
    loading.value = false;
  }
}

async function handleOpenQuery(): Promise<void> {
  const raw = route.query.open;
  if (!raw) return;
  const id = Number(raw);
  if (!id) return;
  try {
    activeEvent.value = await calendar.getEvent(id);
    detailOpen.value = true;
  } catch {
    toast.error('事件不存在或没有权限');
  }
}

function navigate(direction: -1 | 1): void {
  if (view.value === 'month') anchor.value = addMonths(anchor.value, direction);
  else if (view.value === 'week') anchor.value = addDays(anchor.value, direction * 7);
  else if (view.value === 'day') anchor.value = addDays(anchor.value, direction);
  else anchor.value = addDays(anchor.value, direction * 45);
  if (view.value !== 'month') selectedDate.value = anchor.value;
}

function goToday(): void {
  anchor.value = today;
  selectedDate.value = today;
}

function selectDate(date: string): void {
  selectedDate.value = date;
  if (view.value === 'month' && !isSameMonth(date, anchor.value)) anchor.value = date;
}

function cellEntries(date: string): DayEntry[] {
  return entriesByDate.value.get(date) ?? [];
}

function metaFor(date: string): CalendarDayMeta | undefined {
  return dayMeta.value[date];
}

function daySubLabel(date: string): string {
  const meta = metaFor(date);
  if (!meta) return '';
  if (meta.holiday || meta.festival) return meta.holiday || meta.festival || '';
  if (meta.lunarDay === 1) return meta.lunar;
  return lunarDayLabel(meta.lunarDay);
}

function dayBadge(date: string): string {
  const meta = metaFor(date);
  if (!meta) return '';
  if (meta.offDay) return '休';
  if (meta.workday) return '班';
  return '';
}

function dayMetaDetail(date: string): string {
  const meta = metaFor(date);
  if (!meta) return '';
  const parts = [`农历${meta.lunar}`];
  if (meta.holiday) parts.push(meta.holiday);
  if (meta.festival && meta.festival !== meta.holiday) parts.push(meta.festival);
  if (meta.offDay) parts.push('休假');
  else if (meta.workday) parts.push('调休上班');
  return parts.join(' · ');
}

function openCreate(date = selectedDate.value, hour = '09:00'): void {
  editingEvent.value = null;
  defaultHour.value = hour;
  selectedDate.value = date;
  formOpen.value = true;
}

function openDetail(event: CalendarEvent): void {
  activeEvent.value = event;
  detailOpen.value = true;
}

function openEdit(event: CalendarEvent): void {
  detailOpen.value = false;
  editingEvent.value = event;
  defaultHour.value = event.allDay ? '09:00' : timeOf(event.startLocal);
  formOpen.value = true;
}

function onSaved(): void {
  toast.success(editingEvent.value ? '事件已更新' : '事件已创建');
  formOpen.value = false;
  void reload();
}

function requestDelete(event: CalendarEvent, scope: 'this' | 'all'): void {
  deleteTarget.value = { event, scope };
}

async function confirmDelete(): Promise<void> {
  if (!deleteTarget.value) return;
  deleting.value = true;
  try {
    const { event, scope } = deleteTarget.value;
    await calendar.deleteEvent(event.id, scope, event.occurrenceDate);
    toast.success(scope === 'this' ? '已删除本次事件' : '事件已删除');
    deleteTarget.value = null;
    detailOpen.value = false;
    activeEvent.value = null;
    await reload();
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除事件失败');
  } finally {
    deleting.value = false;
  }
}

function goTask(task: Task): void {
  void router.push({ path: '/tasks', query: { open: String(task.id) } });
}

async function completeTask(task: Task): Promise<void> {
  try {
    await api.post(`/tasks/${task.id}/completion`, { actualUserId: null });
    toast.success(`已完成「${task.title}」`);
    await reload();
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '完成待办失败');
  }
}

function goBirthday(birthday: Birthday): void {
  void router.push(birthdayRoute(birthday));
}

function goTimeline(): void {
  void router.push({ path: '/timeline', query: { anchor: selectedDate.value } });
}

function entryClick(entry: DayEntry): void {
  if (entry.event) openDetail(entry.event);
  else if (entry.task) goTask(entry.task);
  else if (entry.birthday) goBirthday(entry.birthday);
}

function pillLabel(entry: DayEntry): string {
  return entry.allDay ? entry.title : `${entry.time} ${entry.title}`;
}
</script>

<template>
  <div class="calendar-page">
    <header class="calendar-page__head">
      <div>
        <p class="at-caption-upper">行事历</p>
        <h1 class="at-display calendar-page__title">家的日程</h1>
        <p class="calendar-page__sub">{{ title }}</p>
      </div>
      <div class="calendar-page__actions">
        <AppButton variant="outline" :disabled="loading" aria-label="刷新" @click="reload">
          <AppIcon name="refresh" :size="16" :class="{ 'is-spinning': loading }" />刷新
        </AppButton>
        <AppButton variant="outline" @click="goToday">今天</AppButton>
        <AppButton @click="openCreate()"><AppIcon name="plus" :size="16" />新建事件</AppButton>
      </div>
    </header>

    <div class="toolbar">
      <div class="toolbar__nav">
        <button type="button" class="nav-btn" aria-label="上一周期" @click="navigate(-1)"><AppIcon name="chevron-left" :size="18" /></button>
        <AppTabs v-model="view" :tabs="tabs" aria-label="日历视图" />
        <button type="button" class="nav-btn" aria-label="下一周期" @click="navigate(1)"><AppIcon name="chevron-right" :size="18" /></button>
      </div>
      <div class="layer-bar" aria-label="日历图层">
        <button type="button" class="layer" :class="{ 'is-on': layers.events }" :aria-pressed="layers.events" @click="layers.events = !layers.events">
          <span class="layer__dot is-event" />事件<span class="layer__count">{{ layerCounts.events }}</span>
        </button>
        <button type="button" class="layer" :class="{ 'is-on': layers.tasks }" :aria-pressed="layers.tasks" @click="layers.tasks = !layers.tasks">
          <span class="layer__dot is-task" />待办<span class="layer__count">{{ layerCounts.tasks }}</span>
        </button>
        <button type="button" class="layer" :class="{ 'is-on': layers.birthdays }" :aria-pressed="layers.birthdays" @click="layers.birthdays = !layers.birthdays">
          <span class="layer__dot is-birthday" />生日<span class="layer__count">{{ layerCounts.birthdays }}</span>
        </button>
      </div>
    </div>

    <div v-if="loading && !calendar.events.length" class="skeleton">
      <div v-for="n in 5" :key="n" class="skeleton__row" />
    </div>

    <section v-else-if="view === 'month'" class="month at-card">
      <div class="month__weekdays">
        <span v-for="day in WEEKDAYS" :key="day">{{ day }}</span>
      </div>
      <div class="month__grid">
        <button
          v-for="date in grid.flat()"
          :key="date"
          type="button"
          class="cell"
          :class="{ 'is-out': !isSameMonth(date, anchor), 'is-today': date === today, 'is-selected': date === selectedDate }"
          @click="selectDate(date)"
        >
          <span class="cell__top">
            <span class="cell__num">{{ Number(date.slice(8)) }}</span>
            <span v-if="daySubLabel(date)" class="cell__lunar" :class="{ 'is-holiday': !!metaFor(date)?.holiday }">{{ daySubLabel(date) }}</span>
            <span v-if="dayBadge(date)" class="cell__badge" :class="{ 'is-off': metaFor(date)?.offDay }">{{ dayBadge(date) }}</span>
          </span>
          <span class="cell__items">
            <span
              v-for="entry in cellEntries(date).slice(0, 2)"
              :key="entry.key"
              class="pill"
              :style="{ '--pill-color': entry.color }"
            >
              <span class="pill__dot" />
              <span class="pill__text">{{ pillLabel(entry) }}</span>
            </span>
            <span v-if="cellEntries(date).length > 2" class="cell__more">+{{ cellEntries(date).length - 2 }}</span>
          </span>
        </button>
      </div>
    </section>

    <section v-else-if="view === 'week'" class="week">
      <div v-for="date in weekDates" :key="date" class="week__col at-card" :class="{ 'is-today': date === today }">
        <button type="button" class="week__head" @click="selectDate(date)">
          <span class="at-caption-upper">{{ WEEKDAYS[(parseDate(date).getDay() + 6) % 7] }}</span>
          <span class="week__date">{{ Number(date.slice(8)) }}</span>
        </button>
        <div class="week__body">
          <button
            v-for="entry in cellEntries(date)"
            :key="entry.key"
            type="button"
            class="entry"
            :class="`is-${entry.kind}`"
            :style="{ '--entry-color': entry.color }"
            @click="entryClick(entry)"
          >
            <span class="entry__time">{{ entry.time }}</span>
            <span class="entry__title">{{ entry.title }}</span>
          </button>
          <button type="button" class="week__add" aria-label="在新事件中添加" @click="openCreate(date)">+ 添加</button>
        </div>
      </div>
    </section>

    <section v-else-if="view === 'day'" class="day at-card">
      <div class="day__head">
        <div>
          <h2 class="day__title">{{ dayLabel(anchor) }}</h2>
          <p v-if="dayMetaDetail(anchor)" class="day__meta">{{ dayMetaDetail(anchor) }}</p>
        </div>
        <div class="day__actions">
          <AppButton v-if="anchor < today" variant="ghost" size="sm" @click="goTimeline">在时光轴中查看</AppButton>
          <AppButton variant="outline" size="sm" @click="openCreate(anchor)"><AppIcon name="plus" :size="14" />添加</AppButton>
        </div>
      </div>
      <div v-if="!cellEntries(anchor).length" class="day__empty">
        <AppEmptyState icon="calendar" title="这一天还没有安排" :description="emptyDescription">
          <AppButton variant="outline" size="sm" @click="openCreate(anchor)">新建事件</AppButton>
        </AppEmptyState>
      </div>
      <div v-else class="day__list">
        <button
          v-for="entry in cellEntries(anchor)"
          :key="entry.key"
          type="button"
          class="entry is-large"
          :class="`is-${entry.kind}`"
          :style="{ '--entry-color': entry.color }"
          @click="entryClick(entry)"
        >
          <span class="entry__time">{{ entry.time }}</span>
          <span class="entry__title">{{ entry.title }}</span>
          <span v-if="entry.task" class="entry__check" aria-hidden="true" @click.stop="completeTask(entry.task)">
            <AppIcon name="check" :size="15" />
          </span>
        </button>
      </div>
    </section>

    <section v-else class="agenda at-card">
      <div v-for="day in agendaDays" :key="day.date" class="agenda__group">
        <div class="agenda__head">
          <h2 class="agenda__date">{{ dayLabel(day.date) }}</h2>
          <span v-if="daySubLabel(day.date)" class="agenda__meta">{{ daySubLabel(day.date) }}</span>
          <span v-if="day.date === today" class="agenda__today">今天</span>
        </div>
        <p v-if="!day.entries.length" class="agenda__empty">今天暂无安排</p>
        <div v-else class="agenda__list">
          <button
            v-for="entry in day.entries"
            :key="entry.key"
            type="button"
            class="entry is-large"
            :class="`is-${entry.kind}`"
            :style="{ '--entry-color': entry.color }"
            @click="entryClick(entry)"
          >
            <span class="entry__time">{{ entry.time }}</span>
            <span class="entry__title">{{ entry.title }}</span>
            <span v-if="entry.task" class="entry__check" aria-hidden="true" @click.stop="completeTask(entry.task)">
              <AppIcon name="check" :size="15" />
            </span>
          </button>
        </div>
      </div>
    </section>

    <section v-if="view === 'month'" class="day-panel at-card">
      <div class="day-panel__head">
        <div>
          <p class="at-caption-upper">当日安排</p>
          <h2 class="day-panel__title">{{ dayLabel(selectedDate) }}</h2>
          <p v-if="dayMetaDetail(selectedDate)" class="day-panel__meta">{{ dayMetaDetail(selectedDate) }}</p>
        </div>
        <div class="day-panel__actions">
          <AppButton v-if="isSelectedPast" variant="ghost" size="sm" @click="goTimeline">在时光轴中查看</AppButton>
          <AppButton variant="outline" size="sm" @click="openCreate(selectedDate)"><AppIcon name="plus" :size="14" />添加</AppButton>
        </div>
      </div>
      <p v-if="!selectedEntries.length" class="day-panel__hint">这一天还没有安排。{{ emptyDescription }}</p>
      <div v-else class="day-panel__list">
        <button
          v-for="entry in selectedEntries"
          :key="entry.key"
          type="button"
          class="entry is-large"
          :class="`is-${entry.kind}`"
          :style="{ '--entry-color': entry.color }"
          @click="entryClick(entry)"
        >
          <span class="entry__time">{{ entry.time }}</span>
          <span class="entry__title">{{ entry.title }}</span>
          <span v-if="entry.task" class="entry__check" aria-hidden="true" @click.stop="completeTask(entry.task)">
            <AppIcon name="check" :size="15" />
          </span>
        </button>
      </div>
    </section>

    <EventFormDialog
      v-model="formOpen"
      :event="editingEvent"
      :default-date="selectedDate"
      :default-hour="defaultHour"
      @saved="onSaved"
    />

    <EventDetailDialog
      :model-value="detailOpen"
      :event="activeEvent"
      @update:model-value="detailOpen = $event"
      @edit="openEdit"
      @remove="requestDelete"
    />

    <AppConfirmDialog
      :model-value="deleteTarget !== null"
      title="删除事件"
      :message="deleteTarget?.scope === 'this'
        ? `只删除「${deleteTarget?.event.title}」这一次发生项？`
        : `确定删除「${deleteTarget?.event.title}」${deleteTarget?.event.recurrenceRule ? '整个重复系列' : ''}吗？删除后无法恢复。`"
      danger
      confirm-text="删除"
      :loading="deleting"
      @update:model-value="deleteTarget = null"
      @confirm="confirmDelete"
    />
  </div>
</template>

<style scoped>
.calendar-page {
  max-width: var(--at-content-max);
  margin: 0 auto;
  padding: var(--at-space-8) var(--at-space-6) var(--at-space-12);
}

.calendar-page__head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--at-space-4);
  padding-bottom: var(--at-space-5);
}

.calendar-page__title {
  margin: var(--at-space-1) 0 var(--at-space-2);
  font-size: var(--at-text-display-md);
}

.calendar-page__sub {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.calendar-page__actions {
  display: flex;
  gap: var(--at-space-2);
  flex: none;
}

.is-spinning {
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
  padding: var(--at-space-3) 0;
  border-top: 1px solid var(--at-color-hairline);
  border-bottom: 1px solid var(--at-color-hairline);
  flex-wrap: wrap;
}

.toolbar__nav {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
}

.nav-btn {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: 50%;
  background: var(--at-bg-surface);
  color: var(--at-color-muted);
}

.nav-btn:hover {
  color: var(--at-color-ink);
  border-color: var(--at-color-ink);
}

.layer-bar {
  display: flex;
  gap: var(--at-space-2);
  flex-wrap: wrap;
}

.layer {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 var(--at-space-3);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface);
  color: var(--at-color-muted-soft);
  font-size: var(--at-text-caption);
}

.layer.is-on {
  color: var(--at-text-primary);
  border-color: var(--at-color-hairline-strong);
}

.layer__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.layer__dot.is-event { color: var(--at-color-ink); }
.layer__dot.is-task { color: var(--at-color-brand); }
.layer__dot.is-birthday { color: var(--at-color-warning); }

.layer__count {
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  color: var(--at-color-muted);
}

.month {
  margin-top: var(--at-space-4);
  border-radius: var(--at-radius-xl);
  overflow: hidden;
}

.month__weekdays {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  border-bottom: 1px solid var(--at-color-hairline);
  background: var(--at-bg-canvas-soft);
}

.month__weekdays span {
  padding: var(--at-space-2);
  text-align: center;
  font-size: var(--at-text-micro);
  font-weight: 500;
  color: var(--at-color-muted);
}

.month__grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
}

.cell {
  min-height: clamp(88px, 9.5vh, 112px);
  padding: var(--at-space-2);
  border-right: 1px solid var(--at-color-hairline-soft);
  border-bottom: 1px solid var(--at-color-hairline-soft);
  display: flex;
  flex-direction: column;
  gap: 5px;
  text-align: left;
  background: transparent;
  transition: background var(--at-duration-fast) var(--at-ease);
}

.cell:nth-child(7n) { border-right: 0; }
.cell:hover { background: var(--at-bg-canvas-soft); }
.cell.is-out {
  background: var(--at-bg-outside);
}
.cell.is-selected { background: var(--at-bg-surface-strong); }

.cell__num {
  display: inline-grid;
  place-items: center;
  min-width: 26px;
  height: 26px;
  padding: 0 4px;
  border-radius: var(--at-radius-pill);
  font-size: var(--at-text-body);
  font-weight: 500;
  line-height: 1;
  color: var(--at-color-ink);
  font-variant-numeric: tabular-nums;
}

.cell.is-out .cell__num {
  font-size: var(--at-text-caption);
  font-weight: 400;
  color: var(--at-color-muted);
}

.cell.is-today .cell__num {
  background: var(--at-color-primary);
  color: var(--at-text-on-primary);
  font-weight: 600;
}

.cell__top {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
}

.cell__lunar {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--at-text-micro);
  color: var(--at-text-secondary);
}

.cell.is-out .cell__lunar {
  color: var(--at-color-muted);
}

.cell__lunar.is-holiday {
  color: var(--at-color-warm);
}

.cell__badge {
  flex: none;
  display: inline-grid;
  place-items: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 5px;
  font-size: 10px;
  line-height: 1;
  background: var(--at-bg-canvas-soft);
  color: var(--at-color-muted);
}

.cell__badge.is-off {
  background: var(--at-color-danger-soft);
  color: var(--at-color-danger);
}

.day__meta,
.day-panel__meta {
  margin-top: 2px;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.agenda__meta {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.cell__items {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.pill {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
  font-size: 11px;
  color: var(--at-text-secondary);
}

.pill__dot {
  flex: none;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--pill-color);
}

.pill__text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cell__more {
  font-size: 11px;
  color: var(--at-color-muted);
  padding-left: 10px;
}

.day-panel {
  margin-top: var(--at-space-6);
  border-radius: var(--at-radius-xl);
  padding: var(--at-space-4) var(--at-space-5);
}

.day-panel__head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--at-space-3);
  flex-wrap: wrap;
}

.day-panel__title {
  margin-top: 2px;
  font-family: var(--at-font-display);
  font-weight: 400;
  font-size: var(--at-text-title-sm);
}

.day-panel__actions {
  display: flex;
  gap: var(--at-space-2);
}

.day-panel__hint {
  padding: var(--at-space-4) 0 var(--at-space-1);
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.day-panel__list,
.day__list,
.agenda__list {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-1);
  padding-top: var(--at-space-3);
}

.entry {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  width: 100%;
  padding: var(--at-space-2) var(--at-space-3);
  border-radius: var(--at-radius-md);
  background: transparent;
  text-align: left;
  position: relative;
  min-height: 38px;
}

.entry:hover {
  background: var(--at-bg-canvas-soft);
}

.entry.is-large {
  padding: var(--at-space-3);
  border: 1px solid var(--at-color-hairline-soft);
  min-height: var(--at-touch-min);
}

.entry.is-large::before {
  content: '';
  position: absolute;
  left: 0;
  top: 10px;
  bottom: 10px;
  width: 3px;
  border-radius: var(--at-radius-pill);
  background: var(--entry-color);
}

.entry__time {
  flex: none;
  width: 58px;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
  font-variant-numeric: tabular-nums;
}

.entry__title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--at-text-caption);
  color: var(--at-text-primary);
}

.entry.is-large .entry__title {
  font-size: var(--at-text-body-sm);
}

.entry__check {
  flex: none;
  width: 26px;
  height: 26px;
  display: grid;
  place-items: center;
  border: 1.5px solid var(--at-color-hairline-strong);
  border-radius: 50%;
  color: var(--at-color-muted);
}

.entry__check:hover {
  border-color: var(--at-color-primary);
  background: var(--at-color-primary);
  color: var(--at-text-on-primary);
}

.week {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: var(--at-space-2);
  margin-top: var(--at-space-4);
}

.week__col {
  border-radius: var(--at-radius-lg);
  min-height: 300px;
  display: flex;
  flex-direction: column;
}

.week__col.is-today {
  border-color: var(--at-color-ink);
}

.week__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--at-space-2) var(--at-space-3);
  border-bottom: 1px solid var(--at-color-hairline-soft);
  background: transparent;
  border-radius: var(--at-radius-lg) var(--at-radius-lg) 0 0;
}

.week__date {
  font-family: var(--at-font-display);
  font-size: var(--at-text-title-sm);
}

.week__col.is-today .week__date {
  color: var(--at-color-brand);
}

.week__body {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: var(--at-space-2);
  flex: 1;
}

.week__add {
  margin-top: auto;
  padding: var(--at-space-2);
  font-size: 11px;
  color: var(--at-color-muted-soft);
  text-align: left;
}

.week__add:hover {
  color: var(--at-color-ink);
}

.day,
.agenda {
  margin-top: var(--at-space-5);
  border-radius: var(--at-radius-xl);
  padding: var(--at-space-5);
}

.day__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
  flex-wrap: wrap;
}

.day__title {
  font-family: var(--at-font-display);
  font-weight: 400;
  font-size: var(--at-text-display-sm);
}

.day__actions {
  display: flex;
  gap: var(--at-space-2);
}

.day__empty {
  padding-top: var(--at-space-2);
}

.agenda__group + .agenda__group {
  border-top: 1px solid var(--at-color-hairline-soft);
  margin-top: var(--at-space-4);
  padding-top: var(--at-space-4);
}

.agenda__head {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
}

.agenda__date {
  font-family: var(--at-font-display);
  font-weight: 400;
  font-size: var(--at-text-title-sm);
}

.agenda__today {
  padding: 2px 8px;
  border-radius: var(--at-radius-pill);
  background: var(--at-color-brand-soft);
  color: var(--at-color-brand);
  font-size: 11px;
}

.agenda__empty {
  margin-top: var(--at-space-2);
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.skeleton {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  padding-top: var(--at-space-6);
}

.skeleton__row {
  height: 72px;
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface-strong);
  animation: pulse 1.4s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}

@media (max-width: 1023px) {
  .calendar-page {
    padding: var(--at-space-6) var(--at-space-4) var(--at-space-12);
  }

  .week {
    grid-template-columns: repeat(7, 160px);
    overflow-x: auto;
    padding-bottom: var(--at-space-2);
  }
}

@media (max-width: 639px) {
  .calendar-page {
    padding: var(--at-space-5) var(--at-space-4) 96px;
  }

  .calendar-page__head {
    flex-direction: column;
    align-items: stretch;
  }

  .calendar-page__actions {
    justify-content: flex-end;
  }

  .calendar-page__title {
    font-size: var(--at-text-display-sm);
  }

  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar__nav {
    justify-content: space-between;
  }

  .layer-bar {
    justify-content: space-between;
  }

  .month__weekdays span {
    padding: var(--at-space-1);
  }

  .cell {
    min-width: 0;
    min-height: 74px;
    padding: var(--at-space-1);
    gap: 3px;
  }

  .cell__top {
    flex-wrap: wrap;
    gap: 2px 3px;
  }

  .cell__num {
    min-width: 24px;
    width: 24px;
    height: 22px;
    padding: 0 3px;
  }

  .cell__badge {
    order: 1;
    min-width: 13px;
    height: 14px;
    padding: 0 2px;
    font-size: 9px;
  }

  .cell__lunar {
    order: 2;
    flex: 0 0 100%;
    white-space: normal;
    overflow: visible;
    line-height: 1.25;
  }

  .pill__text,
  .cell__more {
    display: none;
  }

  .pill {
    gap: 3px;
  }

  .pill__dot {
    width: 6px;
    height: 6px;
  }

  .cell__items {
    flex-direction: row;
    flex-wrap: wrap;
    gap: 3px;
  }

  .day,
  .agenda {
    padding: var(--at-space-4);
  }

  .day__head,
  .day-panel__head {
    flex-direction: column;
    align-items: stretch;
  }

  .day__actions,
  .day-panel__actions {
    justify-content: flex-end;
  }

  .entry__time {
    width: 48px;
  }
}
</style>
