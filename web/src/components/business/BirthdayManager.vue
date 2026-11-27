<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import type { Birthday, BirthdayDate } from '@/types';
import { ApiError } from '@/api/client';
import { useAuthStore } from '@/stores/auth';
import { useCalendarStore, type BirthdayDatePayload, type BirthdayPayload } from '@/stores/calendar';
import { useContactsStore } from '@/stores/contacts';
import { useFamilyStore } from '@/stores/family';
import { useToastStore } from '@/stores/toast';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppButton from '@/components/base/AppButton.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppField from '@/components/base/AppField.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppStatusTag from '@/components/base/AppStatusTag.vue';
import AppSwitch from '@/components/base/AppSwitch.vue';
import AppTextarea from '@/components/base/AppTextarea.vue';

const props = withDefaults(defineProps<{ openId?: number }>(), { openId: 0 });

const calendar = useCalendarStore();
const contacts = useContactsStore();
const family = useFamilyStore();
const auth = useAuthStore();
const toast = useToastStore();

const loading = ref(false);
const loadError = ref('');
const formOpen = ref(false);
const editing = ref<Birthday | null>(null);
const saving = ref(false);
const formError = ref('');
const deleteTarget = ref<Birthday | null>(null);
const deleting = ref(false);

const months = Array.from({ length: 12 }, (_, index) => ({ value: index + 1, label: `${index + 1} 月` }));
const reminderOptions = [
  { value: 0, label: '当天提醒' },
  { value: 1, label: '提前 1 天' },
  { value: 3, label: '提前 3 天' },
  { value: 7, label: '提前 1 周' },
];

const form = reactive({
  name: '',
  solarMonth: 1,
  solarDay: 1,
  solarYearUnknown: true,
  solarBirthYear: '',
  reminderDays: 0,
  lunarEnabled: false,
  lunarMonth: 1,
  lunarDay: 1,
  lunarYearUnknown: true,
  lunarBirthYear: '',
  lunarReminder: false,
  note: '',
  contactId: 0,
  userId: 0,
});

function daysInSolarMonth(month: number): number {
  if (month === 2) return 29;
  return new Date(2024, month, 0).getDate();
}
const solarDayOptions = computed(() =>
  Array.from({ length: daysInSolarMonth(form.solarMonth) }, (_, index) => ({ value: index + 1, label: `${index + 1} 日` })),
);
const lunarDayOptions = computed(() =>
  Array.from({ length: 30 }, (_, index) => ({ value: index + 1, label: `${index + 1} 日` })),
);
const contactOptions = computed(() => [
  { value: 0, label: '不关联' },
  ...contacts.items.map((contact) => ({
    value: contact.id,
    label: contact.organization ? `${contact.name}（${contact.organization}）` : contact.name,
  })),
]);
const memberOptions = computed(() => [
  { value: 0, label: '不关联' },
  ...family.members
    .filter((member) => member.status === 'active')
    .filter((member) => !calendar.birthdays.some((birthday) => birthday.userId === member.id && birthday.id !== editing.value?.id))
    .map((member) => ({ value: member.id, label: member.displayName })),
]);

async function load(): Promise<void> {
  loading.value = true;
  loadError.value = '';
  try {
    await Promise.all([
      calendar.loadBirthdays(true),
      contacts.load().catch(() => undefined),
      family.load().catch(() => undefined),
    ]);
  } catch (err) {
    loadError.value = err instanceof ApiError ? err.message : '生日加载失败';
  } finally {
    loading.value = false;
  }
  if (props.openId) openEditById(props.openId);
}

function resetForm(): void {
  Object.assign(form, {
    name: '', solarMonth: 1, solarDay: 1, solarYearUnknown: true, solarBirthYear: '',
    reminderDays: 0, lunarEnabled: false, lunarMonth: 1, lunarDay: 1, lunarYearUnknown: true,
    lunarBirthYear: '', lunarReminder: false, note: '', contactId: 0, userId: 0,
  });
}

function openCreate(): void {
  editing.value = null;
  resetForm();
  formError.value = '';
  formOpen.value = true;
}

function solarDateOf(birthday: Birthday): BirthdayDate | undefined {
  return birthday.dates.find((date) => date.calendar === 'solar');
}

function lunarDateOf(birthday: Birthday): BirthdayDate | undefined {
  return birthday.dates.find((date) => date.calendar === 'lunar');
}

