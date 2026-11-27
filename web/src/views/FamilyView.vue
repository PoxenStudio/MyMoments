<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { ApiError, api } from '@/api/client';
import { useAuthStore } from '@/stores/auth';
import { useFamilyStore } from '@/stores/family';
import { useCalendarStore, type BirthdayDatePayload } from '@/stores/calendar';
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
import AppSelect from '@/components/base/AppSelect.vue';
import AppStatusTag from '@/components/base/AppStatusTag.vue';
import AppTabs from '@/components/base/AppTabs.vue';
import BirthdayManager from '@/components/business/BirthdayManager.vue';
import AvatarCropDialog from '@/components/business/AvatarCropDialog.vue';

const auth = useAuthStore();
const family = useFamilyStore();
const calendar = useCalendarStore();
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
const createForm = reactive({
  username: '',
  displayName: '',
  relation: '',
  role: 'member',
  mode: 'auto',
  password: '',
  birthMonth: '',
  birthDay: '',
  birthYear: '',
  birthLunarEnabled: false,
  birthLunarMonth: '',
  birthLunarDay: '',
  birthLunarYear: '',
  birthLunarReminder: false,
});
const createShowPassword = ref(false);

const birthMonthOptions = Array.from({ length: 12 }, (_, index) => ({
  value: String(index + 1),
  label: `${index + 1} 月`,
}));
function birthDayOptions(month: string): { value: string; label: string }[] {
  const value = Number(month) || 1;
  const last = value === 2 ? 29 : new Date(2024, value, 0).getDate();
  return Array.from({ length: last }, (_, index) => ({ value: String(index + 1), label: `${index + 1} 日` }));
}
const createBirthDays = computed(() => birthDayOptions(createForm.birthMonth));
watch(() => createForm.birthMonth, () => {
  if (Number(createForm.birthDay) > createBirthDays.value.length) createForm.birthDay = '';
});
const createLunarDays = computed(() => Array.from({ length: 30 }, (_, index) => ({
  value: String(index + 1),
  label: `${index + 1} 日`,
})));

const credentialsOpen = ref(false);
const credentials = reactive({ username: '', password: '' });

const editTarget = ref<User | null>(null);
const editSaving = ref(false);
const editError = ref('');
const editForm = reactive({
  displayName: '',
  relation: '',
  role: 'member',
  status: 'active',
  birthMonth: '',
  birthDay: '',
  birthYear: '',
  birthLunarEnabled: false,
  birthLunarMonth: '',
  birthLunarDay: '',
  birthLunarYear: '',
  birthLunarReminder: false,
});
const editBirthDays = computed(() => birthDayOptions(editForm.birthMonth));
watch(() => editForm.birthMonth, () => {
  if (Number(editForm.birthDay) > editBirthDays.value.length) editForm.birthDay = '';
});
const editLunarDays = computed(() => Array.from({ length: 30 }, (_, index) => ({
  value: String(index + 1),
  label: `${index + 1} 日`,
})));
const editAvatarSaving = ref(false);
const editAvatarInput = ref<HTMLInputElement | null>(null);
const editCropOpen = ref(false);
const editCropFile = ref<File | null>(null);
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
  createForm.birthMonth = '';
  createForm.birthDay = '';
  createForm.birthYear = '';
  createForm.birthLunarEnabled = false;
  createForm.birthLunarMonth = '';
  createForm.birthLunarDay = '';
  createForm.birthLunarYear = '';
  createForm.birthLunarReminder = false;
  createShowPassword.value = false;
  createError.value = '';
  createOpen.value = true;
}

