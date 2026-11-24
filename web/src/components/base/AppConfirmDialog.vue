<script setup lang="ts">
import AppButton from './AppButton.vue';
import AppDialog from './AppDialog.vue';

withDefaults(defineProps<{
  modelValue: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
  loading?: boolean;
}>(), {
  confirmText: '确认',
  cancelText: '取消',
  danger: false,
  loading: false,
});

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  confirm: [];
}>();
</script>

<template>
  <AppDialog
    :model-value="modelValue"
    :title="title"
    size="sm"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <p class="at-confirm__message">{{ message }}</p>
    <template #footer>
      <AppButton variant="ghost" @click="emit('update:modelValue', false)">{{ cancelText }}</AppButton>
      <AppButton :variant="danger ? 'danger' : 'primary'" :loading="loading" @click="emit('confirm')">
        {{ confirmText }}
      </AppButton>
    </template>
  </AppDialog>
</template>

<style scoped>
.at-confirm__message {
  color: var(--at-text-secondary);
  font-size: var(--at-text-body-sm);
  line-height: 1.6;
}
</style>