function openEdit(birthday: Birthday): void {
  editing.value = birthday;
  resetForm();
  const solar = solarDateOf(birthday);
  if (solar) {
    form.solarMonth = solar.month;
    form.solarDay = solar.day;
    form.solarYearUnknown = solar.yearUnknown;
    form.solarBirthYear = solar.birthYear ? String(solar.birthYear) : '';
    form.reminderDays = offsetToDays(solar.reminderOffset ?? 0);
  }
  const lunar = lunarDateOf(birthday);
  if (lunar) {
    form.lunarEnabled = true;
    form.lunarMonth = lunar.month;
    form.lunarDay = lunar.day;
    form.lunarYearUnknown = lunar.yearUnknown;
    form.lunarBirthYear = lunar.birthYear ? String(lunar.birthYear) : '';
    form.lunarReminder = lunar.reminderOffset != null;
  }
  form.name = birthday.name;
  form.note = birthday.note ?? '';
  form.contactId = birthday.contactId ?? 0;
  form.userId = birthday.userId ?? 0;
  formError.value = '';
  formOpen.value = true;
}

function openEditById(id: number): void {
  const birthday = calendar.birthdays.find((item) => item.id === id);
  if (birthday) openEdit(birthday);
}

function offsetToDays(offset: number): number {
  if (offset >= 10080) return 7;
  if (offset >= 2880) return 3;
  if (offset >= 1440) return 1;
  return 0;
}

function daysToOffset(days: number): number {
  if (days === 7) return 10080;
  if (days === 3) return 2880;
  if (days === 1) return 1440;
  return 0;
}

async function submit(): Promise<void> {
  formError.value = '';
  if (!form.name.trim()) {
    formError.value = '请填写姓名';
    return;
  }
  const solarYear = form.solarYearUnknown ? null : Number(form.solarBirthYear);
  if (!form.solarYearUnknown && !solarYear) {
    formError.value = '请填写出生年份，或选择年份未知';
    return;
  }
  const dates: BirthdayDatePayload[] = [{
    calendar: 'solar',
    month: form.solarMonth,
    day: Math.min(form.solarDay, daysInSolarMonth(form.solarMonth)),
    yearUnknown: form.solarYearUnknown,
    birthYear: solarYear,
    reminderOffset: daysToOffset(form.reminderDays),
  }];
  if (form.lunarEnabled) {
    const lunarYear = form.lunarYearUnknown ? null : Number(form.lunarBirthYear);
    if (!form.lunarYearUnknown && !lunarYear) {
      formError.value = '请填写农历出生年份，或选择年份未知';
      return;
    }
    dates.push({
      calendar: 'lunar',
      month: form.lunarMonth,
      day: form.lunarDay,
      yearUnknown: form.lunarYearUnknown,
      birthYear: lunarYear,
      reminderOffset: form.lunarReminder ? 0 : null,
    });
  }
  const payload: BirthdayPayload = {
    name: form.name.trim(),
    note: form.note.trim() || null,
    contactId: form.contactId || null,
    userId: form.userId || null,
    dates,
  };
  saving.value = true;
  try {
    if (editing.value) {
      await calendar.updateBirthday(editing.value.id, payload);
      toast.success('生日已更新');
    } else {
      await calendar.createBirthday(payload);
      toast.success('生日已添加');
    }
    formOpen.value = false;
  } catch (err) {
    formError.value = err instanceof ApiError ? err.message : '保存生日失败';
  } finally {
    saving.value = false;
  }
}

async function confirmDelete(): Promise<void> {
  if (!deleteTarget.value) return;
  deleting.value = true;
  try {
    await calendar.deleteBirthday(deleteTarget.value.id);
    toast.success('生日已删除');
    deleteTarget.value = null;
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除生日失败');
  } finally {
    deleting.value = false;
  }
}

function daysText(birthday: Birthday): string {
  if (birthday.daysUntil === 0) return '今天';
  if (birthday.daysUntil === 1) return '明天';
  if (birthday.daysUntil != null) return `${birthday.daysUntil} 天后`;
  return '';
}

function linkedContact(birthday: Birthday): string {
  const contact = contacts.items.find((item) => item.id === birthday.contactId);
  return contact ? contact.name : '';
}

function linkedMember(birthday: Birthday): string {
  const member = family.members.find((item) => item.id === birthday.userId);
  return member ? member.displayName : '';
}

function canEdit(birthday: Birthday): boolean {
  return birthday.createdBy === auth.user?.id || auth.isAdmin;
}

watch(() => props.openId, (id) => {
  if (id) openEditById(id);
});
watch(() => form.solarMonth, () => {
  const last = daysInSolarMonth(form.solarMonth);
  if (form.solarDay > last) form.solarDay = last;
});

onMounted(load);
</script>

