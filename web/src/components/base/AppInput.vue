<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(defineProps<{
  modelValue: string | number;
  id?: string;
  type?: string;
  placeholder?: string;
  list?: string;
  invalid?: boolean;
  disabled?: boolean;
  autocomplete?: string;
  inputmode?: 'text' | 'numeric' | 'decimal' | 'tel' | 'email' | 'url' | 'search';
  maxlength?: number;
  min?: string | number;
  max?: string | number;
  step?: string | number;
}>(), {
  id: undefined,
  type: 'text',
  placeholder: undefined,
  list: undefined,
  invalid: false,
  disabled: false,
  autocomplete: undefined,
  inputmode: undefined,
  maxlength: undefined,
  min: undefined,
  max: undefined,
  step: undefined,
});

const emit = defineEmits<{ 'update:modelValue': [value: string] }>();

const value = computed({
  get: () => String(props.modelValue ?? ''),
  set: (v: string) => emit('update:modelValue', v),
});
</script>

<template>
  <input
    :id="id"
    v-model="value"
    class="at-input"
    :class="{ 'is-invalid': invalid }"
    :type="type"
    :placeholder="placeholder"
    :list="list"
    :disabled="disabled"
    :autocomplete="autocomplete"
    :inputmode="inputmode"
    :maxlength="maxlength"
    :min="min"
    :max="max"
    :step="step"
    :aria-invalid="invalid || undefined"
  />
</template>

<style scoped>
.at-input {
  width: 100%;
  height: 44px;
  padding: 0 var(--at-space-4);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  color: var(--at-text-primary);
  font-size: var(--at-text-body-sm);
  transition: border-color var(--at-duration-fast) var(--at-ease), box-shadow var(--at-duration-fast) var(--at-ease);
}

.at-input::placeholder {
  color: var(--at-color-muted-soft);
}

.at-input:focus {
  outline: none;
  border-color: var(--at-color-ink);
  box-shadow: 0 0 0 1px var(--at-color-ink);
}

.at-input.is-invalid {
  border-color: var(--at-color-danger);
}

.at-input:disabled {
  background: var(--at-bg-canvas-soft);
  color: var(--at-color-muted);
}

@media (max-width: 639px) {
  .at-input {
    font-size: 16px;
  }
}
</style>
