<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { ApiError, api } from '@/api/client';
import { useToastStore } from '@/stores/toast';
import { asData, type TasksData } from '@/cards/types';
import { formatShortDate, todayISO } from '@/utils/chrono';
import { useAuthStore } from '@/stores/auth';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import type { Task } from '@/types';

const props = defineProps<{ data: unknown }>();
const emit = defineEmits<{ refresh: [] }>();
const router = useRouter();
const toast = useToastStore();
const auth = useAuthStore();
const d = computed(() => asData<TasksData>(props.data));
const completing = ref<number | null>(null);
const today = computed(() => todayISO());

function overdue(task: Task): boolean {
  return !!task.dueDate && task.dueDate < today.value;
}

function open(task: Task): void {
  void router.push({ name: 'tasks', query: { open: String(task.id) } });
}

async function complete(task: Task): Promise<void> {
  if (completing.value) return;
  completing.value = task.id;
  try {
    await api.post(`/tasks/${task.id}/completion`, {});
    toast.success(`已完成「${task.title}」`);
    emit('refresh');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '完成待办失败');
  } finally {
    completing.value = null;
  }
}
</script>

<template>
  <div v-if="!d?.items.length">
    <AppEmptyState title="没有待办" description="把想做的事记下来，家里人会看到" icon="tasks" />
  </div>
  <ul v-else class="tasks">
    <li v-for="task in d.items" :key="task.id" class="tasks__row">
      <button
        type="button"
        class="tasks__check"
        :aria-label="`完成 ${task.title}`"
        :disabled="completing === task.id"
        @click="complete(task)"
      />
      <button type="button" class="tasks__body" @click="open(task)">
        <span class="tasks__title">{{ task.title }}</span>
        <span class="tasks__meta">
          <span v-if="task.dueDate" class="tasks__due" :class="{ 'is-overdue': overdue(task) }">
            <AppIcon name="clock" :size="13" />{{ task.dueDate === today ? '今天' : formatShortDate(task.dueDate) }}
          </span>
          <span v-if="task.assignees.length" class="tasks__assignees">
            <span
              v-for="u in task.assignees"
              :key="u.id"
              class="tasks__assignee"
              :class="u.id === auth.user?.id ? 'is-self' : 'is-other'"
            >{{ u.displayName }}</span>
          </span>
        </span>
      </button>
    </li>
  </ul>
</template>

<style scoped>
.tasks {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.tasks__row {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  min-height: 48px;
  border-bottom: 1px solid var(--at-color-hairline);
}

.tasks__row:last-child {
  border-bottom: 0;
}

.tasks__check {
  width: 20px;
  height: 20px;
  flex: none;
  border: 1.5px solid var(--at-color-hairline-strong);
  border-radius: 50%;
  background: transparent;
  transition: border-color var(--at-duration-fast) var(--at-ease), background var(--at-duration-fast) var(--at-ease);
}

.tasks__check:hover {
  border-color: var(--at-color-primary);
  background: var(--at-bg-surface-strong);
}

.tasks__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
  padding: var(--at-space-2) 0;
  text-align: left;
  background: none;
  border: 0;
  color: inherit;
}

.tasks__title {
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tasks__meta {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.tasks__due {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.tasks__due.is-overdue {
  color: var(--at-color-danger);
}

.tasks__assignees {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  overflow: hidden;
}

.tasks__assignee {
  flex: none;
  padding: 0 6px;
  border-radius: var(--at-radius-pill);
  font-size: 11px;
  line-height: 16px;
  white-space: nowrap;
}

.tasks__assignee.is-self {
  background: #dc2626;
  color: #fff;
}

.tasks__assignee.is-other {
  background: #166534;
  color: #fff;
}
</style>
