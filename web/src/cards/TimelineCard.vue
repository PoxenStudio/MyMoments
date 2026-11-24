<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { asData, type TimelineData } from '@/cards/types';
import { relativeDayLabel } from '@/utils/chrono';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import type { TimelineEntry } from '@/types';

const props = defineProps<{ data: unknown }>();
const router = useRouter();
const d = computed(() => asData<TimelineData>(props.data));
const items = computed(() => d.value?.items ?? []);

function open(entry: TimelineEntry): void {
  const path = entry.detailTarget.view === 'calendar' ? '/calendar' : `/${entry.detailTarget.view}`;
  void router.push({
    path,
    query: { open: String(entry.detailTarget.open), date: entry.detailTarget.date },
  });
}
</script>

<template>
  <div v-if="!items.length">
    <AppEmptyState title="最近没有新动态" description="完成待办、留言或写便签，都会出现在这里" icon="timeline" />
  </div>
  <ul v-else class="feed">
    <li v-for="entry in items" :key="entry.occurrenceKey" class="feed__row">
      <button type="button" class="feed__link" @click="open(entry)">
        <span class="feed__date">{{ relativeDayLabel(entry.occurredOn) }}</span>
        <span class="feed__body">
          <span class="feed__title">{{ entry.title }}</span>
          <span v-if="entry.summary" class="feed__summary">{{ entry.summary }}</span>
        </span>
      </button>
    </li>
  </ul>
</template>

<style scoped>
.feed {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.feed__row {
  border-bottom: 1px solid var(--at-color-hairline);
}

.feed__row:last-child {
  border-bottom: 0;
}

.feed__link {
  display: flex;
  align-items: flex-start;
  gap: var(--at-space-3);
  width: 100%;
  min-height: 48px;
  padding: var(--at-space-2) 0;
  text-align: left;
  background: none;
  border: 0;
  color: inherit;
}

.feed__date {
  flex: none;
  width: 64px;
  padding-top: 2px;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.feed__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.feed__title {
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
}

.feed__summary {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 1;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}
</style>
