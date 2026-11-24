<script setup lang="ts">
import { computed } from 'vue';
import { asData, type RemindersData } from '@/cards/types';
import { formatTimestamp } from '@/utils/chrono';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppIcon from '@/components/base/AppIcon.vue';

const props = defineProps<{ data: unknown }>();
const d = computed(() => asData<RemindersData>(props.data));
const items = computed(() => d.value?.items ?? []);

const sourceLabel: Record<string, string> = {
  task: '待办',
  event: '行事历',
  birthday: '生日',
};
</script>

<template>
  <div v-if="!items.length">
    <AppEmptyState title="没有待处理的提醒" description="设置了提醒的待办和日程会出现在这里" icon="bell" />
  </div>
  <ul v-else class="reminders">
    <li v-for="item in items" :key="item.id" class="reminders__row">
      <span class="reminders__icon"><AppIcon name="bell" :size="15" /></span>
      <span class="reminders__body">
        <span class="reminders__title">{{ item.title ?? '提醒' }}</span>
        <span class="reminders__meta">{{ sourceLabel[item.sourceType] ?? item.sourceType }} · {{ formatTimestamp(item.remindAt) }}</span>
      </span>
    </li>
  </ul>
</template>

<style scoped>
.reminders {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.reminders__row {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  min-height: 48px;
  border-bottom: 1px solid var(--at-color-hairline);
}

.reminders__row:last-child {
  border-bottom: 0;
}

.reminders__icon {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  flex: none;
  border-radius: 50%;
  background: var(--at-bg-surface-strong);
  color: var(--at-color-body);
}

.reminders__body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.reminders__title {
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.reminders__meta {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}
</style>
