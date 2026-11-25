<script setup lang="ts">
import { computed } from 'vue';
import WeatherIcon from '@/components/base/WeatherIcon.vue';
import type { WeatherData } from '@/types';

const props = defineProps<{ data: WeatherData }>();
defineEmits<{ configure: [] }>();

const lowText = computed(() => (props.data.low === null ? '—' : `${Math.round(props.data.low)}°`));
const highText = computed(() => (props.data.high === null ? '—' : `${Math.round(props.data.high)}°`));
</script>

<template>
  <button
    type="button"
    class="weather-mini"
    :title="`${data.description} · ${data.city}，点击设置天气`"
    @click="$emit('configure')"
  >
    <WeatherIcon :code="data.code" :size="24" />
    <span class="weather-mini__temps">
      <span class="weather-mini__low">{{ lowText }}</span>
      <span class="weather-mini__sep">/</span>
      <span class="weather-mini__high">{{ highText }}</span>
    </span>
  </button>
</template>

<style scoped>
.weather-mini {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: none;
  padding: 1px 12px;
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-lg);
  background: var(--at-bg-surface-strong);
  color: var(--at-color-body);
  cursor: pointer;
  transition: background var(--at-duration-fast) var(--at-ease);
}

.weather-mini:hover {
  background: var(--at-bg-canvas-soft);
}

.weather-mini__temps {
  display: flex;
  flex-direction: row;
  align-items: baseline;
  gap: 2px;
  font-size: var(--at-text-caption);
  line-height: 1.2;
  color: var(--at-color-muted);
}

.weather-mini__high {
  color: var(--at-color-body);
}
</style>
