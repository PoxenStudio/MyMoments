<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { ApiError } from '@/api/client';
import { useAuthStore } from '@/stores/auth';
import { useToastStore } from '@/stores/toast';
import { avatarUrl, uploadAvatar } from '@/utils/avatar';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppButton from '@/components/base/AppButton.vue';
import AppField from '@/components/base/AppField.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppPanel from '@/components/base/AppPanel.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import AppStatusTag from '@/components/base/AppStatusTag.vue';
import ApiTokensPanel from '@/components/business/ApiTokensPanel.vue';
import StyleSettingsPanel from '@/components/business/StyleSettingsPanel.vue';

const auth = useAuthStore();
const toast = useToastStore();
const router = useRouter();

const profile = reactive({ displayName: '', relation: '' });
const profileSaving = ref(false);
const profileError = ref('');
const avatarUploading = ref(false);
const avatarInput = ref<HTMLInputElement | null>(null);

const household = reactive({ name: '', timezone: 'Asia/Shanghai', weekStart: '1', dateFormat: '24h', defaultReminderOffset: '0' });
const householdSaving = ref(false);
const householdError = ref('');

const timezoneOptions = [
  { value: 'Asia/Shanghai', label: '中国标准时间（Asia/Shanghai）' },
  { value: 'Asia/Hong_Kong', label: '香港（Asia/Hong_Kong）' },
  { value: 'Asia/Taipei', label: '台北（Asia/Taipei）' },
  { value: 'Asia/Tokyo', label: '东京（Asia/Tokyo）' },
  { value: 'Asia/Singapore', label: '新加坡（Asia/Singapore）' },
  { value: 'America/Los_Angeles', label: '洛杉矶（America/Los_Angeles）' },
  { value: 'America/New_York', label: '纽约（America/New_York）' },
  { value: 'Europe/London', label: '伦敦（Europe/London）' },
  { value: 'UTC', label: '协调世界时（UTC）' },
];

const weekStartOptions = [
  { value: '1', label: '周一' },
  { value: '0', label: '周日' },
];

const dateFormatOptions = [
  { value: '24h', label: '24 小时制（18:30）' },
  { value: '12h', label: '12 小时制（下午 6:30）' },
];

const relationOptions = [
  '爸爸', '妈妈', '女儿', '儿子', '爷爷', '奶奶', '外公',
  '外婆', '姥姥', '姥爷', '叔叔', '姑姑', '姨', '舅',
];

async function saveProfile(): Promise<void> {
  profileError.value = '';
  if (!profile.displayName.trim()) {
    profileError.value = '请填写称呼';
    return;
  }
  profileSaving.value = true;
  try {
    await auth.updateProfile({
      displayName: profile.displayName.trim(),
      relation: profile.relation.trim(),
    });
    toast.success('个人资料已保存');
  } catch (err) {
    profileError.value = err instanceof ApiError ? err.message : '保存失败';
  } finally {
    profileSaving.value = false;
  }
}

function chooseAvatar(): void {
  avatarInput.value?.click();
}

async function onAvatarChange(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (!file) return;
  avatarUploading.value = true;
  try {
    const key = await uploadAvatar(file);
    await auth.updateProfile({ avatarKey: key });
    toast.success('头像已更新');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '头像上传失败');
  } finally {
    avatarUploading.value = false;
  }
}

function customizeCards(): void {
  void router.push({ name: 'home', query: { customize: '1' } });
}

async function saveHousehold(): Promise<void> {
  householdError.value = '';
  if (!household.name.trim()) {
    householdError.value = '请填写家庭名称';
    return;
  }
  householdSaving.value = true;
  try {
    await auth.updateHousehold({
      name: household.name.trim(),
      timezone: household.timezone,
      weekStart: Number(household.weekStart),
      dateFormat: household.dateFormat as '24h' | '12h',
      defaultReminderOffset: Number(household.defaultReminderOffset) || 0,
    });
    toast.success('家庭设置已保存');
  } catch (err) {
    householdError.value = err instanceof ApiError ? err.message : '保存失败';
  } finally {
    householdSaving.value = false;
  }
}

function syncFromState(): void {
  profile.displayName = auth.user?.displayName ?? '';
  profile.relation = auth.user?.relation ?? '';
  household.name = auth.household?.name ?? '';
  household.timezone = auth.household?.timezone ?? 'Asia/Shanghai';
  household.weekStart = String(auth.household?.weekStart ?? 1);
  household.dateFormat = auth.household?.dateFormat ?? '24h';
  household.defaultReminderOffset = String(auth.household?.defaultReminderOffset ?? 0);
}

onMounted(syncFromState);
</script>

