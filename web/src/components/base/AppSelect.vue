<script setup lang="ts">
import AppIcon from './AppIcon.vue';

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

withDefaults(defineProps<{
  modelValue: string;
  options: SelectOption[];
  id?: string;
  placeholder?: string;
  invalid?: boolean;
  disabled?: boolean;
  ariaLabel?: string;
}>(), {
  id: undefined,
  placeholder: undefined,
  invalid: false,
  disabled: false,
  ariaLabel: undefined,
});

const emit = defineEmits<{ 'update:modelValue': [value: string] }>();
</script>

<template>
  <div class="at-select">
    <select
      :id="id"
      class="at-select__control"
      :class="{ 'is-invalid': invalid }"
      :value="modelValue"
      :disabled="disabled"
      :aria-label="ariaLabel"
      :aria-invalid="invalid || undefined"
      @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
    >
      <option v-if="placeholder" value="" disabled>{{ placeholder }}</option>
      <option v-for="opt in options" :key="opt.value" :value="opt.value" :disabled="opt.disabled">
        {{ opt.label }}
      </option>
    </select>
    <AppIcon name="chevron-down" :size="16" class="at-select__chevron" />
  </div>
</template>

<style scoped>
.at-select {
  position: relative;
  width: 100%;
}

.at-select__control {
  width: 100%;
  height: 44px;
  padding: 0 var(--at-space-8) 0 var(--at-space-4);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  color: var(--at-text-primary);
  font-size: var(--at-text-body-sm);
  appearance: none;
  transition: border-color var(--at-duration-fast) var(--at-ease), box-shadow var(--at-duration-fast) var(--at-ease);
}

.at-select__control:focus {
  outline: none;
  border-color: var(--at-color-ink);
  box-shadow: 0 0 0 1px var(--at-color-ink);
}

.at-select__control.is-invalid {
  border-color: var(--at-color-danger);
}

.at-select__chevron {
  position: absolute;
  right: var(--at-space-3);
  top: 50%;
  transform: translateY(-50%);
  pointer-events: none;
  color: var(--at-color-muted);
}

@media (max-width: 639px) {
  .at-select__control {
    font-size: 16px;
  }
}
</style>
