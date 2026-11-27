<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useNotificationStore } from '@/stores/family';
import { formatTimestamp } from '@/utils/chrono';
import AppDialog from './base/AppDialog.vue';
import AppButton from './base/AppButton.vue';
import AppEmptyState from './base/AppEmptyState.vue';
import AppLoading from './base/AppLoading.vue';
import type { NotificationItem } from '@/types';

defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>();

const router = useRouter();
const store = useNotificationStore();

const sorted = computed(() =>
  [...store.items].sort((a, b) => {
    if (!!a.readAt !== !!b.readAt) return a.readAt ? 1 : -1;
    return b.createdAt - a.createdAt;
  }),
);

function target(item: NotificationItem): { name: string; query: Record<string, string> } | null {
  switch (item.sourceType) {
    case 'task':
      return { name: 'tasks', query: { open: String(item.sourceId) } };
    case 'event':
      return { name: 'calendar', query: { open: String(item.sourceId), date: item.occurrenceDate ?? '' } };
    case 'birthday':
      return { name: 'family', query: { tab: 'birthdays', open: String(item.sourceId) } };
    case 'message':
    case 'mention':
      return { name: 'messages', query: { open: String(item.sourceId) } };
    case 'note':
      return { name: 'notes', query: { open: String(item.sourceId) } };
    default:
      return null;
  }
}

async function open(item: NotificationItem): Promise<void> {
  if (!item.readAt) await store.markRead(item.id);
  const dest = target(item);
  emit('update:modelValue', false);
  if (dest) router.push(dest);
}

async function markAll(): Promise<void> {
  await store.markAllRead();
}
</script>

<template>
  <AppDialog
    :model-value="modelValue"
    title="消息通知"
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="notif">
      <div v-if="store.items.length" class="notif__toolbar">
        <span class="at-caption-upper">{{ store.unread }} 条未读</span>
        <AppButton v-if="store.unread" variant="text" size="sm" @click="markAll">全部标为已读</AppButton>
      </div>
      <AppLoading v-if="!store.loaded" :rows="3" />
      <AppEmptyState
        v-else-if="!store.items.length"
        icon="bell"
        title="暂无通知"
        description="提醒、@提及和新留言会出现在这里。"
      />
      <ul v-else class="notif__list">
        <li v-for="item in sorted" :key="item.id">
          <button type="button" class="notif__item" :class="{ 'is-unread': !item.readAt }" @click="open(item)">
            <span v-if="!item.readAt" class="notif__dot" aria-label="未读" />
            <span class="notif__text">
              <span class="notif__summary">{{ item.summary || '你有一条新通知' }}</span>
              <span class="notif__time">{{ formatTimestamp(item.createdAt) }}</span>
            </span>
          </button>
        </li>
      </ul>
    </div>
  </AppDialog>
</template>

<style scoped>
.notif__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: var(--at-space-2);
  border-bottom: 1px solid var(--at-color-hairline);
  margin-bottom: var(--at-space-2);
}

.notif__list {
  display: flex;
  flex-direction: column;
  max-height: 58vh;
  overflow-y: auto;
}

.notif__item {
  display: flex;
  align-items: flex-start;
  gap: var(--at-space-3);
  width: 100%;
  padding: var(--at-space-3);
  border-radius: var(--at-radius-md);
  text-align: left;
}

.notif__item:hover {
  background: var(--at-bg-canvas-soft);
}

.notif__dot {
  flex: none;
  width: 7px;
  height: 7px;
  margin-top: 7px;
  border-radius: 50%;
  background: var(--at-color-brand);
}

.notif__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.notif__summary {
  font-size: var(--at-text-body-sm);
  color: var(--at-text-primary);
}

.notif__item.is-unread .notif__summary {
  font-weight: 500;
}

.notif__time {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}
</style>