<template>
  <div class="at-page settings">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">设置</p>
        <h1 class="at-page__title">账号与家庭</h1>
        <p class="at-page__subtitle">管理你的个人资料、账号安全与家庭设置。</p>
      </div>
    </header>

    <div class="settings__grid">
      <AppPanel
        title="个人资料"
        subtitle="其他成员在成员列表与内容中看到的信息。"
        :class="auth.isAdmin ? undefined : 'settings__card--full'"
      >
        <form class="at-stack" @submit.prevent="saveProfile">
          <div class="settings__identity">
            <button type="button" class="settings__avatar-btn" :disabled="avatarUploading" title="更换头像" @click="chooseAvatar">
              <AppAvatar
                :name="profile.displayName || auth.user?.displayName || '?'"
                :size="52"
                :src="avatarUrl(auth.user?.avatarKey)"
              />
              <span class="settings__avatar-edit"><AppIcon name="camera" :size="14" /></span>
            </button>
            <input ref="avatarInput" type="file" accept="image/*" class="visually-hidden" @change="onAvatarChange" />
            <div>
              <p class="settings__name">{{ auth.user?.displayName }}</p>
              <p class="settings__handle">@{{ auth.user?.username }}</p>
              <p class="settings__avatar-hint">{{ avatarUploading ? '正在上传…' : '点击头像更换' }}</p>
            </div>
            <AppStatusTag :tone="auth.isAdmin ? 'warning' : 'neutral'">{{ auth.isAdmin ? '管理员' : '成员' }}</AppStatusTag>
            <AppButton variant="outline" size="sm" type="button" @click="router.push('/change-password')">修改密码</AppButton>
          </div>
          <div class="settings__row-2">
            <AppField label="称呼" for-id="profile-name">
              <AppInput id="profile-name" v-model="profile.displayName" placeholder="例如：小明" />
            </AppField>
            <AppField label="家庭关系" for-id="profile-relation" hint="可选择或直接输入，如爸爸、女儿">
              <AppInput id="profile-relation" v-model="profile.relation" list="relation-options" placeholder="例如：爸爸" />
              <datalist id="relation-options">
                <option v-for="item in relationOptions" :key="item" :value="item" />
              </datalist>
            </AppField>
          </div>
          <div class="settings__pref-row">
            <div>
              <p class="settings__account-label">显示的卡片</p>
              <p class="settings__account-hint">决定首页显示哪些卡片，可随时调整。</p>
            </div>
            <AppButton variant="outline" size="sm" type="button" @click="customizeCards">整理首页卡片</AppButton>
          </div>
          <p v-if="profileError" class="settings__error" role="alert"><AppIcon name="alert" :size="16" />{{ profileError }}</p>
          <div class="settings__actions">
            <AppButton type="submit" :loading="profileSaving">保存资料</AppButton>
          </div>
        </form>
      </AppPanel>

      <AppPanel v-if="auth.isAdmin" title="家庭设置" subtitle="对全家生效，仅管理员可修改。">
        <form class="at-stack" @submit.prevent="saveHousehold">
          <AppField label="家庭名称" for-id="hh-name">
            <AppInput id="hh-name" v-model="household.name" placeholder="我的家" />
          </AppField>
          <div class="settings__row-2">
            <AppField label="时间格式" for-id="hh-date" hint="全家统一的时间显示格式">
              <AppSelect id="hh-date" v-model="household.dateFormat" :options="dateFormatOptions" />
            </AppField>
            <AppField label="默认提醒提前量" for-id="hh-reminder" hint="单位：分钟，0 表示不提前">
              <AppInput id="hh-reminder" v-model="household.defaultReminderOffset" type="number" min="0" max="10080" />
            </AppField>
          </div>
          <div class="settings__row-2">
            <AppField label="家庭时区" for-id="hh-tz" hint="日期与提醒按此时区计算">
              <AppSelect id="hh-tz" v-model="household.timezone" :options="timezoneOptions" />
            </AppField>
            <AppField label="每周起始日" for-id="hh-week">
              <AppSelect id="hh-week" v-model="household.weekStart" :options="weekStartOptions" />
            </AppField>
          </div>
          <p v-if="householdError" class="settings__error" role="alert"><AppIcon name="alert" :size="16" />{{ householdError }}</p>
          <div class="settings__actions">
            <AppButton type="submit" :loading="householdSaving">保存设置</AppButton>
          </div>
        </form>
      </AppPanel>
    </div>

    <div class="settings__style">
      <StyleSettingsPanel />
    </div>

    <div class="settings__api">
      <ApiTokensPanel />
    </div>
  </div>
</template>

<style scoped>
.settings__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--at-space-4);
  align-items: stretch;
}

.settings__card--full {
  grid-column: 1 / -1;
}

.settings__api {
  margin-top: var(--at-space-4);
}

.settings__style {
  margin-top: var(--at-space-4);
}

.settings__avatar-btn {
  position: relative;
  flex: none;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
  border-radius: 50%;
}

.settings__avatar-edit {
  position: absolute;
  right: -2px;
  bottom: -2px;
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--at-color-ink);
  color: #fff;
  border: 2px solid var(--at-bg-surface);
}

.settings__avatar-hint {
  margin-top: 2px;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted-soft);
}

.settings__pref-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
  padding: var(--at-space-3);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-canvas-soft);
}

.visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
  border: 0;
}

.settings__identity {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--at-space-3);
  padding-bottom: var(--at-space-3);
  border-bottom: 1px solid var(--at-color-hairline);
}

.settings__identity > div {
  flex: 1;
  min-width: 0;
}

.settings__name {
  font-weight: 500;
  font-size: var(--at-text-body);
}

.settings__handle {
  color: var(--at-color-muted);
  font-size: var(--at-text-caption);
}

.settings__row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--at-space-3);
}

.settings__actions {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--at-space-1);
}

.settings__error {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--at-color-danger);
  font-size: 13px;
}

.settings__account-label {
  font-size: var(--at-text-body-sm);
  font-weight: 500;
}

.settings__account-hint {
  margin-top: 2px;
  color: var(--at-color-muted);
  font-size: var(--at-text-caption);
}

@media (max-width: 639px) {
  .settings__grid {
    grid-template-columns: 1fr;
  }

  .settings__row-2 {
    grid-template-columns: 1fr;
  }
}
</style>
