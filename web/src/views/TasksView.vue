<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { Task, TaskTemplate } from '@/types';
import { useFamilyStore } from '@/stores/family';
import { useTasksStore, type TaskQuery } from '@/stores/tasks';
import { useToastStore } from '@/stores/toast';
import { ApiError } from '@/api/client';
import { todayISO } from '@/utils/chrono';
import { dueLabel } from '@/utils/task';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppButton from '@/components/base/AppButton.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import AppStatusTag from '@/components/base/AppStatusTag.vue';
import AppTabs from '@/components/base/AppTabs.vue';
import TaskDetailDialog from '@/components/business/TaskDetailDialog.vue';
import TaskFormDialog from '@/components/business/TaskFormDialog.vue';
import TaskItem from '@/components/business/TaskItem.vue';
import TemplateManagerDialog from '@/components/business/TemplateManagerDialog.vue';

const route = useRoute();
const router = useRouter();
const tasks = useTasksStore();
const family = useFamilyStore();
const toast = useToastStore();

const tab = ref('list');
const memberFilter = ref('all');
const priorityFilter = ref('');
const search = ref('');
const searchInput = ref('');
const detailOpen = ref(false);
const activeTask = ref<Task | null>(null);
const formOpen = ref(false);
const editingTask = ref<Task | null>(null);
const initialTemplate = ref<TaskTemplate | null>(null);
const templateOpen = ref(false);
const deleteTarget = ref<Task | null>(null);
const boardTasks = ref<Task[]>([]);

const today = computed(() => tasks.today || todayISO());
const loading = computed(() => tasks.loading);

const tabs = computed(() => [
  { key: 'list', label: '列表', count: tasks.counts.open },
  { key: 'board', label: '看板' },
  { key: 'completed', label: '已完成', count: tasks.counts.completed },
]);

const memberOptions = computed(() => [
  { value: 'all', label: '全部成员' },
  { value: 'me', label: '我负责的' },
  ...family.members.filter((m) => m.status === 'active').map((m) => ({ value: String(m.id), label: m.displayName })),
]);

const priorityOptions = [
  { value: '', label: '全部优先级' },
  { value: 'urgent', label: '紧急' },
  { value: 'high', label: '高' },
  { value: 'normal', label: '普通' },
  { value: 'low', label: '低' },
];

const groups = computed(() => {
  const list = tasks.tasks;
  const overdue: Task[] = [];
  const todayItems: Task[] = [];
  const upcoming: Task[] = [];
  const someday: Task[] = [];
  for (const task of list) {
    if (!task.dueDate) someday.push(task);
    else if (task.dueDate < today.value) overdue.push(task);
    else if (task.dueDate === today.value) todayItems.push(task);
    else upcoming.push(task);
  }
  return [
    { key: 'overdue', label: '已逾期', tone: 'danger', items: overdue },
    { key: 'today', label: '今天', tone: 'brand', items: todayItems },
    { key: 'upcoming', label: '接下来', tone: 'muted', items: upcoming },
    { key: 'someday', label: '未排期', tone: 'muted', items: someday },
  ].filter((g) => g.items.length > 0);
});

const boardColumns = computed(() => {
  const list = boardTasks.value.filter((t) => !t.archivedAt);
  return [
    { key: 'todo', label: '待办', items: list.filter((t) => t.status === 'todo') },
    { key: 'in_progress', label: '进行中', items: list.filter((t) => t.status === 'in_progress') },
    { key: 'completed', label: '已完成', items: list.filter((t) => t.status === 'completed') },
  ];
});

const emptyTitle = computed(() => {
  if (search.value) return '没有找到匹配的待办';
  if (tab.value === 'completed') return '还没有完成的待办';
  return '暂时没有待办';
});

const emptyDescription = computed(() => {
  if (search.value) return '换个关键词，或清空筛选再试一次';
  if (tab.value === 'completed') return '完成的待办会出现在这里，方便回顾';
  return '把要做的小事记下来，完成后会有记录';
});

