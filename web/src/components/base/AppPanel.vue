<script setup lang="ts">
import { computed, useSlots } from 'vue';

const props = withDefaults(defineProps<{
  title?: string;
  subtitle?: string;
  padding?: 'none' | 'sm' | 'md' | 'lg';
  tone?: 'default' | 'soft' | 'dark';
  hoverable?: boolean;
}>(), {
  title: undefined,
  subtitle: undefined,
  padding: 'md',
  tone: 'default',
  hoverable: false,
});

const slots = useSlots();
const hasHeader = computed(() => !!props.title || !!slots.header || !!slots.actions);
const hasFooter = computed(() => !!slots.footer);
</script>

<template>
  <section
    class="at-panel at-card"
    :class="[`at-panel--pad-${padding}`, `at-panel--${tone}`, hoverable ? 'at-card--hover' : '']"
  >
    <header v-if="hasHeader" class="at-panel__header">
      <slot name="header">
        <div>
          <h2 v-if="title" class="at-panel__title">{{ title }}</h2>
          <p v-if="subtitle" class="at-panel__subtitle">{{ subtitle }}</p>
        </div>
      </slot>
      <div v-if="slots.actions" class="at-panel__actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="at-panel__body">
      <slot />
    </div>
    <footer v-if="hasFooter" class="at-panel__footer">
      <slot name="footer" />
    </footer>
  </section>
</template>

<style scoped>
.at-panel {
  border-radius: var(--at-radius-card, var(--at-radius-xl));
  overflow: hidden;
}

.at-panel--soft {
  background: var(--at-bg-canvas-soft);
}

.at-panel--dark {
  background: var(--at-bg-dark);
  border-color: var(--at-bg-dark);
  color: var(--at-text-on-dark);
}

.at-panel--dark .at-panel__title,
.at-panel--dark .at-panel__subtitle {
  color: var(--at-text-on-dark);
}

.at-panel--pad-none .at-panel__header {
  padding: var(--at-space-4) var(--at-space-4) 0;
}

.at-panel--pad-none .at-panel__body {
  padding: 0;
}

.at-panel--pad-none .at-panel__footer {
  padding: 0 var(--at-space-4) var(--at-space-4);
}

.at-panel--pad-sm .at-panel__header {
  padding: var(--at-space-3) var(--at-space-3) 0;
}

.at-panel--pad-sm .at-panel__body {
  padding: var(--at-space-3);
}

.at-panel--pad-sm .at-panel__footer {
  padding: 0 var(--at-space-3) var(--at-space-3);
}

.at-panel--pad-md .at-panel__header {
  padding: var(--at-space-4) var(--at-space-5) 0;
}

.at-panel--pad-md .at-panel__body {
  padding: var(--at-space-4) var(--at-space-5);
}

.at-panel--pad-md .at-panel__footer {
  padding: 0 var(--at-space-5) var(--at-space-5);
}

.at-panel--pad-lg .at-panel__header {
  padding: var(--at-space-6) var(--at-space-6) 0;
}

.at-panel--pad-lg .at-panel__body {
  padding: var(--at-space-5) var(--at-space-6) var(--at-space-6);
}

.at-panel--pad-lg .at-panel__footer {
  padding: 0 var(--at-space-6) var(--at-space-6);
}

.at-panel__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--at-space-3);
}

.at-panel__title {
  font-size: var(--at-text-title-sm);
  font-weight: 500;
  letter-spacing: 0.18px;
}

.at-panel__subtitle {
  margin-top: 2px;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.at-panel__actions {
  display: flex;
  align-items: center;
  gap: var(--at-space-1);
  flex: none;
}

.at-panel__body {
  min-width: 0;
}

.at-panel__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--at-space-2);
}
</style>
