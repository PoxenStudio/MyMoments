<script setup lang="ts">
withDefaults(defineProps<{
  label?: string;
  forId?: string;
  hint?: string;
  error?: string;
  required?: boolean;
}>(), {
  label: undefined,
  forId: undefined,
  hint: undefined,
  error: undefined,
  required: false,
});
</script>

<template>
  <div class="at-field">
    <label v-if="label" class="at-field__label" :for="forId">
      {{ label }}
      <span v-if="required" class="at-field__required" aria-hidden="true">*</span>
    </label>
    <slot />
    <p v-if="error" class="at-field__error" role="alert">{{ error }}</p>
    <p v-else-if="hint" class="at-field__hint">{{ hint }}</p>
  </div>
</template>

<style scoped>
.at-field {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  min-width: 0;
}

.at-field__label {
  font-size: var(--at-text-caption);
  font-weight: 500;
  color: var(--at-text-primary);
}

.at-field__required {
  color: var(--at-color-danger);
}

.at-field__hint {
  font-size: 13px;
  color: var(--at-color-muted);
}

.at-field__error {
  font-size: 13px;
  color: var(--at-color-danger);
}
</style>
