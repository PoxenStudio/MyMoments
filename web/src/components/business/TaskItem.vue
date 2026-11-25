<script setup lang="ts">
import { computed } from 'vue';
import type { Task } from '@/types';
import { dueLabel, priorityLabels, recurrenceSummary, visibilityLabels } from '@/utils/task';
import AppAvatar from '@/components/base/AppAvatar.vue';
import { avatarUrl } from '@/utils/avatar';
import AppIcon from '@/components/base/AppIcon.vue';
import AppStatusTag from '@/components/base/AppStatusTag.vue';

const props = withDefaults(defineProps<{
  task: Task;
  today: string;
  compact?: boolean;
}>(), {
  compact: false,
});

const emit = defineEmits<{
  complete: [task: Task];
  undo: [task: Task];
  open: [task: Task];
}>();

const completed = computed(() => props.task.status === 'completed');
const due = computed(() => dueLabel(props.task, props.today));
const doneSubtasks = computed(() => props.task.subtasks.filter((s) => s.done).length);
const repeat = computed(() => recurrenceSummary(props.task.recurrenceRule));
const highPriority = computed(() => props.task.priority === 'high' || props.task.priority === 'urgent');

function toggle(): void {
  if (completed.value) emit('undo', props.task);
  else emit('complete', props.task);
}
</script>

<template>
  <div class="task" :class="{ 'is-done': completed, 'task--compact': compact }">
    <button
      type="button"
      class="task__check"
      :class="{ 'is-checked': completed }"
      :aria-label="completed ? `撤销完成 ${task.title}` : `完成 ${task.title}`"
      @click.stop="toggle"
    >
      <AppIcon v-if="completed" name="check" :size="14" />
    </button>

    <button type="button" class="task__body" @click="emit('open', task)">
      <span class="task__title-row">
        <span v-if="task.color" class="task__dot" :style="{ background: task.color }" />
        <span class="task__title">{{ task.title }}</span>
        <AppStatusTag v-if="due.tone === 'overdue'" tone="danger">逾期</AppStatusTag>
        <AppStatusTag v-else-if="due.tone === 'today'" tone="warning">今天</AppStatusTag>
        <AppStatusTag v-if="highPriority" tone="danger">{{ priorityLabels[task.priority] }}</AppStatusTag>
      </span>
      <span class="task__meta">
        <span class="task__meta-item" :class="`is-${due.tone}`">
          <AppIcon name="clock" :size="13" />{{ due.text }}
        </span>
        <span v-if="task.subtasks.length" class="task__meta-item">
          <AppIcon name="check" :size="13" />{{ doneSubtasks }}/{{ task.subtasks.length }}
        </span>
        <span v-if="task.mediaCount" class="task__meta-item">
          <AppIcon name="image" :size="13" />{{ task.mediaCount }}
        </span>
        <span v-if="repeat" class="task__meta-item">
          <AppIcon name="refresh" :size="13" />{{ repeat }}
        </span>
        <span v-if="task.visibility !== 'household'" class="task__meta-item">
          <AppIcon name="lock" :size="13" />{{ visibilityLabels[task.visibility] }}
        </span>
      </span>
    </button>

    <div class="task__people">
      <AppAvatar v-for="person in task.assignees.slice(0, 3)" :key="person.id" :name="person.displayName" :size="24" :src="avatarUrl(person.avatarKey)" />
      <span v-if="task.assignees.length > 3" class="task__more">+{{ task.assignees.length - 3 }}</span>
    </div>
  </div>
</template>

<style scoped>
.task {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  padding: 10px var(--at-space-2);
  border-bottom: 1px solid var(--at-color-hairline);
  transition: background var(--at-duration-fast) var(--at-ease);
}

.task:last-child {
  border-bottom: 0;
}

.task:hover {
  background: var(--at-bg-canvas-soft);
}

.task--compact {
  padding: 8px 0;
}

.task__check {
  flex: none;
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
  border: 1.5px solid var(--at-color-hairline-strong);
  border-radius: 50%;
  color: transparent;
  transition: border-color var(--at-duration-fast) var(--at-ease), background var(--at-duration-fast) var(--at-ease);
}

.task__check:hover {
  border-color: var(--at-color-primary);
}

.task__check.is-checked {
  background: var(--at-color-primary);
  border-color: var(--at-color-primary);
  color: var(--at-text-on-primary);
}

.task__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
  text-align: left;
  padding: 2px 0;
}

.task__title-row {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  min-width: 0;
}

.task__dot {
  flex: none;
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.task__title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--at-text-body-sm);
  color: var(--at-color-ink);
}

.task.is-done .task__title {
  color: var(--at-color-muted-soft);
  text-decoration: line-through;
}

.task__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--at-space-3);
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.task__meta-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  white-space: nowrap;
}

.task__meta-item.is-overdue {
  color: var(--at-color-danger);
}

.task__meta-item.is-today {
  color: var(--at-color-brand);
}

.task__people {
  display: flex;
  align-items: center;
  margin-left: auto;
  padding-left: var(--at-space-2);
}

.task__people > * + * {
  margin-left: -6px;
}

.task__more {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  margin-left: -6px;
  border-radius: 50%;
  background: var(--at-bg-surface-strong);
  border: 2px solid var(--at-bg-surface);
  font-size: 10px;
  color: var(--at-color-muted);
}

@media (max-width: 639px) {
  .task__people {
    display: none;
  }
}
</style>
