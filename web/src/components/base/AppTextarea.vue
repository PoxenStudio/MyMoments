<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(defineProps<{
  modelValue: string;
  id?: string;
  placeholder?: string;
  rows?: number;
  invalid?: boolean;
  disabled?: boolean;
  maxlength?: number;
}>(), {
  id: undefined,
  placeholder: undefined,
  rows: 4,
  invalid: false,
  disabled: false,
  maxlength: undefined,
});

const emit = defineEmits<{ 'update:modelValue': [value: string] }>();

const value = computed({
  get: () => props.modelValue,
  set: (v: string) => emit('update:modelValue', v),
});
</script>

<template>
  <textarea
    :id="id"
    v-model="value"
    class="at-textarea"
    :class="{ 'is-invalid': invalid }"
    :rows="rows"
    :placeholder="placeholder"
    :disabled="disabled"
    :maxlength="maxlength"
    :aria-invalid="invalid || undefined"
  />
</template>

<style scoped>
.at-textarea {
  width: 100%;
  min-height: 96px;
  padding: var(--at-space-3) var(--at-space-4);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  color: var(--at-text-primary);
  font-size: var(--at-text-body-sm);
  line-height: 1.55;
  resize: vertical;
  transition: border-color var(--at-duration-fast) var(--at-ease), box-shadow var(--at-duration-fast) var(--at-ease);
}

.at-textarea::placeholder {
  color: var(--at-color-muted-soft);
}

.at-textarea:focus {
  outline: none;
  border-color: var(--at-color-ink);
  box-shadow: 0 0 0 1px var(--at-color-ink);
}

.at-textarea.is-invalid {
  border-color: var(--at-color-danger);
}

.at-textarea:disabled {
  background: var(--at-bg-canvas-soft);
  color: var(--at-color-muted);
}

@media (max-width: 639px) {
  .at-textarea {
    font-size: 16px;
  }
}
</style>
