<script setup lang="ts">
import { useToastStore } from '@/stores/toast';

const toasts = useToastStore();
</script>

<template>
  <Teleport to="body">
    <div class="at-toasts" aria-live="polite" aria-atomic="false">
      <TransitionGroup name="at-toast">
        <div
          v-for="toast in toasts.items"
          :key="toast.id"
          class="at-toast"
          :class="`at-toast--${toast.tone}`"
          :role="toast.tone === 'danger' ? 'alert' : 'status'"
        >
          <span class="at-toast__message">{{ toast.message }}</span>
          <button
            v-if="toast.actionLabel"
            type="button"
            class="at-toast__action"
            @click="toast.onAction?.(); toasts.dismiss(toast.id)"
          >
            {{ toast.actionLabel }}
          </button>
          <button type="button" class="at-toast__close" aria-label="关闭提示" @click="toasts.dismiss(toast.id)">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.at-toasts {
  position: fixed;
  left: 50%;
  bottom: calc(var(--at-space-6) + env(safe-area-inset-bottom));
  transform: translateX(-50%);
  z-index: var(--at-z-toast);
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  width: min(92vw, 480px);
  pointer-events: none;
}

.at-toast {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  padding: 10px var(--at-space-3) 10px var(--at-space-4);
  background: var(--at-bg-dark);
  color: var(--at-text-on-dark);
  border-radius: var(--at-radius-pill);
  box-shadow: var(--at-shadow-dialog);
  pointer-events: auto;
  font-size: var(--at-text-caption);
}

.at-toast--success .at-toast__message::before {
  content: '';
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--at-color-success);
  margin-right: 8px;
  vertical-align: middle;
}

.at-toast--danger .at-toast__message::before {
  content: '';
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--at-color-danger);
  margin-right: 8px;
  vertical-align: middle;
}

.at-toast__message {
  flex: 1;
  line-height: 1.4;
}

.at-toast__action {
  flex: none;
  color: #fff;
  font-weight: 600;
  font-size: 13px;
  text-decoration: underline;
  text-underline-offset: 3px;
  padding: 6px;
}

.at-toast__close {
  flex: none;
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  color: var(--at-text-on-dark-soft);
}

.at-toast__close:hover {
  background: rgba(255, 255, 255, 0.12);
}

.at-toast-enter-active,
.at-toast-leave-active {
  transition: opacity var(--at-duration-base) var(--at-ease), transform var(--at-duration-base) var(--at-ease);
}

.at-toast-enter-from,
.at-toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

@media (max-width: 639px) {
  .at-toasts {
    bottom: calc(76px + env(safe-area-inset-bottom));
  }
}
</style>