let searchTimer: number | undefined;

watch(tab, () => void reload());
watch(memberFilter, () => void reload());
watch(priorityFilter, () => void reload());

watch(searchInput, (value) => {
  if (searchTimer) window.clearTimeout(searchTimer);
  searchTimer = window.setTimeout(() => {
    search.value = value.trim();
    void reload();
  }, 300);
});

watch(() => route.query.open, () => void handleOpenQuery());
watch(() => route.query.new, (value) => {
  if (value === '1' && !formOpen.value) openCreate();
});

onMounted(async () => {
  await Promise.allSettled([family.load(), tasks.loadTemplates()]);
  await reload();
  await handleOpenQuery();
  if (route.query.new === '1') openCreate();
});

async function reload(): Promise<void> {
  const query: TaskQuery = {
    scope: tab.value === 'completed' ? 'completed' : tab.value === 'board' ? 'all' : 'open',
    assignee: memberFilter.value === 'all' ? undefined : memberFilter.value === 'me' ? 'me' : Number(memberFilter.value),
    priority: priorityFilter.value || undefined,
    q: search.value || undefined,
  };
  try {
    await tasks.load(query);
    if (tab.value === 'board') boardTasks.value = tasks.tasks;
  } catch (err) {
    if (!(err instanceof ApiError)) toast.error('加载待办失败');
  }
}

async function handleOpenQuery(): Promise<void> {
  const raw = route.query.open;
  if (!raw) {
    detailOpen.value = false;
    return;
  }
  const id = Number(raw);
  if (!id) return;
  try {
    const task = await tasks.getTask(id);
    activeTask.value = task;
    detailOpen.value = true;
  } catch {
    toast.error('待办不存在或没有权限');
    removeQuery('open');
  }
}

function removeQuery(key: string): void {
  const query = { ...route.query };
  delete query[key];
  void router.replace({ query });
}

function openDetail(task: Task): void {
  activeTask.value = task;
  detailOpen.value = true;
  void router.replace({ query: { ...route.query, open: String(task.id) } });
}

function closeDetail(): void {
  detailOpen.value = false;
  activeTask.value = null;
  removeQuery('open');
}

function openCreate(): void {
  editingTask.value = null;
  initialTemplate.value = null;
  formOpen.value = true;
  removeQuery('new');
}

function openEdit(task: Task): void {
  closeDetail();
  editingTask.value = task;
  initialTemplate.value = null;
  formOpen.value = true;
}

function useTemplate(template: TaskTemplate): void {
  templateOpen.value = false;
  editingTask.value = null;
  initialTemplate.value = template;
  formOpen.value = true;
}

async function onSaved(task: Task): Promise<void> {
  toast.success(editingTask.value ? '待办已更新' : '待办已创建');
  if (activeTask.value?.id === task.id) activeTask.value = task;
  await reload();
}

async function completeTask(task: Task, actualUserId?: number): Promise<void> {
  try {
    const result = await tasks.completeTask(task.id, actualUserId);
    if (activeTask.value?.id === task.id) activeTask.value = result.task;
    await reload();
    const next = result.nextTask?.dueDate ? `，下次 ${result.nextTask.dueDate.slice(5)}` : '';
    toast.undo(`已完成「${task.title}」${next}`, () => {
      void undoTask(task);
    });
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '完成待办失败');
  }
}

async function undoTask(task: Task): Promise<void> {
  try {
    const result = await tasks.undoTask(task.id);
    if (activeTask.value?.id === task.id) activeTask.value = result.task;
    toast.success(result.nextTaskRemoved ? '已撤销完成并撤回下一次待办' : '已撤销完成');
    await reload();
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '撤销失败');
  }
}

async function refreshTask(task: Task): Promise<void> {
  if (activeTask.value?.id === task.id) {
    try {
      activeTask.value = await tasks.getTask(task.id);
    } catch {
      activeTask.value = task;
    }
  }
  await reload();
}

