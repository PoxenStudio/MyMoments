<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { ApiError, api } from '@/api/client';
import { useAuthStore } from '@/stores/auth';
import { useFamilyStore } from '@/stores/family';
import { useToastStore } from '@/stores/toast';
import type { DashboardLayoutItem, User } from '@/types';
import { cardDefs } from '@/cards/registry';
import { avatarUrl, uploadAvatar } from '@/utils/avatar';
import { relationOptions } from '@/data/relations';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppButton from '@/components/base/AppButton.vue';
import AppCheckbox from '@/components/base/AppCheckbox.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppErrorState from '@/components/base/AppErrorState.vue';
import AppField from '@/components/base/AppField.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppStatusTag from '@/components/base/AppStatusTag.vue';
import AppTabs from '@/components/base/AppTabs.vue';
import BirthdayManager from '@/components/business/BirthdayManager.vue';

const auth = useAuthStore();
const family = useFamilyStore();
const toast = useToastStore();
const route = useRoute();

type FamilyTab = 'members' | 'birthdays';
const tab = ref<FamilyTab>('members');
const tabs = [
  { key: 'members', label: '成员' },
  { key: 'birthdays', label: '生日' },
];
const pageTitle = computed(() => {
  if (tab.value === 'birthdays') return '生日';
  return '家庭成员';
});
const pageSubtitle = computed(() => {
  if (tab.value === 'birthdays') return '记录家人和重要的人的生日，按家庭时区提醒。';
  return '每位成员使用独立账号登录，管理员负责邀请与账号安全。';
});

const loading = ref(false);
const loadError = ref('');

const createOpen = ref(false);
const creating = ref(false);
const createError = ref('');
const createForm = reactive({ username: '', displayName: '', relation: '', role: 'member', mode: 'auto', password: '' });

const credentialsOpen = ref(false);
const credentials = reactive({ username: '', password: '' });

const editTarget = ref<User | null>(null);
const editSaving = ref(false);
const editError = ref('');
const editForm = reactive({ displayName: '', relation: '', role: 'member', status: 'active' });
const editAvatarSaving = ref(false);
const editAvatarInput = ref<HTMLInputElement | null>(null);
const memberCards = reactive<Record<string, boolean>>({});
const memberLayout = ref<DashboardLayoutItem[]>([]);
const memberCardsLoading = ref(false);
const editableCards = cardDefs.filter((def) => def.id !== 'overview');

const resetTarget = ref<User | null>(null);
const resetOpen = ref(false);
const resetting = ref(false);
const resetError = ref('');
const resetForm = reactive({ mode: 'auto', password: '' });

const statusTarget = ref<User | null>(null);
const statusSaving = ref(false);

const sortedMembers = computed(() => [...family.members].sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id));

async function load(): Promise<void> {
  loading.value = true;
  loadError.value = '';
  try {
    await family.load(true);
  } catch (err) {
    loadError.value = err instanceof ApiError ? err.message : '家庭成员加载失败';
  } finally {
    loading.value = false;
  }
}

function openCreate(): void {
  createForm.username = '';
  createForm.displayName = '';
  createForm.relation = '';
  createForm.role = 'member';
  createForm.mode = 'auto';
  createForm.password = '';
  createError.value = '';
  createOpen.value = true;
}

async function submitCreate(): Promise<void> {
  createError.value = '';
  if (!createForm.username.trim() || !createForm.displayName.trim()) {
    createError.value = '请填写登录名和称呼';
    return;
  }
  if (createForm.mode === 'manual' && createForm.password.length < 10) {
    createError.value = '临时密码至少需要 10 个字符';
    return;
  }
  creating.value = true;
  try {
    const res = await family.createMember({
      username: createForm.username.trim(),
      displayName: createForm.displayName.trim(),
      relation: createForm.relation.trim(),
      role: createForm.role,
      password: createForm.mode === 'manual' ? createForm.password : '',
    });
    createOpen.value = false;
    credentials.username = res.user.username;
    credentials.password = res.temporaryPassword;
    credentialsOpen.value = true;
    if (!res.temporaryPassword) toast.success('成员已创建');
  } catch (err) {
    createError.value = err instanceof ApiError ? err.message : '创建成员失败';
  } finally {
    creating.value = false;
  }
}

