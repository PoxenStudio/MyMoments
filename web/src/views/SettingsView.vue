<script setup lang="ts">
import { useAuthStore } from '@/stores/auth';
import { useToastStore } from '@/stores/toast';
import { useRouter } from 'vue-router';
import AppButton from '@/components/base/AppButton.vue';

const auth = useAuthStore();
const toast = useToastStore();
const router = useRouter();

async function logout(): Promise<void> {
  await auth.logout();
  toast.push('已退出登录');
  await router.replace({ name: 'login' });
}
</script>

<template>
  <div class="at-page">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">设置</p>
        <h1 class="at-page__title">账号与家庭</h1>
        <p class="at-page__subtitle">成员、家庭信息与偏好设置将在这里管理。</p>
      </div>
    </header>
    <section class="settings__card">
      <div class="settings__row">
        <span class="at-caption-upper">当前账号</span>
        <span>{{ auth.user?.displayName }} · {{ auth.user?.role === 'admin' ? '管理员' : '成员' }}</span>
      </div>
      <div class="settings__row">
        <span class="at-caption-upper">家庭</span>
        <span>{{ auth.household?.name ?? '我的家' }}</span>
      </div>
      <div class="settings__row">
        <span class="at-caption-upper">时区</span>
        <span>{{ auth.household?.timezone ?? 'Asia/Shanghai' }}</span>
      </div>
    </section>
    <div class="settings__actions">
      <AppButton variant="outline" @click="router.push('/change-password')">修改密码</AppButton>
      <AppButton variant="ghost" @click="logout">退出登录</AppButton>
    </div>
  </div>
</template>

<style scoped>
.settings__card {
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-xl);
  padding: var(--at-space-2) var(--at-space-5);
  max-width: 620px;
}

.settings__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-4);
  padding: var(--at-space-4) 0;
  border-bottom: 1px solid var(--at-color-hairline);
  font-size: var(--at-text-body-sm);
}

.settings__row:last-child {
  border-bottom: 0;
}

.settings__actions {
  display: flex;
  gap: var(--at-space-2);
  margin-top: var(--at-space-5);
}
</style>
