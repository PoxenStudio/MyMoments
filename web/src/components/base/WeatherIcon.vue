<script setup lang="ts">
import { computed, useId } from 'vue';

const props = withDefaults(defineProps<{
  code?: number | null;
  size?: number;
}>(), {
  code: null,
  size: 24,
});

type Kind =
  | 'sun'
  | 'partly'
  | 'cloud'
  | 'overcast'
  | 'fog'
  | 'drizzle'
  | 'rain'
  | 'snow'
  | 'showers'
  | 'snow-showers'
  | 'storm';

const kind = computed<Kind>(() => {
  switch (props.code) {
    case 0:
      return 'sun';
    case 1:
      return 'partly';
    case 2:
      return 'cloud';
    case 3:
      return 'overcast';
    case 45:
    case 48:
      return 'fog';
    case 51:
    case 53:
    case 55:
    case 56:
    case 57:
      return 'drizzle';
    case 61:
    case 63:
    case 65:
    case 66:
    case 67:
      return 'rain';
    case 71:
    case 73:
    case 75:
    case 77:
      return 'snow';
    case 80:
    case 81:
    case 82:
      return 'showers';
    case 85:
    case 86:
      return 'snow-showers';
    case 95:
    case 96:
    case 99:
      return 'storm';
    default:
      return 'cloud';
  }
});

// 每个实例使用独立的 defs id，避免同页多个天气图标互相覆盖。
const uid = useId();
const sunId = `${uid}-sun`;
const cloudId = `${uid}-cloud`;
</script>

<template>
  <svg
    class="at-weather-icon"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    aria-hidden="true"
    focusable="false"
  >
    <defs>
      <g :id="sunId">
        <circle cx="12" cy="12" r="4.2" stroke="none" />
        <g stroke-linecap="round">
          <line x1="12" y1="2.4" x2="12" y2="4.5" />
          <line x1="12" y1="19.5" x2="12" y2="21.6" />
          <line x1="2.4" y1="12" x2="4.5" y2="12" />
          <line x1="19.5" y1="12" x2="21.6" y2="12" />
          <line x1="5.3" y1="5.3" x2="6.9" y2="6.9" />
          <line x1="17.1" y1="17.1" x2="18.7" y2="18.7" />
          <line x1="5.3" y1="18.7" x2="6.9" y2="17.1" />
          <line x1="17.1" y1="5.3" x2="18.7" y2="6.9" />
        </g>
      </g>
      <g :id="cloudId">
        <circle cx="8.8" cy="13.4" r="3.4" />
        <circle cx="13.4" cy="11.6" r="4.6" />
        <circle cx="16.6" cy="14.4" r="2.4" />
        <rect x="8.8" y="13.6" width="7.8" height="3.2" rx="1.6" />
      </g>
    </defs>

    <template v-if="kind === 'sun'">
      <use :href="`#${sunId}`" fill="#f5a524" stroke="#f5a524" stroke-width="1.8" />
    </template>

    <template v-else-if="kind === 'partly' || kind === 'showers' || kind === 'snow-showers'">
      <use
        :href="`#${sunId}`"
        :transform="kind === 'partly' ? 'translate(1.16 0.16) scale(0.62)' : 'translate(1.2 -0.8) scale(0.6)'"
        fill="#f5a524"
        stroke="#f5a524"
        stroke-width="1.9"
      />
      <use
        :href="`#${cloudId}`"
        :transform="kind === 'partly' ? 'translate(3.4 3.6) scale(0.78)' : 'translate(3.6 1.2) scale(0.74)'"
        fill="#dfe9f3"
      />
      <g v-if="kind === 'showers'" stroke="#4c93e8" stroke-width="1.7" stroke-linecap="round">
        <line x1="9.8" y1="14.9" x2="9.2" y2="17.9" />
        <line x1="12.8" y1="14.9" x2="12.2" y2="18.1" />
        <line x1="15.5" y1="14.9" x2="14.9" y2="17.9" />
      </g>
      <g v-else fill="#a9c9f2">
        <circle cx="9.6" cy="16" r="0.95" />
        <circle cx="12.4" cy="17.4" r="0.95" />
        <circle cx="15.2" cy="16" r="0.95" />
      </g>
    </template>

    <template v-else-if="kind === 'overcast'">
      <use
        :href="`#${cloudId}`"
        transform="translate(12 12) scale(0.7) translate(-12 -12) translate(-1.6 -1.8)"
        fill="#c3cedb"
        opacity="0.75"
      />
      <use :href="`#${cloudId}`" fill="#9fadc0" />
    </template>

    <template v-else-if="kind === 'fog'">
      <use :href="`#${cloudId}`" transform="translate(0 -1.4)" fill="#cfdbe7" />
      <g stroke="#a6b3c1" stroke-width="1.7" stroke-linecap="round">
        <line x1="5.2" y1="17.4" x2="18.8" y2="17.4" />
        <line x1="7.2" y1="20.2" x2="16.8" y2="20.2" />
      </g>
    </template>

    <template v-else-if="kind === 'drizzle' || kind === 'rain' || kind === 'snow' || kind === 'storm'">
      <use :href="`#${cloudId}`" transform="translate(0 -2.2)" fill="#dce7f1" />
      <template v-if="kind === 'drizzle'">
        <g stroke="#6aa7ee" stroke-width="1.7" stroke-linecap="round">
          <line x1="10.4" y1="16" x2="9.9" y2="18.2" />
          <line x1="14" y1="16" x2="13.5" y2="18.2" />
        </g>
      </template>
      <template v-else-if="kind === 'rain'">
        <g stroke="#4c93e8" stroke-width="1.8" stroke-linecap="round">
          <line x1="9.4" y1="16" x2="8.6" y2="19" />
          <line x1="12.4" y1="16" x2="11.6" y2="19" />
          <line x1="15.4" y1="16" x2="14.6" y2="19" />
        </g>
      </template>
      <template v-else-if="kind === 'snow'">
        <g fill="#a9c9f2">
          <circle cx="9.4" cy="17.4" r="1" />
          <circle cx="12.2" cy="18.8" r="1" />
          <circle cx="15.4" cy="17.4" r="1" />
        </g>
      </template>
      <template v-else>
        <path d="M13.2 14.4 9.9 19.4h2.5l-.8 3.2 4.4-5.8h-2.6l1.3-2.4Z" fill="#f2ae2e" />
      </template>
    </template>

    <template v-else>
      <use :href="`#${cloudId}`" fill="#d8e4ef" />
      <circle cx="11.8" cy="9.4" r="2.2" fill="#ffffff" opacity="0.45" />
    </template>
  </svg>
</template>

<style scoped>
.at-weather-icon {
  flex: none;
  display: block;
}
</style>