function openEdit(member: User): void {
  editTarget.value = member;
  editForm.displayName = member.displayName;
  editForm.relation = member.relation ?? '';
  editForm.role = member.role;
  editForm.status = member.status;
  editError.value = '';
  memberLayout.value = [];
  for (const def of editableCards) memberCards[def.id] = def.defaultVisible;
  if (auth.isAdmin) void loadMemberCards(member.id);
}

async function loadMemberCards(id: number): Promise<void> {
  memberCardsLoading.value = true;
  try {
    const res = await api.get<{ layout: DashboardLayoutItem[] }>(`/family/members/${id}/style`);
    memberLayout.value = res.layout ?? [];
    for (const def of editableCards) {
      const item = memberLayout.value.find((entry) => entry.id === def.id);
      memberCards[def.id] = item ? item.visible : def.defaultVisible;
    }
  } catch {
    for (const def of editableCards) memberCards[def.id] = def.defaultVisible;
  } finally {
    memberCardsLoading.value = false;
  }
}

function chooseEditAvatar(): void {
  editAvatarInput.value?.click();
}

async function onEditAvatarChange(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (!file || !editTarget.value) return;
  editAvatarSaving.value = true;
  try {
    const key = await uploadAvatar(file, editTarget.value.id);
    editTarget.value = { ...editTarget.value, avatarKey: key };
    toast.success('头像已更新');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '头像上传失败');
  } finally {
    editAvatarSaving.value = false;
  }
}

async function submitEdit(): Promise<void> {
  if (!editTarget.value) return;
  editError.value = '';
  const target = editTarget.value;
  const payload: Partial<Pick<User, 'displayName' | 'relation' | 'role' | 'status' | 'avatarKey'>> = {
    displayName: editForm.displayName.trim(),
    relation: editForm.relation.trim(),
  };
  if (auth.isAdmin) {
    payload.role = editForm.role === 'admin' ? 'admin' : 'member';
    payload.avatarKey = target.avatarKey;
    if (target.id !== auth.user?.id) payload.status = editForm.status === 'active' ? 'active' : 'inactive';
  }
  if (!payload.displayName) {
    editError.value = '请填写称呼';
    return;
  }
  editSaving.value = true;
  try {
    const updated = await family.updateMember(target.id, payload);
    if (auth.isAdmin && target.id !== auth.user?.id) {
      const byId = new Map(memberLayout.value.map((item) => [item.id, item]));
      const nextLayout: DashboardLayoutItem[] = cardDefs.map((def) => {
        const existing = byId.get(def.id);
        return {
          id: def.id,
          visible: def.id === 'overview' ? true : (memberCards[def.id] ?? existing?.visible ?? def.defaultVisible),
          size: existing?.size ?? 'standard',
          background: existing?.background ?? 'default',
        };
      });
      await api.patch(`/family/members/${target.id}/style`, { dashboard: JSON.stringify(nextLayout) });
    }
    if (updated.id === auth.user?.id) {
      auth.user = { ...updated };
    }
    editTarget.value = null;
    toast.success('成员资料已更新');
  } catch (err) {
    editError.value = err instanceof ApiError ? err.message : '保存失败';
  } finally {
    editSaving.value = false;
  }
}

function openReset(member: User): void {
  resetTarget.value = member;
  resetForm.mode = 'auto';
  resetForm.password = '';
  resetError.value = '';
  resetOpen.value = true;
}

async function submitReset(): Promise<void> {
  if (!resetTarget.value) return;
  if (resetForm.mode === 'manual' && resetForm.password.length < 10) {
    resetError.value = '临时密码至少需要 10 个字符';
    return;
  }
  resetting.value = true;
  resetError.value = '';
  try {
    const password = await family.resetMemberPassword(
      resetTarget.value.id,
      resetForm.mode === 'manual' ? resetForm.password : '',
    );
    credentials.username = resetTarget.value.username;
    credentials.password = password;
    resetOpen.value = false;
    resetTarget.value = null;
    credentialsOpen.value = true;
  } catch (err) {
    resetError.value = err instanceof ApiError ? err.message : '重置密码失败';
  } finally {
    resetting.value = false;
  }
}

