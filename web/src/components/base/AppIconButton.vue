<script setup lang="ts">
import { computed } from 'vue';
import AppIcon from './AppIcon.vue';
import type { IconName } from './icons';

const props = withDefaults(defineProps<{
  icon: IconName;
  label: string;
  variant?: 'ghost' | 'outline' | 'primary';
  size?: 'sm' | 'md';
  active?: boolean;
  disabled?: boolean;
}>(), {
  variant: 'ghost',
  size: 'md',
  active: false,
  disabled: false,
});

defineEmits<{ click: [event: MouseEvent] }>();

const classes = computed(() => [
  'at-icon-btn',
  `at-icon-btn--${props.variant}`,
  `at-icon-btn--${props.size}`,
  props.active ? 'is-active' : '',
]);
</script>

<template>
  <button
    type="button"
    :class="classes"
    :aria-label="label"
    :title="label"
    :aria-pressed="active || undefined"
    :disabled="disabled"
    @click="$emit('click', $event)"
  >
    <AppIcon :name="icon" :size="size === 'sm' ? 16 : 20" />
    <slot />
  </button>
</template>

<style scoped>
.at-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: var(--at-touch-min);
  height: var(--at-touch-min);
  border-radius: var(--at-radius-pill);
  color: var(--at-color-body-strong);
  transition: background var(--at-duration-fast) var(--at-ease), color var(--at-duration-fast) var(--at-ease);
}

.at-icon-btn:hover:not(:disabled) {
  background: var(--at-bg-surface-strong);
  color: var(--at-color-ink);
}

.at-icon-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.at-icon-btn--outline {
  border: 1px solid var(--at-color-hairline-strong);
  background: var(--at-bg-surface);
}

.at-icon-btn--primary {
  background: var(--at-color-primary);
  color: var(--at-text-on-primary);
}

.at-icon-btn--primary:hover:not(:disabled) {
  background: var(--at-color-primary-active);
  color: var(--at-text-on-primary);
}

.at-icon-btn.is-active {
  background: var(--at-bg-surface-strong);
  color: var(--at-color-ink);
}

.at-icon-btn--sm {
  width: 32px;
  height: 32px;
}

@media (pointer: coarse), (max-width: 767px) {
  .at-icon-btn--sm {
    width: var(--at-touch-min);
    height: var(--at-touch-min);
  }
}
</style>
