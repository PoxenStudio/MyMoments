<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue';
import AppIconButton from './AppIconButton.vue';

const props = withDefaults(defineProps<{
  modelValue: boolean;
  title: string;
  size?: 'sm' | 'md' | 'lg';
  closeOnBackdrop?: boolean;
  hideClose?: boolean;
}>(), {
  size: 'md',
  closeOnBackdrop: true,
  hideClose: false,
});

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>();

const panel = ref<HTMLElement | null>(null);
let previousFocus: HTMLElement | null = null;

function close(): void {
  emit('update:modelValue', false);
}

function onBackdrop(): void {
  if (props.closeOnBackdrop) close();
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    event.stopPropagation();
    close();
    return;
  }
  if (event.key !== 'Tab' || !panel.value) return;
  const focusables = panel.value.querySelectorAll<HTMLElement>(
    'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])',
  );
  if (focusables.length === 0) return;
  const first = focusables[0];
  const last = focusables[focusables.length - 1];
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
  }
}

watch(() => props.modelValue, async (open) => {
  if (open) {
    previousFocus = document.activeElement as HTMLElement | null;
    document.body.style.overflow = 'hidden';
    await nextTick();
    const target = panel.value?.querySelector<HTMLElement>('[autofocus], input, textarea, select, button');
    target?.focus();
  } else {
    document.body.style.overflow = '';
    previousFocus?.focus?.();
    previousFocus = null;
  }
});

onBeforeUnmount(() => {
  document.body.style.overflow = '';
});
</script>

<template>
  <Teleport to="body">
    <Transition name="at-dialog">
      <div v-if="modelValue" class="at-dialog" @keydown="onKeydown">
        <div class="at-dialog__backdrop" @click="onBackdrop" />
        <div
          ref="panel"
          class="at-dialog__panel"
          :class="`at-dialog__panel--${size}`"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
        >
          <header class="at-dialog__header">
            <h2 class="at-dialog__title">{{ title }}</h2>
            <AppIconButton v-if="!hideClose" icon="close" label="关闭" size="sm" @click="close" />
          </header>
          <div class="at-dialog__body">
            <slot />
          </div>
          <footer v-if="$slots.footer" class="at-dialog__footer">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.at-dialog {
  position: fixed;
  inset: 0;
  z-index: var(--at-z-dialog);
  display: grid;
  place-items: center;
  padding: var(--at-space-6);
}

.at-dialog__backdrop {
  position: absolute;
  inset: 0;
  background: var(--at-color-scrim);
  backdrop-filter: blur(2px);
}

.at-dialog__panel {
  position: relative;
  width: min(100%, 560px);
  max-height: min(86vh, 900px);
  display: flex;
  flex-direction: column;
  background: var(--at-bg-surface);
  border-radius: var(--at-radius-xl);
  box-shadow: var(--at-shadow-dialog);
  overflow: hidden;
}

.at-dialog__panel--sm {
  max-width: 420px;
}

.at-dialog__panel--lg {
  max-width: 760px;
}

.at-dialog__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
  padding: var(--at-space-5) var(--at-space-5) var(--at-space-2);
}

.at-dialog__title {
  font-family: var(--at-font-display);
  font-weight: 300;
  font-size: var(--at-text-display-sm);
  line-height: 1.2;
}

.at-dialog__body {
  padding: var(--at-space-3) var(--at-space-5) var(--at-space-5);
  overflow-y: auto;
}

.at-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--at-space-2);
  padding: var(--at-space-4) var(--at-space-5);
  border-top: 1px solid var(--at-color-hairline);
  background: var(--at-bg-canvas-soft);
}

.at-dialog-enter-active,
.at-dialog-leave-active {
  transition: opacity var(--at-duration-base) var(--at-ease);
}

.at-dialog-enter-active .at-dialog__panel,
.at-dialog-leave-active .at-dialog__panel {
  transition: transform var(--at-duration-base) var(--at-ease), opacity var(--at-duration-base) var(--at-ease);
}

.at-dialog-enter-from,
.at-dialog-leave-to {
  opacity: 0;
}

.at-dialog-enter-from .at-dialog__panel,
.at-dialog-leave-to .at-dialog__panel {
  transform: translateY(12px) scale(0.99);
  opacity: 0;
}

@media (max-width: 639px) {
  .at-dialog {
    padding: 0;
    align-items: end;
  }

  .at-dialog__panel {
    width: 100%;
    max-width: none;
    max-height: 92vh;
    border-radius: var(--at-radius-xl) var(--at-radius-xl) 0 0;
    padding-bottom: env(safe-area-inset-bottom);
  }

  .at-dialog-enter-from .at-dialog__panel,
  .at-dialog-leave-to .at-dialog__panel {
    transform: translateY(40px);
  }
}
</style>
