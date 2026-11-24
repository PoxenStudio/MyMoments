<script setup lang="ts">
import AppIcon from './AppIcon.vue';
import type { IconName } from './icons';

withDefaults(defineProps<{
  title: string;
  description?: string;
  icon?: IconName;
}>(), {
  description: undefined,
  icon: 'notes',
});
</script>

<template>
  <div class="at-empty">
    <span class="at-empty__icon">
      <AppIcon :name="icon" :size="26" />
    </span>
    <h3 class="at-empty__title">{{ title }}</h3>
    <p v-if="description" class="at-empty__desc">{{ description }}</p>
    <div v-if="$slots.default" class="at-empty__action">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.at-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: var(--at-space-8) var(--at-space-4);
  gap: var(--at-space-2);
}

.at-empty__icon {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: var(--at-bg-surface-strong);
  color: var(--at-color-muted);
  margin-bottom: var(--at-space-1);
}

.at-empty__title {
  font-size: var(--at-text-body-sm);
  font-weight: 500;
}

.at-empty__desc {
  max-width: 340px;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  line-height: 1.6;
}

.at-empty__action {
  margin-top: var(--at-space-2);
}
</style>