<template>
  <section class="birthdays">
    <div class="birthdays__bar">
      <p class="birthdays__hint">公历生日会同步到日历并必定提醒；农历生日按每年换算后的公历日期展示，可选择是否提醒。</p>
      <AppButton variant="outline" @click="openCreate"><AppIcon name="plus" :size="16" />添加生日</AppButton>
    </div>

    <p v-if="loadError" class="birthdays__error" role="alert">{{ loadError }}</p>
    <div v-else-if="loading && !calendar.birthdays.length" class="birthdays__loading">正在加载生日…</div>
    <AppEmptyState
      v-else-if="!calendar.birthdays.length"
      icon="calendar"
      title="还没有生日记录"
      description="添加生日后，日历会每年自动显示，并可以提前提醒。"
    >
      <AppButton variant="outline" @click="openCreate">添加第一个生日</AppButton>
    </AppEmptyState>
    <ul v-else class="birthdays__list">
      <li v-for="birthday in calendar.birthdays" :key="birthday.id" class="birthday at-card">
        <AppAvatar :name="birthday.name" :size="42" />
        <div class="birthday__main">
          <div class="birthday__name-row">
            <span class="birthday__name">{{ birthday.name }}</span>
            <AppStatusTag v-if="birthday.daysUntil === 0" tone="warning">今天</AppStatusTag>
            <AppStatusTag v-else tone="neutral">{{ daysText(birthday) }}</AppStatusTag>
          </div>
          <p v-if="solarDateOf(birthday)" class="birthday__date">
            公历 {{ solarDateOf(birthday)?.month }} 月 {{ solarDateOf(birthday)?.day }} 日
            <span v-if="birthday.nextDate">· 下次 {{ birthday.nextDate }}</span>
            <span v-if="birthday.age != null">· 即将 {{ birthday.age }} 岁</span>
            <span v-else-if="solarDateOf(birthday)?.yearUnknown">· 年份未知</span>
          </p>
          <p v-if="lunarDateOf(birthday)" class="birthday__date">
            农历 {{ lunarDateOf(birthday)?.month }} 月 {{ lunarDateOf(birthday)?.day }} 日
            <span v-if="lunarDateOf(birthday)?.nextDate">· 下次 {{ lunarDateOf(birthday)?.nextDate }}</span>
            <span v-if="lunarDateOf(birthday)?.reminderOffset != null">· 农历提醒</span>
          </p>
          <p v-if="birthday.note" class="birthday__note">{{ birthday.note }}</p>
          <p v-if="linkedContact(birthday) || linkedMember(birthday)" class="birthday__links">
            <span v-if="linkedMember(birthday)">成员：{{ linkedMember(birthday) }}</span>
            <span v-if="linkedContact(birthday)">联系人：{{ linkedContact(birthday) }}</span>
          </p>
        </div>
        <div v-if="canEdit(birthday)" class="birthday__actions">
          <AppIconButton icon="edit" label="编辑生日" size="sm" @click="openEdit(birthday)" />
          <AppIconButton icon="trash" label="删除生日" size="sm" @click="deleteTarget = birthday" />
        </div>
      </li>
    </ul>

    <AppDialog v-model="formOpen" :title="editing ? '编辑生日' : '添加生日'" size="md">
      <form class="at-stack" @submit.prevent="submit">
        <AppField label="姓名" for-id="birthday-name" required>
          <AppInput id="birthday-name" v-model="form.name" placeholder="例如：外婆" :maxlength="80" />
        </AppField>

        <p class="birthdays__section">公历生日（必填，必定提醒）</p>
        <div class="birthdays__grid">
          <AppField label="月份" for-id="birthday-month">
            <select id="birthday-month" v-model.number="form.solarMonth" class="birthdays__select" aria-label="公历生日月份">
              <option v-for="month in months" :key="month.value" :value="month.value">{{ month.label }}</option>
            </select>
          </AppField>
          <AppField label="日期" for-id="birthday-day">
            <select id="birthday-day" v-model.number="form.solarDay" class="birthdays__select" aria-label="公历生日日期">
              <option v-for="day in solarDayOptions" :key="day.value" :value="day.value">{{ day.label }}</option>
            </select>
          </AppField>
        </div>
        <AppSwitch v-model="form.solarYearUnknown" label="年份未知" description="只记录月日，不计算年龄" />
        <AppField v-if="!form.solarYearUnknown" label="出生年份" for-id="birthday-year">
          <AppInput id="birthday-year" v-model="form.solarBirthYear" type="number" min="1900" :max="new Date().getFullYear()" />
        </AppField>
        <AppField label="提前提醒" for-id="birthday-reminder">
          <select id="birthday-reminder" v-model.number="form.reminderDays" class="birthdays__select" aria-label="提前提醒">
            <option v-for="option in reminderOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </AppField>

        <AppSwitch
          v-model="form.lunarEnabled"
          label="记录农历生日"
          description="农历按平月计算，闰月不单独区分"
        />
        <template v-if="form.lunarEnabled">
          <div class="birthdays__grid">
            <AppField label="农历月份" for-id="birthday-lunar-month">
              <select id="birthday-lunar-month" v-model.number="form.lunarMonth" class="birthdays__select" aria-label="农历生日月份">
                <option v-for="month in months" :key="month.value" :value="month.value">{{ month.label }}</option>
              </select>
            </AppField>
            <AppField label="农历日期" for-id="birthday-lunar-day">
              <select id="birthday-lunar-day" v-model.number="form.lunarDay" class="birthdays__select" aria-label="农历生日日期">
                <option v-for="day in lunarDayOptions" :key="day.value" :value="day.value">{{ day.label }}</option>
              </select>
            </AppField>
          </div>
          <AppSwitch v-model="form.lunarYearUnknown" label="农历年份未知" description="只记录农历月日，不计算农历周岁" />
          <AppField v-if="!form.lunarYearUnknown" label="农历出生年份" for-id="birthday-lunar-year">
            <AppInput id="birthday-lunar-year" v-model="form.lunarBirthYear" type="number" min="1900" :max="new Date().getFullYear()" />
          </AppField>
          <AppSwitch v-model="form.lunarReminder" label="农历生日提醒" description="勾选后会在农历生日当天提醒" />
        </template>

        <div class="birthdays__grid">
          <AppField label="关联成员" for-id="birthday-member">
            <select id="birthday-member" v-model.number="form.userId" class="birthdays__select" aria-label="关联成员">
              <option v-for="member in memberOptions" :key="member.value" :value="member.value">{{ member.label }}</option>
            </select>
          </AppField>
          <AppField label="关联联系人" for-id="birthday-contact">
            <select id="birthday-contact" v-model.number="form.contactId" class="birthdays__select" aria-label="关联联系人">
              <option v-for="contact in contactOptions" :key="contact.value" :value="contact.value">{{ contact.label }}</option>
            </select>
          </AppField>
        </div>
        <AppField label="备注" for-id="birthday-note">
          <AppTextarea id="birthday-note" v-model="form.note" :rows="3" placeholder="例如：喜欢兰花" :maxlength="1000" />
        </AppField>
        <p v-if="formError" class="birthdays__error" role="alert"><AppIcon name="alert" :size="16" />{{ formError }}</p>
      </form>
      <template #footer>
        <AppButton variant="ghost" @click="formOpen = false">取消</AppButton>
        <AppButton :loading="saving" @click="submit">保存</AppButton>
      </template>
    </AppDialog>

    <AppConfirmDialog
      :model-value="deleteTarget !== null"
      title="删除生日"
      :message="`确定删除「${deleteTarget?.name}」的生日吗？日历中的生日事件和提醒会一并移除。`"
      danger
      confirm-text="删除"
      :loading="deleting"
      @update:model-value="deleteTarget = null"
      @confirm="confirmDelete"
    />
  </section>
