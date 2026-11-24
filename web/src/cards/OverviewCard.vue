<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { formatShortDate, formatTime } from '@/utils/chrono';
import AppIcon from '@/components/base/AppIcon.vue';
import { asData, type OverviewData } from '@/cards/types';

const props = defineProps<{ data: unknown }>();
const d = computed(() => asData<OverviewData>(props.data));
const counts = computed(() => d.value?.counts ?? { open: 0, completed: 0, archived: 0, today: 0, overdue: 0 });
const nextEvent = computed(() => d.value?.nextEvent ?? null);
</script>

<template>
  <div class="overview">
    <dl class="overview__stats">
      <div class="overview__stat">
        <dt>待办</dt>
        <dd>{{ counts.open }}</dd>
      </div>
      <div class="overview__stat">
        <dt>今天</dt>
        <dd>{{ counts.today }}</dd>
      </div>
      <div class="overview__stat">
        <dt>逾期</dt>
        <dd :class="{ 'is-danger': counts.overdue > 0 }">{{ counts.overdue }}</dd>
      </div>
      <div class="overview__stat">
        <dt>今日安排</dt>
        <dd>{{ d?.eventsToday ?? 0 }}</dd>
      </div>
    </dl>

    <div class="overview__rows">
      <RouterLink v-if="nextEvent" :to="{ name: 'calendar', query: { open: String(nextEvent.id), date: nextEvent.occurrenceDate ?? '' } }" class="overview__row">
        <span class="overview__badge"><AppIcon name="calendar" :size="15" /></span>
        <span class="overview__text">
          <span class="overview__label">接下来</span>
          <span class="overview__value">{{ nextEvent.title }}</span>
        </span>
        <span class="overview__meta">{{ formatShortDate(nextEvent.occurrenceDate ?? nextEvent.startLocal.slice(0, 10)) }} {{ nextEvent.allDay ? '' : formatTime(nextEvent.startLocal.slice(11)) }}</span>
      </RouterLink>

      <RouterLink v-if="d?.birthdays.length" :to="{ name: 'calendar' }" class="overview__row">
        <span class="overview__badge"><AppIcon name="gift" :size="15" /></span>
        <span class="overview__text">
          <span class="overview__label">生日临近</span>
          <span class="overview__value">{{ d.birthdays.map((b) => b.name).join('、') }}</span>
        </span>
        <span class="overview__meta">{{ d.birthdays[0]?.daysUntil === 0 ? '今天' : `${d.birthdays[0]?.daysUntil} 天后` }}</span>
      </RouterLink>

      <RouterLink v-if="d?.remindersPending" :to="{ name: 'tasks' }" class="overview__row">
        <span class="overview__badge"><AppIcon name="bell" :size="15" /></span>
        <span class="overview__text">
          <span class="overview__label">提醒</span>
          <span class="overview__value">还有 {{ d.remindersPending }} 条提醒待查看</span>
        </span>
      </RouterLink>
    </div>
  </div>
</template>

<style scoped>
.overview__stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--at-space-3);
}

.overview__stat dt {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.overview__stat dd {
  margin-top: 2px;
  font-family: var(--at-font-display);
  font-size: 30px;
  font-weight: 300;
  line-height: 1.1;
  color: var(--at-color-ink);
}

.overview__stat dd.is-danger {
  color: var(--at-color-danger);
}

.overview__rows {
  display: flex;
  flex-direction: column;
  margin-top: var(--at-space-4);
  border-top: 1px solid var(--at-color-hairline);
}

.overview__row {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  min-height: 44px;
  padding: var(--at-space-2) 0;
  border-bottom: 1px solid var(--at-color-hairline);
  color: inherit;
}

.overview__row:last-child {
  border-bottom: 0;
}

.overview__row:hover .overview__value {
  color: var(--at-color-ink);
}

.overview__badge {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  flex: none;
  border-radius: 50%;
  background: var(--at-bg-surface-strong);
  color: var(--at-color-body);
}

.overview__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.overview__label {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.overview__value {
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview__meta {
  flex: none;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

@media (max-width: 639px) {
  .overview__stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--at-space-4);
  }
}
</style>
