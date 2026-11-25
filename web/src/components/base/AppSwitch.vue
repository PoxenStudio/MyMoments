<script setup lang="ts">
defineProps<{
  modelValue: boolean;
  label: string;
  description?: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>();
</script>

<template>
  <div class="at-switch-row">
    <div class="at-switch-row__text">
      <span class="at-switch-row__label">{{ label }}</span>
      <span v-if="description" class="at-switch-row__desc">{{ description }}</span>
    </div>
    <button
      type="button"
      class="at-switch"
      :class="{ 'is-on': modelValue }"
      role="switch"
      :aria-checked="modelValue"
      :aria-label="label"
      :disabled="disabled"
      @click="emit('update:modelValue', !modelValue)"
    >
      <span class="at-switch__thumb" />
    </button>
  </div>
</template>

<style scoped>
.at-switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-4);
  min-height: var(--at-touch-min);
}

.at-switch-row__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.at-switch-row__label {
  font-size: var(--at-text-body-sm);
}

.at-switch-row__desc {
  font-size: 13px;
  color: var(--at-color-muted);
}

.at-switch {
  flex: none;
  width: 46px;
  height: 28px;
  padding: 3px;
  border-radius: var(--at-radius-pill);
  background: var(--at-color-hairline-strong);
  transition: background var(--at-duration-base) var(--at-ease);
}

.at-switch.is-on {
  background: var(--at-color-primary);
}

.at-switch__thumb {
  display: block;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--at-text-on-primary);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  transition: transform var(--at-duration-base) var(--at-ease);
}

.at-switch.is-on .at-switch__thumb {
  transform: translateX(18px);
}

.at-switch:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
