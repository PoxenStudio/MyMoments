<script setup lang="ts">
export interface TabItem {
  key: string;
  label: string;
  count?: number;
}

defineProps<{
  modelValue: string;
  tabs: TabItem[];
  ariaLabel?: string;
}>();

const emit = defineEmits<{ 'update:modelValue': [value: string] }>();

function onKeydown(event: KeyboardEvent, index: number, tabs: TabItem[]): void {
  if (event.key !== 'ArrowRight' && event.key !== 'ArrowLeft') return;
  event.preventDefault();
  const next = event.key === 'ArrowRight'
    ? (index + 1) % tabs.length
    : (index - 1 + tabs.length) % tabs.length;
  emit('update:modelValue', tabs[next].key);
  const el = (event.currentTarget as HTMLElement).parentElement?.children[next] as HTMLElement | undefined;
  el?.focus();
}
</script>

<template>
  <div class="at-tabs" role="tablist" :aria-label="ariaLabel">
    <button
      v-for="(tab, index) in tabs"
      :key="tab.key"
      type="button"
      role="tab"
      class="at-tabs__tab"
      :class="{ 'is-active': tab.key === modelValue }"
      :aria-selected="tab.key === modelValue"
      :tabindex="tab.key === modelValue ? 0 : -1"
      @click="emit('update:modelValue', tab.key)"
      @keydown="onKeydown($event, index, tabs)"
    >
      {{ tab.label }}
      <span v-if="tab.count !== undefined" class="at-tabs__count">{{ tab.count }}</span>
    </button>
  </div>
</template>

<style scoped>
.at-tabs {
  display: flex;
  gap: var(--at-space-1);
  overflow-x: auto;
  padding: 3px;
  background: var(--at-bg-surface-strong);
  border-radius: var(--at-radius-pill);
  scrollbar-width: none;
}

.at-tabs::-webkit-scrollbar {
  display: none;
}

.at-tabs__tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 var(--at-space-4);
  border-radius: var(--at-radius-pill);
  font-size: var(--at-text-caption);
  font-weight: 500;
  color: var(--at-color-muted);
  white-space: nowrap;
  transition: background var(--at-duration-fast) var(--at-ease), color var(--at-duration-fast) var(--at-ease);
}

.at-tabs__tab:hover {
  color: var(--at-color-ink);
}

.at-tabs__tab.is-active {
  background: var(--at-bg-surface);
  color: var(--at-color-ink);
  box-shadow: var(--at-shadow-panel);
}

.at-tabs__count {
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  color: var(--at-color-muted);
}
</style>
