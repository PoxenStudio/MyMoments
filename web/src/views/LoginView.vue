<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { ApiError } from '@/api/client';
import AppButton from '@/components/base/AppButton.vue';
import AppCheckbox from '@/components/base/AppCheckbox.vue';
import AppField from '@/components/base/AppField.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import { useToastStore } from '@/stores/toast';

const LAST_USERNAME_KEY = 'mm.auth.lastUsername';

const auth = useAuthStore();
const toast = useToastStore();
const route = useRoute();
const router = useRouter();

const username = ref('');
const password = ref('');
const remember = ref(true);
const lastUsername = ref('');
const showPassword = ref(false);
const loading = ref(false);
const error = ref('');

onMounted(() => {
  const saved = localStorage.getItem(LAST_USERNAME_KEY);
  if (!saved) return;
  lastUsername.value = saved;
  username.value = saved;
});

watch([username, password], () => {
  if (error.value) error.value = '';
});

async function submit(): Promise<void> {
  if (!username.value.trim() || !password.value) {
    error.value = '请输入用户名和密码';
    return;
  }
  loading.value = true;
  error.value = '';
  const name = username.value.trim();
  try {
    const user = await auth.login(name, password.value, remember.value);
    localStorage.setItem(LAST_USERNAME_KEY, name);
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
        <span class="at-display">朝夕</span>
      </div>
      <p class="auth__quote at-display">家里的事，<br />都放在这里。</p>
      <p class="auth__note">待办、日程、留言和回忆都在这里。</p>
      <div class="auth__orb auth__orb--mint" />
      <div class="auth__orb auth__orb--peach" />
    </aside>
    <section class="auth__panel">
      <div class="auth__form">
        <p class="at-caption-upper">登录</p>
        <h1 class="auth__title at-display">回到家里</h1>
        <p class="auth__sub">使用家庭管理员为你创建的账号登录。</p>
        <p v-if="lastUsername" class="auth__last">上次登录账号：{{ lastUsername }}</p>
        <form class="at-stack" @submit.prevent="submit">
          <AppField label="用户名" for-id="login-username">
            <AppInput id="login-username" v-model="username" autocomplete="username" placeholder="例如：xiaoming" />
          </AppField>
          <AppField label="密码" for-id="login-password">
            <div class="auth__password">
              <AppInput
                id="login-password"
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                placeholder="输入密码"
                :invalid="!!error"
              />
              <button
                type="button"
                class="auth__reveal"
                :aria-label="showPassword ? '隐藏密码' : '显示密码'"
                :aria-pressed="showPassword"
                :title="showPassword ? '隐藏密码' : '显示密码'"
                @click="showPassword = !showPassword"
              >
                <AppIcon :name="showPassword ? 'eye-off' : 'eye'" :size="18" />
              </button>
            </div>
          </AppField>
          <AppCheckbox
            id="login-remember"
            v-model="remember"
            label="记住登录"
            description="7 天内免登录"
          />
          <p v-if="error" class="auth__error" role="alert" aria-live="polite">
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

.auth__last {
  color: var(--at-color-muted);
  font-size: 13px;
  margin-top: calc(var(--at-space-2) * -1);
}

.auth__password {
  position: relative;
}

.auth__password :deep(.at-input) {
  padding-right: 46px;
}

.auth__reveal {
  position: absolute;
  right: 5px;
  top: 50%;
  transform: translateY(-50%);
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: var(--at-radius-md);
  color: var(--at-color-muted);
}

.auth__reveal:hover {
  background: var(--at-bg-canvas-soft);
  color: var(--at-color-ink);
}

.auth__error {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 10px var(--at-space-3);
  border: 1px solid color-mix(in srgb, var(--at-color-danger) 32%, transparent);
  border-radius: var(--at-radius-md);
  background: var(--at-color-danger-soft);
  color: var(--at-color-danger);
  font-size: 13px;
  line-height: 1.45;
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
