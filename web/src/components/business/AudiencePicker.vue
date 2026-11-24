<script setup lang="ts">
import type { User } from '@/types';

const props = withDefaults(defineProps<{
  modelValue: number[];
  members: User[];
  disabled?: boolean;
}>(), {
  disabled: false,
});

const emit = defineEmits<{ 'update:modelValue': [value: number[]] }>();

function toggle(id: number): void {
  if (props.disabled) return;
  emit('update:modelValue', props.modelValue.includes(id)
    ? props.modelValue.filter((v) => v !== id)
    : [...props.modelValue, id]);
}
</script>

<template>
  <div class="audience" role="group" aria-label="选择可见成员">
    <button
      v-for="member in members"
      :key="member.id"
      type="button"
      class="audience__chip"
      :class="{ 'is-on': modelValue.includes(member.id) }"
      :aria-pressed="modelValue.includes(member.id)"
      :disabled="disabled"
      @click="toggle(member.id)"
    >
      {{ member.displayName }}
    </button>
    <span v-if="members.length === 0" class="audience__empty">暂无可选成员</span>
  </div>
</template>

<style scoped>
.audience {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.audience__chip {
  padding: 5px 12px;
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface);
  color: var(--at-color-body);
  font-size: var(--at-text-micro);
  transition: background var(--at-duration-fast) var(--at-ease), color var(--at-duration-fast) var(--at-ease);
}

.audience__chip:hover {
  background: var(--at-bg-surface-strong);
}

.audience__chip.is-on {
  background: var(--at-color-ink);
  border-color: var(--at-color-ink);
  color: var(--at-text-on-primary);
}

.audience__chip:disabled {
  opacity: 0.55;
}

.audience__empty {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}
</style>