async function toggleStatus(member: User): Promise<void> {
  statusSaving.value = true;
  try {
    const next = member.status === 'active' ? 'inactive' : 'active';
    await family.updateMember(member.id, { status: next });
    toast.success(next === 'active' ? '账号已恢复' : '账号已停用');
    statusTarget.value = null;
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '操作失败');
  } finally {
    statusSaving.value = false;
  }
}

async function copyCredentials(): Promise<void> {
  try {
    await navigator.clipboard.writeText(`用户名：${credentials.username}\n密码：${credentials.password}`);
    toast.success('已复制到剪贴板');
  } catch {
    toast.error('复制失败，请手动记录');
  }
}

function roleLabel(role: string): string {
  return role === 'admin' ? '管理员' : '成员';
}

function applyQueryTab(): void {
  const value = String(route.query.tab ?? '');
  if (value === 'birthdays' || value === 'members') tab.value = value;
}

watch(() => route.query.tab, applyQueryTab);

onMounted(() => {
  applyQueryTab();
  void load();
});
</script>

<template>
  <div class="at-page family">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">家庭</p>
        <h1 class="at-page__title">{{ pageTitle }}</h1>
        <p class="at-page__subtitle">{{ pageSubtitle }}</p>
      </div>
      <div v-if="auth.isAdmin && tab === 'members'" class="at-page__actions">
        <AppButton @click="openCreate"><AppIcon name="user-plus" :size="16" />添加成员</AppButton>
      </div>
    </header>

    <AppTabs v-model="tab" :tabs="tabs" aria-label="家庭内容" class="family__tabs" />

    <template v-if="tab === 'members'">
      <AppErrorState v-if="loadError" title="成员加载失败" :description="loadError" @retry="load" />
      <div v-else-if="loading" class="family__loading">正在加载成员…</div>

      <AppEmptyState
        v-else-if="sortedMembers.length === 0"
        icon="family"
        title="还没有家庭成员"
        description="管理员可以添加家庭成员账号。"
      >
        <AppButton v-if="auth.isAdmin" @click="openCreate">添加第一位成员</AppButton>
      </AppEmptyState>

      <ul v-else class="family__list">
        <li v-for="member in sortedMembers" :key="member.id" class="family__member at-card" :class="{ 'is-inactive': member.status !== 'active' }">
          <AppAvatar :name="member.displayName" :size="44" :src="avatarUrl(member.avatarKey)" />
          <div class="family__info">
            <div class="family__name-row">
              <span class="family__name">{{ member.displayName }}</span>
              <AppStatusTag v-if="member.relation" tone="brand">{{ member.relation }}</AppStatusTag>
              <AppStatusTag :tone="member.role === 'admin' ? 'warning' : 'neutral'">{{ roleLabel(member.role) }}</AppStatusTag>
              <AppStatusTag v-if="member.status !== 'active'" tone="danger">已停用</AppStatusTag>
              <AppStatusTag v-else-if="member.mustChangePassword" tone="warning">待改密码</AppStatusTag>
            </div>
            <p class="family__meta">@{{ member.username }}<span v-if="member.id === auth.user?.id"> · 这是你</span></p>
          </div>
          <div class="family__actions">
            <AppIconButton icon="edit" label="编辑成员" size="sm" @click="openEdit(member)" />
            <AppIconButton v-if="auth.isAdmin" icon="key" label="重置密码" size="sm" @click="openReset(member)" />
            <AppIconButton
              v-if="auth.isAdmin && member.id !== auth.user?.id"
              :icon="member.status === 'active' ? 'x-circle' : 'check-circle'"
              :label="member.status === 'active' ? '停用账号' : '恢复账号'"
              size="sm"
              @click="statusTarget = member"
            />
          </div>
        </li>
      </ul>
    </template>

    <BirthdayManager v-else :open-id="Number(route.query.open) || 0" />

    <AppDialog v-model="createOpen" title="添加家庭成员" size="md">
      <form class="at-stack" @submit.prevent="submitCreate">
        <div class="family__grid">
          <AppField label="登录名" for-id="member-username" hint="3–32 位字母、数字、下划线或短横线">
            <AppInput id="member-username" v-model="createForm.username" autocomplete="off" placeholder="例如：xiaohong" />
          </AppField>
          <AppField label="称呼" for-id="member-display">
            <AppInput id="member-display" v-model="createForm.displayName" placeholder="例如：小红" />
          </AppField>
        </div>
        <div class="family__grid">
          <AppField label="家庭关系" for-id="member-relation" hint="可选择或直接输入，如妈妈、女儿">
            <AppInput id="member-relation" v-model="createForm.relation" list="member-relation-options" placeholder="例如：妈妈" />
            <datalist id="member-relation-options">
              <option v-for="item in relationOptions" :key="item" :value="item" />
            </datalist>
          </AppField>
          <AppField label="角色" for-id="member-role">
            <select id="member-role" v-model="createForm.role" class="family__select">
              <option value="member">成员</option>
              <option value="admin">管理员</option>
            </select>
          </AppField>
        </div>
        <AppField label="初始密码" for-id="member-password-mode">
          <div class="family__segment" role="radiogroup" aria-label="初始密码方式">
            <button
              type="button"
              class="family__segment-btn"
              :class="{ 'is-active': createForm.mode === 'auto' }"
              role="radio"
              :aria-checked="createForm.mode === 'auto'"
              @click="createForm.mode = 'auto'"
            >
              系统生成
            </button>
            <button
              type="button"
              class="family__segment-btn"
              :class="{ 'is-active': createForm.mode === 'manual' }"
              role="radio"
              :aria-checked="createForm.mode === 'manual'"
              @click="createForm.mode = 'manual'"
            >
              自行设置
            </button>
          </div>
        </AppField>
        <AppField v-if="createForm.mode === 'manual'" label="临时密码" for-id="member-password" hint="至少 10 个字符，成员首次登录后需要修改">
          <AppInput id="member-password" v-model="createForm.password" type="password" autocomplete="new-password" />
        </AppField>
        <p v-else class="family__hint">创建后将生成随机临时密码，仅在成功页面显示一次。</p>
        <p v-if="createError" class="family__error" role="alert"><AppIcon name="alert" :size="16" />{{ createError }}</p>
      </form>
      <template #footer>
        <AppButton variant="ghost" @click="createOpen = false">取消</AppButton>
        <AppButton :loading="creating" @click="submitCreate">创建成员</AppButton>
      </template>
    </AppDialog>

    <AppDialog v-model="credentialsOpen" title="请记录临时密码" size="sm">
      <p class="family__hint">关闭后将无法再次查看，请通过安全渠道告知成员。</p>
      <div class="family__credentials">
        <div class="family__credential-row">
          <span class="at-caption-upper">用户名</span>
          <code>{{ credentials.username }}</code>
        </div>
        <div class="family__credential-row">
          <span class="at-caption-upper">临时密码</span>
          <code>{{ credentials.password }}</code>
        </div>
      </div>
      <template #footer>
        <AppButton variant="outline" @click="copyCredentials"><AppIcon name="copy" :size="16" />复制</AppButton>
        <AppButton @click="credentialsOpen = false">我已记录</AppButton>
      </template>
    </AppDialog>

    <AppDialog :model-value="editTarget !== null" title="编辑成员" size="md" @update:model-value="editTarget = null">
      <form class="at-stack" @submit.prevent="submitEdit">
        <div v-if="auth.isAdmin" class="family__avatar-edit">
          <button type="button" class="family__avatar-btn" :disabled="editAvatarSaving" title="更换头像" @click="chooseEditAvatar">
            <AppAvatar
              :name="editTarget?.displayName ?? '?'"
              :size="52"
              :src="avatarUrl(editTarget?.avatarKey)"
            />
            <span class="family__avatar-badge"><AppIcon name="camera" :size="14" /></span>
          </button>
          <div class="family__avatar-meta">
            <p class="family__avatar-title">{{ editTarget?.displayName }}</p>
            <p class="family__avatar-hint">{{ editAvatarSaving ? '正在上传…' : '点击头像为该成员更换头像' }}</p>
          </div>
          <input ref="editAvatarInput" type="file" accept="image/*" class="visually-hidden" @change="onEditAvatarChange" />
        </div>
        <AppField label="称呼" for-id="edit-display">
          <AppInput id="edit-display" v-model="editForm.displayName" />
        </AppField>
        <AppField label="家庭关系" for-id="edit-relation" hint="可选择或直接输入，如爸爸、女儿">
          <AppInput id="edit-relation" v-model="editForm.relation" list="edit-relation-options" placeholder="例如：爸爸" />
          <datalist id="edit-relation-options">
            <option v-for="item in relationOptions" :key="item" :value="item" />
          </datalist>
        </AppField>
        <div v-if="auth.isAdmin" class="family__grid">
          <AppField label="角色" for-id="edit-role">
            <select id="edit-role" v-model="editForm.role" class="family__select">
              <option value="member">成员</option>
              <option value="admin">管理员</option>
            </select>
          </AppField>
          <AppField v-if="editTarget && editTarget.id !== auth.user?.id" label="账号状态" for-id="edit-status">
            <select id="edit-status" v-model="editForm.status" class="family__select">
              <option value="active">启用</option>
              <option value="inactive">停用</option>
            </select>
          </AppField>
        </div>
        <AppField
          v-if="auth.isAdmin && editTarget && editTarget.id !== auth.user?.id"
          label="显示的卡片"
          hint="决定该成员首页显示哪些卡片，保存后立即生效"
        >
          <div class="family__cards" :class="{ 'is-loading': memberCardsLoading }">
            <AppCheckbox
              v-for="def in editableCards"
              :key="def.id"
              v-model="memberCards[def.id]"
              :label="def.title"
              :description="def.description"
              :disabled="memberCardsLoading"
            />
          </div>
        </AppField>
        <p v-if="editError" class="family__error" role="alert"><AppIcon name="alert" :size="16" />{{ editError }}</p>
      </form>
      <template #footer>
        <AppButton variant="ghost" @click="editTarget = null">取消</AppButton>
        <AppButton :loading="editSaving" @click="submitEdit">保存</AppButton>
      </template>
    </AppDialog>

    <AppDialog v-model="resetOpen" title="重置密码" size="md">
      <p class="family__hint">
        将为 {{ resetTarget?.displayName }} 设置新的临时密码，并退出其所有已登录设备。
      </p>
      <form class="at-stack" @submit.prevent="submitReset">
        <div class="family__segment" role="radiogroup" aria-label="密码生成方式">
          <button
            type="button"
            class="family__segment-btn"
            :class="{ 'is-active': resetForm.mode === 'auto' }"
            role="radio"
            :aria-checked="resetForm.mode === 'auto'"
            @click="resetForm.mode = 'auto'"
          >
            系统生成
          </button>
          <button
            type="button"
            class="family__segment-btn"
            :class="{ 'is-active': resetForm.mode === 'manual' }"
            role="radio"
            :aria-checked="resetForm.mode === 'manual'"
            @click="resetForm.mode = 'manual'"
          >
            自行设置
          </button>
        </div>
        <AppField v-if="resetForm.mode === 'manual'" label="临时密码" for-id="reset-password" hint="至少 10 个字符">
          <AppInput id="reset-password" v-model="resetForm.password" type="password" autocomplete="new-password" />
        </AppField>
        <p v-if="resetError" class="family__error" role="alert"><AppIcon name="alert" :size="16" />{{ resetError }}</p>
      </form>
      <template #footer>
        <AppButton variant="ghost" @click="resetOpen = false">取消</AppButton>
        <AppButton :loading="resetting" @click="submitReset">重置密码</AppButton>
      </template>
    </AppDialog>

    <AppConfirmDialog
      :model-value="statusTarget !== null"
      :title="statusTarget?.status === 'active' ? '停用该账号？' : '恢复该账号？'"
      :message="statusTarget?.status === 'active'
        ? `${statusTarget?.displayName} 将立即退出登录，历史内容会继续保留。`
        : `${statusTarget?.displayName} 将可以重新登录，但不会恢复旧会话。`"
      :danger="statusTarget?.status === 'active'"
      :confirm-text="statusTarget?.status === 'active' ? '停用账号' : '恢复账号'"
      :loading="statusSaving"
      @update:model-value="statusTarget = null"
      @confirm="statusTarget && toggleStatus(statusTarget)"
    />
  </div>
