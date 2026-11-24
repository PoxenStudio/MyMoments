<script setup lang="ts">
defineProps<{
  modelValue: boolean;
  label?: string;
  description?: string;
  disabled?: boolean;
  id?: string;
}>();

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>();
</script>

<template>
  <label class="at-checkbox" :class="{ 'is-disabled': disabled }">
    <input
      :id="id"
      type="checkbox"
      class="at-checkbox__input"
      :checked="modelValue"
      :disabled="disabled"
      @change="emit('update:modelValue', ($event.target as HTMLInputElement).checked)"
    />
    <span class="at-checkbox__box" aria-hidden="true">
      <svg v-if="modelValue" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
        <path d="M4.5 12.5 9.5 17.5 20 6.5" />
      </svg>
    </span>
    <span v-if="label || description" class="at-checkbox__text">
      <span v-if="label" class="at-checkbox__label">{{ label }}</span>
      <span v-if="description" class="at-checkbox__desc">{{ description }}</span>
    </span>
    <slot />
  </label>
</template>

<style scoped>
.at-checkbox {
  display: flex;
  align-items: flex-start;
  gap: var(--at-space-3);
  cursor: pointer;
  min-height: var(--at-touch-min);
  padding: var(--at-space-1) 0;
}

.at-checkbox.is-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.at-checkbox__input {
  position: absolute;
  opacity: 0;
  width: 1px;
  height: 1px;
}

.at-checkbox__box {
  flex: none;
  width: 20px;
  height: 20px;
  margin-top: 2px;
  display: grid;
  place-items: center;
  border: 1.5px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-xs);
  background: var(--at-bg-surface);
  color: var(--at-text-on-primary);
  transition: background var(--at-duration-fast) var(--at-ease), border-color var(--at-duration-fast) var(--at-ease);
}

.at-checkbox__input:checked + .at-checkbox__box {
  background: var(--at-color-primary);
  border-color: var(--at-color-primary);
}

.at-checkbox__input:focus-visible + .at-checkbox__box {
  outline: 2px solid var(--at-color-ink);
  outline-offset: 2px;
}

.at-checkbox__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.at-checkbox__label {
  font-size: var(--at-text-body-sm);
  color: var(--at-text-primary);
}

.at-checkbox__desc {
  font-size: 13px;
  color: var(--at-color-muted);
}
</style>
