<script setup lang="ts">
import { computed } from 'vue';
import { asData } from '@/cards/types';
import { relativeDayLabel } from '@/utils/chrono';
import AppButton from '@/components/base/AppButton.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import WeatherIcon from '@/components/base/WeatherIcon.vue';
import type { WeatherData } from '@/types';

const props = defineProps<{ data: unknown }>();
defineEmits<{ configure: [] }>();
const d = computed(() => asData<WeatherData>(props.data));

const updatedText = computed(() => {
  if (!d.value) return '';
  const date = new Date(d.value.updatedAt * 1000);
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
});
</script>

<template>
  <div v-if="!d">
    <AppEmptyState title="还没有天气" description="设置城市后就能看到实时天气" icon="sun">
      <AppButton size="sm" variant="outline" @click="$emit('configure')">设置位置</AppButton>
    </AppEmptyState>
  </div>
  <div v-else class="weather">
    <div class="weather__now">
      <span class="weather__icon"><WeatherIcon :code="d.code" :size="40" /></span>
      <div class="weather__temp">
        <span class="weather__value">{{ Math.round(d.temperature) }}°</span>
        <span class="weather__desc">{{ d.description }} · {{ d.city }}</span>
      </div>
      <div class="weather__range">
        <span>最高 {{ d.high !== null ? Math.round(d.high) : '—' }}°</span>
        <span>最低 {{ d.low !== null ? Math.round(d.low) : '—' }}°</span>
      </div>
    </div>

    <ul v-if="d.forecast.length > 1" class="weather__forecast">
      <li v-for="day in d.forecast.slice(1)" :key="day.date" class="weather__day">
        <span class="weather__day-label">{{ relativeDayLabel(day.date) }}</span>
        <WeatherIcon :code="day.code" :size="19" />
        <span class="weather__day-temp">{{ Math.round(day.max) }}° / {{ Math.round(day.min) }}°</span>
      </li>
    </ul>

    <div class="weather__foot">
      <span>{{ d.stale ? '数据更新于' : '更新于' }} {{ updatedText }}<template v-if="d.stale"> · 暂时离线</template></span>
      <button type="button" class="weather__settings" @click="$emit('configure')">设置位置</button>
    </div>
  </div>
</template>

<style scoped>
.weather__now {
  display: flex;
  align-items: center;
  gap: var(--at-space-4);
}

.weather__icon {
  display: grid;
  place-items: center;
  width: 56px;
  height: 56px;
  flex: none;
  border-radius: 50%;
  background: var(--at-bg-surface-strong);
  color: var(--at-color-body);
}

.weather__temp {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.weather__value {
  font-family: var(--at-font-display);
  font-size: 42px;
  font-weight: 300;
  line-height: 1;
  color: var(--at-color-ink);
}

.weather__desc {
  margin-top: 4px;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.weather__range {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: none;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
  text-align: right;
}

.weather__forecast {
  display: flex;
  flex-direction: column;
  margin: var(--at-space-4) 0 0;
  padding: 0;
  list-style: none;
  border-top: 1px solid var(--at-color-hairline);
}

.weather__day {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  min-height: 40px;
  border-bottom: 1px solid var(--at-color-hairline);
  color: var(--at-color-body);
}

.weather__day-label {
  flex: 1;
  font-size: var(--at-text-caption);
}

.weather__day-temp {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.weather__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-2);
  margin-top: var(--at-space-3);
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.weather__settings {
  background: none;
  border: 0;
  padding: 0;
  font-size: var(--at-text-micro);
  color: var(--at-color-body);
  text-decoration: underline;
  text-underline-offset: 3px;
}
</style>
