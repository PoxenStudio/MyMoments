<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(defineProps<{
  variant?: 'primary' | 'outline' | 'ghost' | 'danger' | 'text';
  size?: 'sm' | 'md' | 'lg';
  type?: 'button' | 'submit' | 'reset';
  loading?: boolean;
  disabled?: boolean;
  block?: boolean;
}>(), {
  variant: 'primary',
  size: 'md',
  type: 'button',
  loading: false,
  disabled: false,
  block: false,
});

const classes = computed(() => [
  'at-btn',
  `at-btn--${props.variant}`,
  `at-btn--${props.size}`,
  props.block ? 'at-btn--block' : '',
]);
</script>

<template>
  <button
    :type="type"
    :class="classes"
    :disabled="disabled || loading"
    :aria-busy="loading || undefined"
  >
    <span v-if="loading" class="at-btn__spinner" aria-hidden="true" />
    <slot />
  </button>
</template>

<style scoped>
.at-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--at-space-2);
  height: var(--at-control-height);
  padding: 0 var(--at-space-5);
  border-radius: var(--at-radius-pill);
  border: 1px solid transparent;
  font-size: 15px;
  font-weight: 500;
  line-height: 1;
  white-space: nowrap;
  transition: background var(--at-duration-fast) var(--at-ease), color var(--at-duration-fast) var(--at-ease), border-color var(--at-duration-fast) var(--at-ease);
}

.at-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.at-btn--primary {
  background: var(--at-color-primary);
  color: var(--at-text-on-primary);
}

.at-btn--primary:hover:not(:disabled) {
  background: var(--at-color-primary-active);
}

.at-btn--outline {
  background: transparent;
  color: var(--at-color-ink);
  border-color: var(--at-color-hairline-strong);
}

.at-btn--outline:hover:not(:disabled) {
  background: var(--at-bg-surface);
  border-color: var(--at-color-muted-soft);
}

.at-btn--ghost {
  background: transparent;
  color: var(--at-color-ink);
}

.at-btn--ghost:hover:not(:disabled) {
  background: var(--at-bg-surface-strong);
}

.at-btn--text {
  background: transparent;
  color: var(--at-color-body-strong);
  padding: 0 var(--at-space-1);
  height: auto;
  border-radius: var(--at-radius-xs);
}

.at-btn--text:hover:not(:disabled) {
  color: var(--at-color-ink);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.at-btn--danger {
  background: var(--at-color-danger);
  color: #fff;
}

.at-btn--danger:hover:not(:disabled) {
  background: #b91c1c;
}

.at-btn--sm {
  height: 32px;
  padding: 0 var(--at-space-3);
  font-size: 14px;
}

.at-btn--lg {
  height: 48px;
  padding: 0 var(--at-space-6);
  font-size: 16px;
}

.at-btn--block {
  width: 100%;
}

.at-btn__spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: at-spin 0.7s linear infinite;
}

@keyframes at-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 639px) {
  .at-btn {
    min-height: var(--at-touch-min);
  }

  .at-btn--sm {
    min-height: 36px;
  }
}
</style>
