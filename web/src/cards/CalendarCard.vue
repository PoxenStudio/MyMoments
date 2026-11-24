<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { asData, type CalendarData } from '@/cards/types';
import { relativeDayLabel, formatTime } from '@/utils/chrono';
import AppEmptyState from '@/components/base/AppEmptyState.vue';

const props = defineProps<{ data: unknown }>();
const d = computed(() => asData<CalendarData>(props.data));
const items = computed(() => d.value?.items ?? []);
</script>

<template>
  <div v-if="!items.length">
    <AppEmptyState title="近七天没有安排" description="去行事历看看，或者给自己留一点空白" icon="calendar" />
  </div>
  <ul v-else class="agenda">
    <li v-for="event in items" :key="`${event.id}-${event.occurrenceDate ?? event.startLocal}`" class="agenda__row">
      <RouterLink
        :to="{ name: 'calendar', query: { open: String(event.id), date: event.occurrenceDate ?? event.startLocal.slice(0, 10) } }"
        class="agenda__link"
      >
        <span class="agenda__date">{{ relativeDayLabel(event.occurrenceDate ?? event.startLocal.slice(0, 10)) }}</span>
        <span class="agenda__body">
          <span class="agenda__title">{{ event.title }}</span>
          <span class="agenda__meta">
            {{ event.allDay ? '全天' : formatTime(event.startLocal.slice(11)) }}
            <template v-if="event.location"> · {{ event.location }}</template>
          </span>
        </span>
        <span class="agenda__dot" :style="{ background: event.color ?? 'var(--at-color-muted-soft)' }" />
      </RouterLink>
    </li>
  </ul>
</template>

<style scoped>
.agenda {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.agenda__row {
  border-bottom: 1px solid var(--at-color-hairline);
}

.agenda__row:last-child {
  border-bottom: 0;
}

.agenda__link {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  min-height: 48px;
  padding: var(--at-space-2) 0;
  color: inherit;
}

.agenda__date {
  flex: none;
  width: 64px;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.agenda__body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.agenda__title {
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agenda__meta {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.agenda__dot {
  width: 6px;
  height: 6px;
  flex: none;
  border-radius: 50%;
}
</style>
