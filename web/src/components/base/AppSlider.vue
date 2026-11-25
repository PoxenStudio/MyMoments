<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(defineProps<{
  modelValue: number;
  min?: number;
  max?: number;
  step?: number;
  label: string;
  suffix?: string;
  disabled?: boolean;
}>(), {
  min: 0,
  max: 100,
  step: 1,
  suffix: '',
  disabled: false,
});

const emit = defineEmits<{
  'update:modelValue': [value: number];
  commit: [value: number];
}>();

const percent = computed(() => {
  if (props.max === props.min) return 0;
  return ((props.modelValue - props.min) / (props.max - props.min)) * 100;
});

function onInput(event: Event): void {
  emit('update:modelValue', Number((event.target as HTMLInputElement).value));
}

function onChange(event: Event): void {
  emit('commit', Number((event.target as HTMLInputElement).value));
}
</script>

<template>
  <div class="at-slider" :class="{ 'is-disabled': disabled }">
    <div class="at-slider__head">
      <span class="at-slider__label">{{ label }}</span>
      <span class="at-slider__value at-tabular">{{ modelValue }}{{ suffix }}</span>
    </div>
    <input
      class="at-slider__input"
      type="range"
      :min="min"
      :max="max"
      :step="step"
      :value="modelValue"
      :disabled="disabled"
      :aria-label="label"
      :style="{ '--at-slider-fill': `${percent}%` }"
      @input="onInput"
      @change="onChange"
    />
  </div>
</template>

<style scoped>
.at-slider {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  min-width: 0;
}

.at-slider__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
  font-size: var(--at-text-caption);
}

.at-slider__label {
  font-weight: 500;
}

.at-slider__value {
  color: var(--at-color-muted);
  font-size: var(--at-text-micro);
}

.at-slider__input {
  appearance: none;
  width: 100%;
  height: var(--at-touch-min);
  margin: 0;
  background: transparent;
  cursor: pointer;
}

.at-slider__input::-webkit-slider-runnable-track {
  height: 4px;
  border-radius: var(--at-radius-pill);
  background: linear-gradient(
    to right,
    var(--at-color-ink) 0,
    var(--at-color-ink) var(--at-slider-fill),
    var(--at-color-hairline-strong) var(--at-slider-fill),
    var(--at-color-hairline-strong) 100%
  );
}

.at-slider__input::-moz-range-track {
  height: 4px;
  border-radius: var(--at-radius-pill);
  background: var(--at-color-hairline-strong);
}

.at-slider__input::-moz-range-progress {
  height: 4px;
  border-radius: var(--at-radius-pill);
  background: var(--at-color-ink);
}

.at-slider__input::-webkit-slider-thumb {
  appearance: none;
  width: 20px;
  height: 20px;
  margin-top: -8px;
  border-radius: 50%;
  border: 2px solid var(--at-bg-surface);
  background: var(--at-color-ink);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.25);
}

.at-slider__input::-moz-range-thumb {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid var(--at-bg-surface);
  background: var(--at-color-ink);
}

.at-slider__input:disabled {
  cursor: not-allowed;
}

.at-slider.is-disabled {
  opacity: 0.5;
}
</style>
