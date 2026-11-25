<script setup lang="ts">
withDefaults(defineProps<{
  tone?: 'neutral' | 'success' | 'danger' | 'warning' | 'brand';
}>(), {
  tone: 'neutral',
});
</script>

<template>
  <span class="at-tag" :class="`at-tag--${tone}`">
    <slot />
  </span>
</template>

<style scoped>
.at-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 10px;
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface-strong);
  color: var(--at-color-body-strong);
  font-size: var(--at-text-micro);
  font-weight: 600;
  letter-spacing: 0.4px;
  white-space: nowrap;
}

.at-tag--success {
  background: var(--at-color-success-soft);
  color: var(--at-color-success);
}

.at-tag--danger {
  background: var(--at-color-danger-soft);
  color: var(--at-color-danger);
}

.at-tag--warning {
  background: var(--at-color-warning-soft);
  color: var(--at-color-warning);
}

.at-tag--brand {
  background: var(--at-color-brand-soft);
  color: var(--at-color-brand);
}
</style>