async function submitCreate(): Promise<void> {
  createError.value = '';
  if (!createForm.username.trim() || !createForm.displayName.trim()) {
    createError.value = '请填写登录名和称呼';
    return;
  }
  const birthMonth = Number(createForm.birthMonth);
  const birthDay = Number(createForm.birthDay);
  if (!birthMonth || !birthDay) {
    createError.value = '请选择生日月份和日期';
    return;
  }
  const birthYear = createForm.birthYear.trim() ? Number(createForm.birthYear) : null;
  if (birthYear !== null && (birthYear < 1900 || birthYear > new Date().getFullYear())) {
    createError.value = '出生年份无效';
    return;
  }
  const dates: BirthdayDatePayload[] = [{
    calendar: 'solar',
    month: birthMonth,
    day: birthDay,
    yearUnknown: birthYear === null,
    birthYear,
    reminderOffset: 0,
  }];
  if (createForm.birthLunarEnabled) {
    const lunarMonth = Number(createForm.birthLunarMonth);
    const lunarDay = Number(createForm.birthLunarDay);
    if (!lunarMonth || !lunarDay) {
      createError.value = '请选择农历生日月份和日期';
      return;
    }
    const lunarYear = createForm.birthLunarYear.trim() ? Number(createForm.birthLunarYear) : null;
    if (lunarYear !== null && (lunarYear < 1900 || lunarYear > new Date().getFullYear())) {
      createError.value = '农历出生年份无效';
      return;
    }
    dates.push({
      calendar: 'lunar',
      month: lunarMonth,
      day: lunarDay,
      yearUnknown: lunarYear === null,
      birthYear: lunarYear,
      reminderOffset: createForm.birthLunarReminder ? 0 : null,
    });
  }
  if (createForm.mode === 'manual' && createForm.password.length < 6) {
    createError.value = '临时密码至少需要 6 个字符';
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
    try {
      await calendar.createBirthday({
        name: createForm.displayName.trim(),
        userId: res.user.id,
        dates,
      });
    } catch (err) {
      toast.error(err instanceof ApiError ? `生日保存失败：${err.message}` : '生日保存失败，可在生日列表中补充');
    }
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
  editForm.birthMonth = '';
  editForm.birthDay = '';
  editForm.birthYear = '';
  editForm.birthLunarEnabled = false;
  editForm.birthLunarMonth = '';
  editForm.birthLunarDay = '';
  editForm.birthLunarYear = '';
  editForm.birthLunarReminder = false;
  editError.value = '';
  memberLayout.value = [];
  for (const def of editableCards) memberCards[def.id] = def.defaultVisible;
  void loadMemberBirthday(member.id);
  if (auth.isAdmin) void loadMemberCards(member.id);
}

/** 读取成员已关联的生日，用于在编辑表单中回填。 */
async function loadMemberBirthday(userId: number): Promise<void> {
  try {
    await calendar.loadBirthdays();
  } catch {
    return;
  }
  const birthday = calendar.birthdays.find((item) => item.userId === userId);
  if (!birthday || editTarget.value?.id !== userId) return;
  const solar = birthday.dates.find((date) => date.calendar === 'solar');
  if (solar) {
    editForm.birthMonth = String(solar.month);
    editForm.birthDay = String(solar.day);
    editForm.birthYear = solar.yearUnknown || !solar.birthYear ? '' : String(solar.birthYear);
  }
  const lunar = birthday.dates.find((date) => date.calendar === 'lunar');
  if (lunar) {
    editForm.birthLunarEnabled = true;
    editForm.birthLunarMonth = String(lunar.month);
    editForm.birthLunarDay = String(lunar.day);
    editForm.birthLunarYear = lunar.yearUnknown || !lunar.birthYear ? '' : String(lunar.birthYear);
    editForm.birthLunarReminder = lunar.reminderOffset != null;
  }
}

/** 保存编辑表单中的生日：公历必填，农历按开关写入。 */
async function saveMemberBirthday(userId: number): Promise<void> {
  try {
    await calendar.loadBirthdays();
  } catch {
    // 读取失败时按未关联处理，后续接口会返回具体错误。
  }
  const month = Number(editForm.birthMonth);
  const day = Number(editForm.birthDay);
  const existing = calendar.birthdays.find((item) => item.userId === userId);
  if (!month || !day) {
    if (existing) await calendar.deleteBirthday(existing.id);
    return;
  }
  const birthYear = editForm.birthYear.trim() ? Number(editForm.birthYear) : null;
  const dates: BirthdayDatePayload[] = [{
    calendar: 'solar',
    month,
    day,
    yearUnknown: birthYear === null,
    birthYear,
    reminderOffset: 0,
  }];
  if (editForm.birthLunarEnabled) {
    const lunarMonth = Number(editForm.birthLunarMonth);
    const lunarDay = Number(editForm.birthLunarDay);
    const lunarYear = editForm.birthLunarYear.trim() ? Number(editForm.birthLunarYear) : null;
    if (lunarMonth && lunarDay) {
      dates.push({
        calendar: 'lunar',
        month: lunarMonth,
        day: lunarDay,
        yearUnknown: lunarYear === null,
        birthYear: lunarYear,
        reminderOffset: editForm.birthLunarReminder ? 0 : null,
      });
    }
  }
  const payload = {
    name: editForm.displayName.trim(),
    dates,
  };
  if (existing) await calendar.updateBirthday(existing.id, payload);
  else await calendar.createBirthday({ ...payload, userId });
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

function onEditAvatarChange(event: Event): void {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (!file) return;
  editCropFile.value = file;
  editCropOpen.value = true;
}

async function onEditAvatarCropped(file: File): Promise<void> {
  if (!editTarget.value) return;
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
  const birthMonth = Number(editForm.birthMonth);
  const birthDay = Number(editForm.birthDay);
  if (auth.isAdmin) {
    if (Boolean(birthMonth) !== Boolean(birthDay)) {
      editError.value = '请选择完整的生日月份和日期';
      return;
    }
    if (editForm.birthYear.trim()) {
      const year = Number(editForm.birthYear);
      if (year < 1900 || year > new Date().getFullYear()) {
        editError.value = '出生年份无效';
        return;
      }
    }
    if (editForm.birthLunarEnabled) {
      const lunarMonth = Number(editForm.birthLunarMonth);
      const lunarDay = Number(editForm.birthLunarDay);
      if (!lunarMonth || !lunarDay) {
        editError.value = '请选择完整的农历生日月份和日期';
        return;
      }
      if (editForm.birthLunarYear.trim()) {
        const lunarYear = Number(editForm.birthLunarYear);
        if (lunarYear < 1900 || lunarYear > new Date().getFullYear()) {
          editError.value = '农历出生年份无效';
          return;
        }
      }
    }
  }
  editSaving.value = true;
  try {
    const updated = await family.updateMember(target.id, payload);
    if (auth.isAdmin) await saveMemberBirthday(target.id);
    if (auth.isAdmin && target.id !== auth.user?.id) {
      const byId = new Map(memberLayout.value.map((item) => [item.id, item]));
      const nextLayout: DashboardLayoutItem[] = cardDefs.map((def) => {
        const existing = byId.get(def.id);
        const wide = def.id === 'overview' || def.id === 'timeline';
        return {
          id: def.id,
          visible: def.id === 'overview' ? true : (memberCards[def.id] ?? existing?.visible ?? def.defaultVisible),
          background: existing?.background ?? 'default',
          x: existing?.x ?? 0,
          y: existing?.y ?? 0,
          w: existing?.w ?? (wide ? 8 : 4),
          h: existing?.h ?? 6,
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
  if (resetForm.mode === 'manual' && resetForm.password.length < 6) {
    resetError.value = '临时密码至少需要 6 个字符';
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
        <div class="family__grid">
          <AppField label="公历生日月份" for-id="member-birth-month" required>
            <AppSelect
              id="member-birth-month"
              v-model="createForm.birthMonth"
              :options="birthMonthOptions"
              placeholder="月份"
              aria-label="公历生日月份"
            />
          </AppField>
          <AppField label="公历生日日期" for-id="member-birth-day" required>
            <AppSelect
              id="member-birth-day"
              v-model="createForm.birthDay"
              :options="createBirthDays"
              placeholder="日期"
              aria-label="公历生日日期"
            />
          </AppField>
        </div>
        <AppField label="出生年份" for-id="member-birth-year" hint="不确定时可留空，只记录月日">
          <AppInput
            id="member-birth-year"
            v-model="createForm.birthYear"
            type="number"
            min="1900"
            :max="new Date().getFullYear()"
            placeholder="例如：1965"
          />
        </AppField>
        <AppCheckbox
          v-model="createForm.birthLunarEnabled"
          label="同时记录农历生日"
          description="农历按平月计算，闰月不单独区分"
        />
        <template v-if="createForm.birthLunarEnabled">
          <div class="family__grid">
            <AppField label="农历月份" for-id="member-lunar-month">
              <AppSelect
                id="member-lunar-month"
                v-model="createForm.birthLunarMonth"
                :options="birthMonthOptions"
                placeholder="月份"
                aria-label="农历生日月份"
              />
            </AppField>
            <AppField label="农历日期" for-id="member-lunar-day">
              <AppSelect
                id="member-lunar-day"
                v-model="createForm.birthLunarDay"
                :options="createLunarDays"
                placeholder="日期"
                aria-label="农历生日日期"
              />
            </AppField>
          </div>
          <AppField label="农历出生年份" for-id="member-lunar-year" hint="不确定时可留空">
            <AppInput
              id="member-lunar-year"
              v-model="createForm.birthLunarYear"
              type="number"
              min="1900"
              :max="new Date().getFullYear()"
              placeholder="例如：1965"
            />
          </AppField>
          <AppCheckbox
            v-model="createForm.birthLunarReminder"
            label="农历生日提醒"
            description="勾选后会在农历生日当天提醒"
          />
        </template>
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
        <AppField v-if="createForm.mode === 'manual'" label="临时密码" for-id="member-password" hint="至少 6 个字符，成员首次登录后需要修改">
          <div class="family__password">
            <AppInput
              id="member-password"
              v-model="createForm.password"
              :type="createShowPassword ? 'text' : 'password'"
              autocomplete="new-password"
            />
            <button
              type="button"
              class="family__reveal"
              :aria-label="createShowPassword ? '隐藏密码' : '显示密码'"
              :aria-pressed="createShowPassword"
              :title="createShowPassword ? '隐藏密码' : '显示密码'"
              @click="createShowPassword = !createShowPassword"
            >
              <AppIcon :name="createShowPassword ? 'eye-off' : 'eye'" :size="18" />
            </button>
          </div>
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
        <template v-if="auth.isAdmin">
          <div class="family__grid">
            <AppField label="公历生日月份" for-id="edit-birth-month">
              <AppSelect
                id="edit-birth-month"
                v-model="editForm.birthMonth"
                :options="birthMonthOptions"
                placeholder="月份"
                aria-label="公历生日月份"
              />
            </AppField>
            <AppField label="公历生日日期" for-id="edit-birth-day">
              <AppSelect
                id="edit-birth-day"
                v-model="editForm.birthDay"
                :options="editBirthDays"
                placeholder="日期"
                aria-label="公历生日日期"
              />
            </AppField>
          </div>
          <AppField label="出生年份" for-id="edit-birth-year" hint="不确定时可留空，只记录月日">
            <AppInput
              id="edit-birth-year"
              v-model="editForm.birthYear"
              type="number"
              min="1900"
              :max="new Date().getFullYear()"
              placeholder="例如：1965"
            />
          </AppField>
          <AppCheckbox
            v-model="editForm.birthLunarEnabled"
            label="同时记录农历生日"
            description="农历按平月计算，闰月不单独区分"
          />
          <template v-if="editForm.birthLunarEnabled">
            <div class="family__grid">
              <AppField label="农历月份" for-id="edit-lunar-month">
                <AppSelect
                  id="edit-lunar-month"
                  v-model="editForm.birthLunarMonth"
                  :options="birthMonthOptions"
                  placeholder="月份"
                  aria-label="农历生日月份"
                />
              </AppField>
              <AppField label="农历日期" for-id="edit-lunar-day">
                <AppSelect
                  id="edit-lunar-day"
                  v-model="editForm.birthLunarDay"
                  :options="editLunarDays"
                  placeholder="日期"
                  aria-label="农历生日日期"
                />
              </AppField>
            </div>
            <AppField label="农历出生年份" for-id="edit-lunar-year" hint="不确定时可留空">
              <AppInput
                id="edit-lunar-year"
                v-model="editForm.birthLunarYear"
                type="number"
                min="1900"
                :max="new Date().getFullYear()"
                placeholder="例如：1965"
              />
            </AppField>
            <AppCheckbox
              v-model="editForm.birthLunarReminder"
              label="农历生日提醒"
              description="勾选后会在农历生日当天提醒"
            />
          </template>
        </template>
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
        <AppField v-if="resetForm.mode === 'manual'" label="临时密码" for-id="reset-password" hint="至少 6 个字符">
          <AppInput id="reset-password" v-model="resetForm.password" type="password" autocomplete="new-password" />
        </AppField>
        <p v-if="resetError" class="family__error" role="alert"><AppIcon name="alert" :size="16" />{{ resetError }}</p>
      </form>
      <template #footer>
        <AppButton variant="ghost" @click="resetOpen = false">取消</AppButton>
        <AppButton :loading="resetting" @click="submitReset">重置密码</AppButton>
      </template>
    </AppDialog>

    <AvatarCropDialog v-model="editCropOpen" :file="editCropFile" @confirm="onEditAvatarCropped" />

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

.family__password {
  position: relative;
}

.family__password :deep(.at-input) {
  padding-right: 46px;
}

.family__reveal {
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

.family__reveal:hover {
  background: var(--at-bg-canvas-soft);
  color: var(--at-color-ink);
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