function requestDelete(task: Task): void {
  deleteTarget.value = task;
}

async function confirmDelete(): Promise<void> {
  if (!deleteTarget.value) return;
  try {
    await tasks.deleteTask(deleteTarget.value.id);
    toast.success('待办已删除');
    if (activeTask.value?.id === deleteTarget.value.id) closeDetail();
    deleteTarget.value = null;
    await reload();
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除待办失败');
  }
}

function memberNames(task: Task): string {
  return task.assignees.map((a) => a.displayName).join('、');
}
</script>

<template>
  <div class="tasks-page">
    <header class="tasks-page__head">
      <div>
        <p class="at-caption-upper">待办</p>
        <h1 class="at-display tasks-page__title">待办清单</h1>
        <p class="tasks-page__sub">
          今天 {{ tasks.counts.today }} 件
          <template v-if="tasks.counts.overdue"> · <span class="is-danger">{{ tasks.counts.overdue }} 件逾期</span></template>
          <template v-if="tasks.counts.open"> · 共 {{ tasks.counts.open }} 件待办</template>
        </p>
      </div>
      <div class="tasks-page__actions">
        <AppButton variant="outline" @click="templateOpen = true">
          <AppIcon name="grid" :size="16" />模板
        </AppButton>
        <AppButton @click="openCreate">
          <AppIcon name="plus" :size="16" />新建待办
        </AppButton>
      </div>
    </header>

    <div class="toolbar">
      <AppTabs v-model="tab" :tabs="tabs" aria-label="待办视图" />
      <div class="toolbar__filters">
        <AppSelect v-model="memberFilter" :options="memberOptions" aria-label="按成员筛选" class="toolbar__select" />
        <AppSelect v-model="priorityFilter" :options="priorityOptions" aria-label="按优先级筛选" class="toolbar__select" />
        <label class="search">
          <AppIcon name="search" :size="16" />
          <input v-model="searchInput" type="search" placeholder="搜索待办" aria-label="搜索待办" />
        </label>
      </div>
    </div>

    <div v-if="loading && !tasks.tasks.length" class="skeleton">
      <div v-for="n in 4" :key="n" class="skeleton__row" />
    </div>

    <div v-else-if="tasks.error && !tasks.tasks.length" class="error-panel">
      <AppIcon name="alert" :size="22" />
      <p>待办加载失败</p>
      <AppButton variant="outline" size="sm" @click="reload">重试</AppButton>
    </div>

    <template v-else>
      <section v-if="tab === 'list'" class="list">
        <div v-for="group in groups" :key="group.key" class="group">
          <div class="group__head">
            <span class="at-caption-upper" :class="group.tone === 'danger' ? 'is-danger' : ''">{{ group.label }}</span>
            <span class="group__count">{{ group.items.length }}</span>
          </div>
          <div class="group__body">
            <TaskItem
              v-for="task in group.items"
              :key="task.id"
              :task="task"
              :today="today"
              @complete="completeTask"
              @undo="undoTask"
              @open="openDetail"
            />
          </div>
        </div>
        <AppEmptyState v-if="!groups.length" :title="emptyTitle" :description="emptyDescription" icon="tasks">
          <AppButton v-if="!search" size="sm" variant="outline" @click="openCreate">新建待办</AppButton>
        </AppEmptyState>
      </section>

      <section v-else-if="tab === 'board'" class="board">
        <div v-for="column in boardColumns" :key="column.key" class="board__col">
          <div class="board__head">
            <span class="at-caption-upper">{{ column.label }}</span>
            <span class="group__count">{{ column.items.length }}</span>
          </div>
          <div class="board__cards">
            <article v-for="task in column.items" :key="task.id" class="bcard">
              <button type="button" class="bcard__main" @click="openDetail(task)">
                <span class="bcard__title">{{ task.title }}</span>
                <span class="bcard__meta" :class="`is-${dueLabel(task, today).tone}`">
                  <AppIcon name="clock" :size="13" />{{ dueLabel(task, today).text }}
                </span>
                <span v-if="memberNames(task)" class="bcard__people">{{ memberNames(task) }}</span>
                <AppStatusTag v-if="task.priority === 'high' || task.priority === 'urgent'" tone="danger">高</AppStatusTag>
              </button>
              <div class="bcard__foot">
                <span class="bcard__avatars">
                  <AppAvatar v-for="person in task.assignees.slice(0, 3)" :key="person.id" :name="person.displayName" :size="22" />
                </span>
                <button
                  v-if="task.status !== 'completed'"
                  type="button"
                  class="bcard__done"
                  :aria-label="`完成 ${task.title}`"
                  @click="completeTask(task)"
                >
                  <AppIcon name="check" :size="13" />
                </button>
                <button
                  v-else
                  type="button"
                  class="bcard__done is-undo"
                  :aria-label="`撤销完成 ${task.title}`"
                  @click="undoTask(task)"
                >
                  <AppIcon name="undo" :size="13" />
                </button>
              </div>
            </article>
            <p v-if="!column.items.length" class="board__empty">暂无</p>
          </div>
        </div>
      </section>

      <section v-else class="list">
        <div v-if="tasks.tasks.length" class="group__body group__body--panel">
          <TaskItem
            v-for="task in tasks.tasks"
            :key="task.id"
            :task="task"
            :today="today"
            @undo="undoTask"
            @open="openDetail"
          />
        </div>
        <AppEmptyState v-else :title="emptyTitle" :description="emptyDescription" icon="check" />
      </section>
    </template>

    <TaskFormDialog
      v-model="formOpen"
      :task="editingTask"
      :templates="tasks.templates"
      :initial-template="initialTemplate"
      :default-date="tab === 'list' ? today : null"
      @saved="onSaved"
    />

    <TaskDetailDialog
      :model-value="detailOpen"
      :task="activeTask"
      :today="today"
      @update:model-value="!$event && closeDetail()"
      @changed="refreshTask"
      @edit="openEdit"
      @remove="requestDelete"
      @complete="completeTask"
      @undo="undoTask"
    />

    <TemplateManagerDialog
      v-model="templateOpen"
      :templates="tasks.templates"
      @use="useTemplate"
      @changed="tasks.loadTemplates(true)"
    />

    <AppConfirmDialog
      :model-value="deleteTarget !== null"
      title="删除待办"
      :message="`确定删除「${deleteTarget?.title ?? ''}」吗？删除后无法恢复。`"
      danger
      confirm-text="删除"
      @update:model-value="deleteTarget = null"
      @confirm="confirmDelete"
    />
  </div>