</template>

<style scoped>
.family__tabs {
  margin-bottom: var(--at-space-4);
}

.family__loading {
  color: var(--at-text-secondary);
  font-size: var(--at-text-body-sm);
  padding: var(--at-space-6) 0;
}

.family__list {
  list-style: none;
  display: grid;
  gap: var(--at-space-2);
}

.family__member {
  display: flex;
  align-items: center;
  gap: var(--at-space-4);
  padding: var(--at-space-4) var(--at-space-5);
  border-radius: var(--at-radius-xl);
}

.family__member.is-inactive {
  opacity: 0.62;
}

.family__info {
  flex: 1;
  min-width: 0;
}

.family__name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.family__name {
  font-weight: 500;
  font-size: var(--at-text-body);
}

.family__meta {
  margin-top: 2px;
  color: var(--at-color-muted);
  font-size: var(--at-text-caption);
}

.family__actions {
  display: flex;
  gap: 2px;
}

.family__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--at-space-3);
}

.family__avatar-edit {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  padding-bottom: var(--at-space-3);
  border-bottom: 1px solid var(--at-color-hairline);
}

.family__avatar-btn {
  position: relative;
  flex: none;
  padding: 0;
  border: 0;
  background: transparent;
  border-radius: 50%;
  cursor: pointer;
}

