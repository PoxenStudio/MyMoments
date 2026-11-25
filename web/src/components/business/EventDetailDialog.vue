<script setup lang="ts">
import { computed } from 'vue';
import type { CalendarEvent } from '@/types';
import { recurrenceSummary } from '@/utils/task';
import { dayLabel, timeOf } from '@/utils/calendar';
import AppAvatar from '@/components/base/AppAvatar.vue';
import { avatarUrl } from '@/utils/avatar';
import AppButton from '@/components/base/AppButton.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppStatusTag from '@/components/base/AppStatusTag.vue';

const props = defineProps<{
  modelValue: boolean;
  event: CalendarEvent | null;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  edit: [event: CalendarEvent];
  remove: [event: CalendarEvent, scope: 'this' | 'all'];
}>();

const date = computed(() => props.event?.occurrenceDate ?? props.event?.startLocal.slice(0, 10) ?? '');
const timeText = computed(() => {
  if (!props.event || props.event.allDay) return '全天';
  const start = timeOf(props.event.startLocal);
  const end = props.event.endLocal ? timeOf(props.event.endLocal) : '';
  return end ? `${start} – ${end}` : start;
});
const mapURL = computed(() => {
  const location = props.event?.location;
  return location ? `https://uri.amap.com/search?keyword=${encodeURIComponent(location)}` : '';
});
</script>

<template>
  <AppDialog :model-value="modelValue" title="事件详情" size="md" @update:model-value="emit('update:modelValue', $event)">
    <article v-if="event" class="event-detail">
      <header class="event-detail__head">
        <span class="event-detail__swatch" :style="{ background: event.color || 'var(--at-color-primary)' }" />
        <div>
          <h3 class="event-detail__title">{{ event.title }}</h3>
          <p class="event-detail__when">{{ dayLabel(date) }} · {{ timeText }}</p>
        </div>
      </header>

      <div class="event-detail__tags">
        <AppStatusTag v-if="event.recurrenceRule" tone="neutral">{{ recurrenceSummary(event.recurrenceRule) }}</AppStatusTag>
        <AppStatusTag v-if="event.visibility === 'private'" tone="warning">仅自己可见</AppStatusTag>
        <AppStatusTag v-if="event.source === 'birthday'" tone="brand">生日</AppStatusTag>
      </div>

      <dl class="event-detail__list">
        <div v-if="event.location" class="event-detail__row">
          <dt><AppIcon name="map-pin" :size="15" />地点</dt>
          <dd>
            <a :href="mapURL" target="_blank" rel="noopener noreferrer">{{ event.location }}<AppIcon name="external" :size="13" /></a>
          </dd>
        </div>
        <div v-if="event.assignees.length" class="event-detail__row">
          <dt><AppIcon name="user" :size="15" />参与</dt>
          <dd class="event-detail__people">
            <span v-for="person in event.assignees" :key="person.id" class="event-detail__person">
              <AppAvatar :name="person.displayName" :size="22" :src="avatarUrl(person.avatarKey)" />{{ person.displayName }}
            </span>
          </dd>
        </div>
        <div v-if="event.description" class="event-detail__row">
          <dt><AppIcon name="notes" :size="15" />说明</dt>
          <dd class="event-detail__desc">{{ event.description }}</dd>
        </div>
      </dl>
    </article>
    <template #footer>
      <template v-if="event">
        <AppButton
          v-if="event.recurrenceRule"
          variant="ghost"
          @click="emit('remove', event, 'this')"
        >
          仅删除本次
        </AppButton>
        <AppButton v-else variant="ghost" @click="emit('remove', event, 'all')">删除</AppButton>
        <AppButton v-if="event.recurrenceRule" variant="ghost" @click="emit('remove', event, 'all')">删除系列</AppButton>
        <AppButton variant="outline" @click="emit('edit', event)"><AppIcon name="edit" :size="16" />编辑</AppButton>
        <AppButton @click="emit('update:modelValue', false)">完成</AppButton>
      </template>
    </template>
  </AppDialog>
</template>

<style scoped>
.event-detail {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.event-detail__head {
  display: flex;
  gap: var(--at-space-3);
  align-items: flex-start;
}

.event-detail__swatch {
  flex: none;
  width: 6px;
  align-self: stretch;
  border-radius: var(--at-radius-pill);
}

.event-detail__title {
  font-family: var(--at-font-display);
  font-weight: 400;
  font-size: var(--at-text-title-sm);
}

.event-detail__when {
  margin-top: 4px;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.event-detail__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.event-detail__list {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  border-top: 1px solid var(--at-color-hairline);
  padding-top: var(--at-space-4);
}

.event-detail__row {
  display: grid;
  grid-template-columns: 72px 1fr;
  gap: var(--at-space-3);
  font-size: var(--at-text-caption);
}

.event-detail__row dt {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--at-color-muted);
}

.event-detail__row dd a {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--at-color-brand);
}

.event-detail__people {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.event-detail__person {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.event-detail__desc {
  white-space: pre-wrap;
  line-height: 1.6;
  color: var(--at-text-secondary);
}
</style>
