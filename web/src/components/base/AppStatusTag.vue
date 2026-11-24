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
  color: #15803d;
}

.at-tag--danger {
  background: var(--at-color-danger-soft);
  color: #b91c1c;
}

.at-tag--warning {
  background: #fffbeb;
  color: #b45309;
}

.at-tag--brand {
  background: var(--at-color-brand-soft);
  color: var(--at-color-brand);
}
</style>
