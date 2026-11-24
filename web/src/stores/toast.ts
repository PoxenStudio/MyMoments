import { defineStore } from 'pinia';
import { ref } from 'vue';

export type ToastTone = 'neutral' | 'success' | 'danger';

export interface ToastItem {
  id: number;
  message: string;
  tone: ToastTone;
  actionLabel?: string;
  onAction?: () => void;
  duration: number;
}

let seq = 0;

export const useToastStore = defineStore('toast', () => {
  const items = ref<ToastItem[]>([]);

  function dismiss(id: number): void {
    items.value = items.value.filter((t) => t.id !== id);
  }

  function push(message: string, tone: ToastTone = 'neutral', options: { actionLabel?: string; onAction?: () => void; duration?: number } = {}): number {
    const duplicate = items.value.find((t) => t.message === message && t.tone === tone);
    if (duplicate) {
      dismiss(duplicate.id);
    }
    const id = ++seq;
    const duration = options.duration ?? (tone === 'danger' ? 6000 : 3000);
    items.value = [...items.value, { id, message, tone, duration, actionLabel: options.actionLabel, onAction: options.onAction }];
    if (duration > 0) {
      window.setTimeout(() => dismiss(id), duration);
    }
    return id;
  }

  function success(message: string): number {
    return push(message, 'success');
  }

  function error(message: string): number {
    return push(message, 'danger');
  }

  function undo(message: string, onAction: () => void): number {
    return push(message, 'neutral', { actionLabel: '撤销', onAction, duration: 8000 });
  }

  return { items, push, dismiss, success, error, undo };
});
