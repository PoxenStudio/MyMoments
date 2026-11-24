<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { asData, type BirthdaysData } from '@/cards/types';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';

const props = defineProps<{ data: unknown }>();
const d = computed(() => asData<BirthdaysData>(props.data));
const items = computed(() => d.value?.items ?? []);

function label(days: number | undefined): string {
  if (days === undefined) return '';
  if (days === 0) return '今天';
  if (days === 1) return '明天';
  return `${days} 天后`;
}
</script>

<template>
  <div v-if="!items.length">
    <AppEmptyState title="还没有生日记录" description="在行事历中记下家人的生日" icon="gift" />
  </div>
  <ul v-else class="birthdays">
    <li v-for="item in items" :key="item.id" class="birthdays__row">
      <RouterLink :to="{ name: 'calendar' }" class="birthdays__link">
        <AppAvatar :name="item.name" :size="34" />
        <span class="birthdays__body">
          <span class="birthdays__name">{{ item.name }}</span>
          <span class="birthdays__date">{{ item.month }}月{{ item.day }}日
            <template v-if="item.age !== null && item.age !== undefined"> · {{ item.age }} 岁</template>
          </span>
        </span>
        <span class="birthdays__days" :class="{ 'is-today': item.daysUntil === 0 }">{{ label(item.daysUntil) }}</span>
      </RouterLink>
    </li>
  </ul>
</template>

<style scoped>
.birthdays {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.birthdays__row {
  border-bottom: 1px solid var(--at-color-hairline);
}

.birthdays__row:last-child {
  border-bottom: 0;
}

.birthdays__link {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  min-height: 48px;
  padding: var(--at-space-2) 0;
  color: inherit;
}

.birthdays__body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.birthdays__name {
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
}

.birthdays__date {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.birthdays__days {
  flex: none;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.birthdays__days.is-today {
  color: var(--at-color-primary);
  font-weight: 500;
}
</style>
