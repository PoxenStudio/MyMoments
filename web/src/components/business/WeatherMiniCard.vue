<script setup lang="ts">
import { computed } from 'vue';
import AppIcon from '@/components/base/AppIcon.vue';
import type { WeatherData } from '@/types';
import type { IconName } from '@/components/base/icons';

const props = defineProps<{ data: WeatherData }>();
defineEmits<{ configure: [] }>();

function iconFor(code: number | undefined): IconName {
  if (code === undefined) return 'cloud';
  if (code === 0 || code === 1) return 'sun';
  if (code === 2 || code === 3) return 'cloud';
  if (code === 45 || code === 48) return 'cloud-fog';
  if ((code >= 51 && code <= 67) || (code >= 80 && code <= 82)) return 'cloud-rain';
  if ((code >= 71 && code <= 77) || code === 85 || code === 86) return 'cloud-snow';
  if (code >= 95) return 'cloud-storm';
  return 'cloud';
}

const icon = computed(() => iconFor(props.data.code));
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
    <AppIcon :name="icon" :size="22" :stroke-width="1.4" />
    <span class="weather-mini__temps">
      <span class="weather-mini__low">最低 {{ lowText }}</span>
      <span class="weather-mini__high">最高 {{ highText }}</span>
    </span>
  </button>
</template>

<style scoped>
.weather-mini {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  flex: none;
  padding: 6px 10px;
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
  flex-direction: column;
  align-items: center;
  gap: 1px;
  font-size: var(--at-text-micro);
  line-height: 1.2;
  color: var(--at-color-muted);
}

.weather-mini__high {
  color: var(--at-color-body);
}
</style>