</template>

<style scoped>
.tasks-page {
  max-width: var(--at-content-max);
  margin: 0 auto;
  padding: var(--at-space-8) var(--at-space-6) var(--at-space-12);
}

.tasks-page__head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--at-space-4);
  padding-bottom: var(--at-space-5);
}

.tasks-page__title {
  margin: var(--at-space-1) 0 var(--at-space-2);
  font-size: var(--at-text-display-md);
}

.tasks-page__sub {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.tasks-page__actions {
  display: flex;
  gap: var(--at-space-2);
  flex: none;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-4);
  padding: var(--at-space-3) 0;
  border-top: 1px solid var(--at-color-hairline);
  border-bottom: 1px solid var(--at-color-hairline);
  flex-wrap: wrap;
}

.toolbar__filters {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  flex-wrap: wrap;
}

.toolbar__select {
  width: 132px;
}

.search {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  height: 38px;
  padding: 0 var(--at-space-3);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  color: var(--at-color-muted);
}

.search:focus-within {
  border-color: var(--at-color-primary);
}

.search input {
  width: 150px;
  border: 0;
  outline: 0;
  background: transparent;
  font-size: var(--at-text-caption);
  color: var(--at-text-primary);
}

.list {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-6);
  padding-top: var(--at-space-6);
}

