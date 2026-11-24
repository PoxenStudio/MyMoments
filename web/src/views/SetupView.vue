<script setup lang="ts">
import { ref } from 'vue';
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

const householdName = ref('我的家');
const username = ref('');
const displayName = ref('');
const password = ref('');
const confirm = ref('');
const loading = ref(false);
const error = ref('');

async function submit(): Promise<void> {
  error.value = '';
  if (!username.value.trim() || !displayName.value.trim()) {
    error.value = '请填写用户名和称呼';
    return;
  }
  if (password.value.length < 10) {
    error.value = '密码至少需要 10 个字符';
    return;
  }
  if (password.value !== confirm.value) {
    error.value = '两次输入的密码不一致';
    return;
  }
  loading.value = true;
  try {
    const user = await auth.setup({
      householdName: householdName.value.trim() || '我的家',
      username: username.value.trim(),
      displayName: displayName.value.trim(),
      password: password.value,
    });
    toast.success(`家时光已就绪，${user.displayName}`);
    await router.replace('/');
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '初始化失败，请稍后重试';
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="setup">
    <span class="setup__orb setup__orb--mint" />
    <span class="setup__orb setup__orb--peach" />
    <div class="setup__panel">
      <div class="setup__brand">
        <img src="/icons/icon-192.png" alt="" width="36" height="36" />
        <span class="at-display">家时光</span>
      </div>
      <p class="at-caption-upper">首次初始化</p>
      <h1 class="setup__title at-display">为这个家，<br />起一个名字。</h1>
      <p class="setup__sub">创建第一个管理员账号。初始化完成后，其他家庭成员由管理员邀请加入。</p>
      <form class="at-stack" @submit.prevent="submit">
        <AppField label="家庭名称" for-id="setup-household">
          <AppInput id="setup-household" v-model="householdName" placeholder="我的家" />
        </AppField>
        <div class="setup__row">
          <AppField label="用户名" for-id="setup-username" hint="用于登录，仅限字母、数字、下划线和短横线">
            <AppInput id="setup-username" v-model="username" autocomplete="username" placeholder="例如：xiaoming" />
          </AppField>
          <AppField label="你的称呼" for-id="setup-display">
            <AppInput id="setup-display" v-model="displayName" autocomplete="nickname" placeholder="例如：小明" />
          </AppField>
        </div>
        <AppField label="密码" for-id="setup-password" hint="至少 10 个字符，建议混合字母、数字与符号">
          <AppInput id="setup-password" v-model="password" type="password" autocomplete="new-password" />
        </AppField>
        <AppField label="确认密码" for-id="setup-confirm">
          <AppInput id="setup-confirm" v-model="confirm" type="password" autocomplete="new-password" />
        </AppField>
        <p v-if="error" class="setup__error" role="alert"><AppIcon name="alert" :size="16" />{{ error }}</p>
        <AppButton type="submit" block size="lg" :loading="loading">创建家庭</AppButton>
      </form>
    </div>
  </div>
</template>

<style scoped>
.setup {
  position: relative;
  overflow: hidden;
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: var(--at-space-6);
  background: var(--at-bg-dark);
}

.setup__orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(70px);
  opacity: 0.3;
  pointer-events: none;
}

.setup__orb--mint {
  width: 420px;
  height: 420px;
  top: -140px;
  right: -80px;
  background: var(--at-gradient-mint);
}

.setup__orb--peach {
  width: 320px;
  height: 320px;
  bottom: -110px;
  left: -70px;
  background: var(--at-gradient-peach);
}

.setup__panel {
  position: relative;
  z-index: 1;
  width: min(100%, 520px);
  padding: var(--at-space-8);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-xxl);
  box-shadow: var(--at-shadow-dialog);
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.setup__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 22px;
  margin-bottom: var(--at-space-2);
}

.setup__brand img {
  border-radius: 9px;
}

.setup__title {
  font-size: 40px;
  line-height: 1.15;
}

.setup__sub {
  color: var(--at-text-secondary);
  font-size: var(--at-text-caption);
  line-height: 1.65;
  margin-top: calc(var(--at-space-2) * -1);
  margin-bottom: var(--at-space-2);
}

.setup__row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--at-space-3);
}

.setup__error {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--at-color-danger);
  font-size: 13px;
}

@media (max-width: 639px) {
  .setup {
    padding: var(--at-space-4);
  }

  .setup__panel {
    padding: var(--at-space-5);
  }

  .setup__title {
    font-size: 30px;
  }

  .setup__row {
    grid-template-columns: 1fr;
  }
}
</style>
