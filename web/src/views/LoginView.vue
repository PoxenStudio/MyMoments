<script setup lang="ts">
import { ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { ApiError } from '@/api/client';
import AppButton from '@/components/base/AppButton.vue';
import AppField from '@/components/base/AppField.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import { useToastStore } from '@/stores/toast';

const auth = useAuthStore();
const toast = useToastStore();
const route = useRoute();
const router = useRouter();

const username = ref('');
const password = ref('');
const loading = ref(false);
const error = ref('');

async function submit(): Promise<void> {
  if (!username.value.trim() || !password.value) {
    error.value = '请输入用户名和密码';
    return;
  }
  loading.value = true;
  error.value = '';
  try {
    const user = await auth.login(username.value.trim(), password.value);
    toast.success(`欢迎回来，${user.displayName}`);
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
    await router.replace(redirect);
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '登录失败，请稍后重试';
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="auth">
    <aside class="auth__art">
      <div class="auth__brand">
        <img src="/icons/icon-192.png" alt="" width="40" height="40" />
        <span class="at-display">家时光</span>
      </div>
      <p class="auth__quote at-display">把日子过成<br />可以翻阅的样子。</p>
      <p class="auth__note">待办、日程、留言与回忆，都收在一个安静的家庭面板里。</p>
      <div class="auth__orb auth__orb--mint" />
      <div class="auth__orb auth__orb--peach" />
    </aside>
    <section class="auth__panel">
      <div class="auth__form">
        <p class="at-caption-upper">登录</p>
        <h1 class="auth__title at-display">回到家里</h1>
        <p class="auth__sub">使用家庭管理员为你创建的账号登录。</p>
        <form class="at-stack" @submit.prevent="submit">
          <AppField label="用户名" for-id="login-username">
            <AppInput id="login-username" v-model="username" autocomplete="username" placeholder="例如：xiaoming" />
          </AppField>
          <AppField label="密码" for-id="login-password">
            <AppInput id="login-password" v-model="password" type="password" autocomplete="current-password" placeholder="输入密码" />
          </AppField>
          <p v-if="error" class="auth__error" role="alert">
            <AppIcon name="alert" :size="16" />{{ error }}
          </p>
          <AppButton type="submit" block :loading="loading">登录</AppButton>
        </form>
        <p class="auth__help">忘记密码？请联系家庭管理员重置。</p>
      </div>
    </section>
  </div>
</template>

<style scoped>
.auth {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 1.05fr 1fr;
  background: var(--at-bg-page);
}

.auth__art {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: var(--at-space-4);
  padding: 64px;
  background: var(--at-bg-dark);
  color: var(--at-text-on-dark);
}

.auth__brand {
  position: absolute;
  top: 40px;
  left: 64px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 24px;
}

.auth__brand img {
  border-radius: 10px;
}

.auth__quote {
  position: relative;
  z-index: 2;
  font-size: clamp(40px, 4.6vw, 64px);
  line-height: 1.16;
  letter-spacing: -1px;
}

.auth__note {
  position: relative;
  z-index: 2;
  max-width: 380px;
  color: var(--at-text-on-dark-soft);
  font-size: var(--at-text-body-sm);
  line-height: 1.7;
}

.auth__orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(60px);
  opacity: 0.32;
}

.auth__orb--mint {
  width: 320px;
  height: 320px;
  right: -80px;
  bottom: -60px;
  background: var(--at-gradient-mint);
}

.auth__orb--peach {
  width: 240px;
  height: 240px;
  left: -60px;
  bottom: 80px;
  background: var(--at-gradient-peach);
}

.auth__panel {
  display: grid;
  place-items: center;
  padding: var(--at-space-6);
}

.auth__form {
  width: min(100%, 380px);
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.auth__title {
  font-size: 40px;
}

.auth__sub {
  color: var(--at-text-secondary);
  font-size: var(--at-text-caption);
  margin-top: calc(var(--at-space-2) * -1);
  margin-bottom: var(--at-space-2);
}

.auth__error {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--at-color-danger);
  font-size: 13px;
}

.auth__help {
  color: var(--at-color-muted);
  font-size: 13px;
  text-align: center;
}

@media (max-width: 899px) {
  .auth {
    grid-template-columns: 1fr;
  }

  .auth__art {
    display: none;
  }

  .auth__panel {
    min-height: 100vh;
  }

  .auth__title {
    font-size: 34px;
  }
}
</style>
