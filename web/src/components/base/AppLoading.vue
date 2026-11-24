<script setup lang="ts">
withDefaults(defineProps<{
  rows?: number;
  label?: string;
}>(), {
  rows: 3,
  label: '加载中',
});
</script>

<template>
  <div class="at-loading" role="status" :aria-label="label">
    <span v-for="i in rows" :key="i" class="at-loading__line" :style="{ width: `${100 - (i - 1) * 12}%` }" />
  </div>
</template>

<style scoped>
.at-loading {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  padding: var(--at-space-2) 0;
}

.at-loading__line {
  height: 12px;
  border-radius: var(--at-radius-pill);
  background: linear-gradient(90deg, var(--at-bg-surface-strong) 25%, #f7f6f5 37%, var(--at-bg-surface-strong) 63%);
  background-size: 400% 100%;
  animation: at-shimmer 1.4s ease infinite;
}

@keyframes at-shimmer {
  0% {
    background-position: 100% 50%;
  }
  100% {
    background-position: 0 50%;
  }
}
</style>
