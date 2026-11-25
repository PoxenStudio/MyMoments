<script setup lang="ts">
import { ref } from 'vue';
import { ApiError } from '@/api/client';
import AppButton from '@/components/base/AppButton.vue';
import AppPanel from '@/components/base/AppPanel.vue';
import AppSwitch from '@/components/base/AppSwitch.vue';
import StyleEditor from '@/components/business/StyleEditor.vue';
import { useAuthStore } from '@/stores/auth';
import { useThemeStore } from '@/stores/theme';
import { useToastStore } from '@/stores/toast';
import type { ThemePatch } from '@/types';

const auth = useAuthStore();
const theme = useThemeStore();
const toast = useToastStore();
const busy = ref(false);

async function updatePersonal(patch: ThemePatch): Promise<void> {
  try {
    await theme.setPersonal(patch);
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '样式保存失败');
  }
}

async function toggleDefault(value: boolean): Promise<void> {
  busy.value = true;
  try {
    await theme.setUseDefault(value);
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '设置失败');
  } finally {
    busy.value = false;
  }
}

async function followHousehold(): Promise<void> {
  busy.value = true;
  try {
    await theme.setUseDefault(true);
    toast.push('已切换为跟随家庭样式');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '设置失败');
  } finally {
    busy.value = false;
  }
}

async function updateHousehold(patch: ThemePatch): Promise<void> {
  try {
    await theme.setHousehold(patch);
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '家庭样式保存失败');
  }
}

async function restoreHousehold(): Promise<void> {
  busy.value = true;
  try {
    await theme.resetHousehold();
    toast.push('家庭默认样式已恢复');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '恢复默认失败');
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <div class="style-settings">
    <AppPanel title="我的样式" subtitle="默认跟随家庭样式，自定义后仅覆盖你自己的显示。">
      <div class="style-settings__top">
        <AppSwitch
          :model-value="theme.useDefault"
          label="跟随家庭样式"
          description="开启后使用管理员设定的家庭全局样式"
          :disabled="busy"
          @update:model-value="toggleDefault"
        />
        <AppButton variant="ghost" size="sm" :disabled="busy || theme.useDefault" @click="followHousehold">跟随家庭样式</AppButton>
      </div>
      <StyleEditor
        v-if="!theme.useDefault"
        :model="theme.effective"
        @change="updatePersonal"
      />
      <p v-else class="style-settings__note">当前正在跟随家庭样式，关闭上方开关即可自定义。</p>
    </AppPanel>

    <AppPanel
      v-if="auth.isAdmin"
      title="家庭默认样式"
      subtitle="所有未自定义样式的成员都会看到这套设置。"
    >
      <div class="style-settings__top">
        <p class="style-settings__hint">品牌主色、文字与语义色不随样式改变。</p>
        <AppButton variant="ghost" size="sm" :disabled="busy" @click="restoreHousehold">恢复默认</AppButton>
      </div>
      <StyleEditor :model="theme.householdTheme" @change="updateHousehold" />
    </AppPanel>
  </div>
</template>

<style scoped>
.style-settings {
  display: grid;
  gap: var(--at-space-5);
}

.style-settings__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-4);
  margin-bottom: var(--at-space-4);
  flex-wrap: wrap;
}

.style-settings__top :deep(.at-switch-row) {
  flex: 1;
  min-width: 220px;
}

.style-settings__hint {
  margin: 0;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.style-settings__note {
  margin: var(--at-space-3) 0 0;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}
</style>
