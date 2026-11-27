<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { ApiError } from '@/api/client';
import AppButton from '@/components/base/AppButton.vue';
import AppField from '@/components/base/AppField.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import { useToastStore } from '@/stores/toast';

const auth = useAuthStore();
const toast = useToastStore();
const router = useRouter();

const current = ref('');
const next = ref('');
const confirm = ref('');
const loading = ref(false);
const error = ref('');
const forced = computed(() => auth.mustChangePassword);

async function submit(): Promise<void> {
  error.value = '';
  if (!current.value || !next.value) {
    error.value = '请填写当前密码与新密码';
    return;
  }
  if (next.value.length < 6) {
    error.value = '新密码至少需要 6 个字符';
    return;
  }
  if (next.value === current.value) {
    error.value = '新密码不能与当前密码相同';
    return;
  }
  if (next.value !== confirm.value) {
    error.value = '两次输入的新密码不一致';
    return;
  }
  loading.value = true;
  try {
    await auth.changePassword(current.value, next.value);
    toast.success('密码已更新');
    await router.replace('/');
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '修改失败，请稍后重试';
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="at-page change">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">账号安全</p>
        <h1 class="at-page__title">{{ forced ? '先设置新密码' : '修改密码' }}</h1>
        <p class="at-page__subtitle">
          {{ forced ? '这是临时密码，首次登录需要设置自己的密码。' : '建议使用至少 6 个字符，并混合字母、数字与符号。' }}
        </p>
      </div>
    </header>
    <form class="change__form at-card" @submit.prevent="submit">
      <AppField label="当前密码" for-id="pw-current">
        <AppInput id="pw-current" v-model="current" type="password" autocomplete="current-password" />
      </AppField>
      <AppField label="新密码" for-id="pw-new" hint="至少 6 个字符">
        <AppInput id="pw-new" v-model="next" type="password" autocomplete="new-password" />
      </AppField>
      <AppField label="确认新密码" for-id="pw-confirm">
        <AppInput id="pw-confirm" v-model="confirm" type="password" autocomplete="new-password" />
      </AppField>
      <p v-if="error" class="change__error" role="alert"><AppIcon name="alert" :size="16" />{{ error }}</p>
      <div class="change__actions">
        <AppButton v-if="!forced" variant="ghost" type="button" @click="router.back()">取消</AppButton>
        <AppButton type="submit" :loading="loading">保存新密码</AppButton>
      </div>
    </form>
  </div>
</template>

<style scoped>
.change__form {
  max-width: 440px;
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
  padding: var(--at-space-5);
  border-radius: var(--at-radius-xl);
}

.change__error {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--at-color-danger);
  font-size: 13px;
}

.change__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--at-space-2);
  margin-top: var(--at-space-2);
}
</style>