.family__avatar-badge {
  position: absolute;
  right: -2px;
  bottom: -2px;
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--at-color-ink);
  color: var(--at-text-on-primary);
  border: 2px solid var(--at-bg-surface);
}

.family__avatar-title {
  margin: 0;
  font-size: var(--at-text-body-sm);
  font-weight: 500;
}

.family__avatar-hint {
  margin: 2px 0 0;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.family__cards {
  display: grid;
  gap: 0;
  max-height: 260px;
  overflow-y: auto;
  padding: var(--at-space-2);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-canvas-soft);
}

.family__cards.is-loading {
  opacity: 0.6;
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

.family__select {
  width: 100%;
  min-height: var(--at-touch-min);
  padding: 0 var(--at-space-3);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface);
  color: var(--at-text-primary);
  font-size: var(--at-text-body-sm);
}

.family__segment {
  display: inline-flex;
  padding: 3px;
  background: var(--at-bg-surface-strong);
  border-radius: var(--at-radius-pill);
  gap: 2px;
}

.family__segment-btn {
  border: 0;
  background: transparent;
  padding: 7px 16px;
  border-radius: var(--at-radius-pill);
  font-size: var(--at-text-caption);
  color: var(--at-text-secondary);
  cursor: pointer;
  min-height: 32px;
}