</template>

<style scoped>
.birthdays {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.birthdays__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-4);
  flex-wrap: wrap;
}

.birthdays__hint {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.birthdays__loading {
  padding: var(--at-space-6) 0;
  color: var(--at-text-secondary);
  font-size: var(--at-text-body-sm);
}

.birthdays__list {
  list-style: none;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--at-space-3);
}

.birthday {
  display: flex;
  align-items: flex-start;
  gap: var(--at-space-3);
  padding: var(--at-space-4);
  border-radius: var(--at-radius-xl);
}

.birthday__main {
  flex: 1;
  min-width: 0;
}

.birthday__name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.birthday__name {
  font-size: var(--at-text-body);
  font-weight: 500;
}

.birthday__date {
  margin-top: 3px;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.birthday__note {
  margin-top: var(--at-space-2);
  font-size: var(--at-text-micro);
  color: var(--at-text-secondary);
  line-height: 1.6;
}

.birthday__links {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
  margin-top: var(--at-space-2);
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.birthday__actions {
  display: flex;
  gap: 2px;
}

.birthdays__section {
  font-size: var(--at-text-caption);
  font-weight: 500;
  color: var(--at-text-secondary);
}

.birthdays__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--at-space-3);
}

.birthdays__select {
  width: 100%;
  min-height: var(--at-touch-min);
  padding: 0 var(--at-space-3);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface);
  color: var(--at-text-primary);
  font-size: var(--at-text-body-sm);
}

.birthdays__error {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--at-color-danger);
  font-size: 13px;
}

@media (max-width: 1023px) {
  .birthdays__list {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 639px) {
  .birthdays__bar {
    flex-direction: column;
    align-items: stretch;
  }

  .birthdays__grid {
    grid-template-columns: 1fr;
  }
}
</style>