.group__head {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  padding-bottom: var(--at-space-2);
}

.group__count {
  display: inline-grid;
  place-items: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface-strong);
  font-size: 11px;
  color: var(--at-color-muted);
}

.group__body {
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-lg);
  padding: 0 var(--at-space-3);
  box-shadow: var(--at-shadow-panel);
}

.group__body--panel {
  padding: var(--at-space-2) var(--at-space-4);
}

.is-danger {
  color: var(--at-color-danger);
}

.board {
  display: grid;
  grid-template-columns: repeat(3, minmax(260px, 1fr));
  gap: var(--at-space-4);
  padding-top: var(--at-space-6);
  overflow-x: auto;
}

.board__col {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
}

.board__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--at-space-1);
}

.board__cards {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  padding: var(--at-space-2);
  background: var(--at-bg-canvas-soft);
  border-radius: var(--at-radius-lg);
  min-height: 140px;
}

.board__empty {
  padding: var(--at-space-4);
  text-align: center;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted-soft);
}

.bcard {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  padding: var(--at-space-3);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
  box-shadow: var(--at-shadow-panel);
  transition: box-shadow var(--at-duration-fast) var(--at-ease), transform var(--at-duration-fast) var(--at-ease);
}

.bcard:hover {
  box-shadow: var(--at-shadow-hover);
  transform: translateY(-1px);
}

.bcard__main {
  display: flex;
  flex-direction: column;
  gap: 5px;
  text-align: left;
}

.bcard__title {
  font-size: var(--at-text-body-sm);
  color: var(--at-color-ink);
  line-height: 1.45;
}

.bcard__meta {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.bcard__meta.is-overdue {
  color: var(--at-color-danger);
}

.bcard__meta.is-today {
  color: var(--at-color-brand);
}

.bcard__people {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted-soft);
}

.bcard__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-2);
  padding-top: var(--at-space-2);
  border-top: 1px solid var(--at-color-hairline-soft);
}

.bcard__avatars {
  display: flex;
}

.bcard__avatars > * + * {
  margin-left: -6px;
}

.bcard__done {
  width: 26px;
  height: 26px;
  display: grid;
  place-items: center;
  border: 1.5px solid var(--at-color-hairline-strong);
  border-radius: 50%;
  color: var(--at-color-muted);
}

.bcard__done:hover {
  border-color: var(--at-color-primary);
  background: var(--at-color-primary);
  color: var(--at-text-on-primary);
}

.bcard__done.is-undo {
  background: var(--at-color-primary);
  border-color: var(--at-color-primary);
  color: var(--at-text-on-primary);
}

.skeleton {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  padding-top: var(--at-space-6);
}

.skeleton__row {
  height: 56px;
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface-strong);
  animation: pulse 1.4s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}

.error-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--at-space-3);
  padding: var(--at-space-12) var(--at-space-4);
  color: var(--at-color-muted);
}

@media (max-width: 1023px) {
  .tasks-page {
    padding: var(--at-space-6) var(--at-space-4) var(--at-space-12);
  }

  .board {
    grid-template-columns: repeat(3, minmax(240px, 1fr));
  }
}

@media (max-width: 639px) {
  .tasks-page {
    padding: var(--at-space-5) var(--at-space-4) 96px;
  }

  .tasks-page__head {
    flex-direction: column;
    align-items: stretch;
  }

  .tasks-page__actions {
    justify-content: flex-end;
  }

  .tasks-page__title {
    font-size: var(--at-text-display-sm);
  }

  .toolbar__select {
    flex: 1;
    width: auto;
  }

  .search {
    flex: 1 1 100%;
  }

  .search input {
    width: 100%;
  }

  .board {
    grid-template-columns: repeat(3, 260px);
  }
}
</style>