.family__segment-btn.is-active {
  background: var(--at-bg-surface);
  color: var(--at-text-primary);
  box-shadow: var(--at-shadow-sm);
}

.family__hint {
  color: var(--at-text-secondary);
  font-size: var(--at-text-caption);
  line-height: 1.6;
}

.family__error {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--at-color-danger);
  font-size: 13px;
}

.family__credentials {
  margin-top: var(--at-space-4);
  border: 1px dashed var(--at-color-hairline-strong);
  border-radius: var(--at-radius-lg);
  padding: var(--at-space-3) var(--at-space-4);
  display: grid;
  gap: var(--at-space-3);
  background: var(--at-bg-canvas-soft);
}

.family__credential-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
}

.family__credential-row code {
  font-family: var(--at-font-mono);
  font-size: var(--at-text-body-sm);
  background: var(--at-bg-surface);
  padding: 3px 10px;
  border-radius: var(--at-radius-sm);
  border: 1px solid var(--at-color-hairline);
  user-select: all;
}

@media (max-width: 639px) {
  .family__member {
    flex-wrap: wrap;
    padding: var(--at-space-4);
  }

  .family__actions {
    width: 100%;
    justify-content: flex-end;
  }

  .family__grid {
    grid-template-columns: 1fr;
  }
}
</style>
