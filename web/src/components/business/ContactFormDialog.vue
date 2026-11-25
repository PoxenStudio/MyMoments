<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import AppButton from '@/components/base/AppButton.vue';
import AppCheckbox from '@/components/base/AppCheckbox.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppField from '@/components/base/AppField.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import AppSwitch from '@/components/base/AppSwitch.vue';
import AppAvatar from '@/components/base/AppAvatar.vue';
import { ApiError } from '@/api/client';
import { useContactsStore, type ContactPayload } from '@/stores/contacts';
import { useToastStore } from '@/stores/toast';
import type { Contact, ContactItem, ContactReminder } from '@/types';
import { avatarUrl, uploadContactAvatar } from '@/utils/avatar';
import { taskColors } from '@/utils/task';

const props = defineProps<{
  modelValue: boolean;
  contact: Contact | null;
}>();

const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [contact: Contact] }>();

const store = useContactsStore();
const toast = useToastStore();

const phoneLabels = ['私人', '办公', '住宅'];
const saving = ref(false);
const avatarUploading = ref(false);

interface PhoneDraft {
  value: string;
  label: string;
  isPrimaryCall: boolean;
  isPrimarySms: boolean;
}
interface CustomDraft {
  title: string;
  frequency: '' | 'monthly' | 'weekly' | 'once';
  dueDate: string;
  color: string;
}

const form = reactive({
  name: '',
  organization: '',
  title: '',
  note: '',
  birthdayMonth: '',
  birthdayDay: '',
  birthYear: '',
  showInCalendar: false,
  wechat: '',
  officePhone: '',
  avatarKey: null as string | null,
  phones: [] as PhoneDraft[],
  emails: [] as string[],
  address: '',
  relations: [] as number[],
  springFestival: false,
  traditional: false,
  western: false,
  customs: [] as CustomDraft[],
});

const editing = computed(() => props.contact !== null);
const availableContacts = computed(() => store.items.filter((c) => c.id !== props.contact?.id));
const hasFullBirthday = computed(() => form.birthdayMonth !== '' && form.birthdayDay !== '');

const relationQuery = ref('');
const relationOpen = ref(false);
const relationActive = ref(0);
const relationWrap = ref<HTMLElement | null>(null);
const relationInput = ref<HTMLInputElement | null>(null);

const relationCandidates = computed(() => {
  const query = relationQuery.value.trim().toLowerCase();
  if (!query) return [];
  return availableContacts.value
    .filter((c) => !form.relations.includes(c.id) && c.name.toLowerCase().startsWith(query))
    .slice(0, 8);
});

function relationName(id: number): string {
  return store.items.find((c) => c.id === id)?.name ?? `联系人 #${id}`;
}

function onRelationInput(): void {
  relationOpen.value = relationQuery.value.trim().length > 0;
  relationActive.value = 0;
}

function addRelation(id: number): void {
  if (!form.relations.includes(id)) form.relations.push(id);
  relationQuery.value = '';
  relationOpen.value = false;
  relationActive.value = 0;
  relationInput.value?.focus();
}

function removeRelation(id: number): void {
  form.relations = form.relations.filter((v) => v !== id);
}

function onRelationKeydown(event: KeyboardEvent): void {
  if (event.key === 'ArrowDown') {
    event.preventDefault();
    if (!relationOpen.value) onRelationInput();
    if (!relationCandidates.value.length) {
      relationActive.value = 0;
      return;
    }
    relationActive.value = Math.min(relationActive.value + 1, relationCandidates.value.length - 1);
  } else if (event.key === 'ArrowUp') {
    event.preventDefault();
    relationActive.value = Math.max(relationActive.value - 1, 0);
  } else if (event.key === 'Enter') {
    const candidate = relationCandidates.value[relationActive.value];
    if (relationOpen.value && candidate) {
      event.preventDefault();
      addRelation(candidate.id);
    }
  } else if (event.key === 'Escape') {
    relationOpen.value = false;
  } else if (event.key === 'Backspace' && !relationQuery.value && form.relations.length) {
    form.relations = form.relations.slice(0, -1);
  }
}

function onRelationFocusOut(event: FocusEvent): void {
  if (!relationWrap.value?.contains(event.relatedTarget as Node | null)) {
    relationOpen.value = false;
  }
}

function resetForm(): void {
  const c = props.contact;
  form.name = c?.name ?? '';
  form.organization = c?.organization ?? '';
  form.title = c?.title ?? '';
  form.note = c?.note ?? '';
  form.birthdayMonth = c?.birthdayMonth != null ? String(c.birthdayMonth) : '';
  form.birthdayDay = c?.birthdayDay != null ? String(c.birthdayDay) : '';
  form.birthYear = c?.birthYear != null ? String(c.birthYear) : '';
  form.showInCalendar = c?.showInCalendar ?? false;
  form.wechat = c?.wechat ?? '';
  form.officePhone = c?.officePhone ?? '';
  form.avatarKey = c?.avatarKey ?? null;
  form.address = c?.items.find((i) => i.kind === 'address')?.value ?? '';
  form.emails = (c?.items ?? []).filter((i) => i.kind === 'email').map((i) => i.value);
  form.phones = (c?.items ?? [])
    .filter((i) => i.kind === 'phone')
    .map((i) => ({ value: i.value, label: i.label ?? '私人', isPrimaryCall: !!i.isPrimaryCall, isPrimarySms: !!i.isPrimarySms }));
  form.relations = (c?.relations ?? [])
    .map((r) => r.relatedContactId)
    .filter((id) => id !== c?.id);
  form.springFestival = (c?.reminders ?? []).some((r) => r.kind === 'spring_festival' && r.enabled);
  form.traditional = (c?.reminders ?? []).some((r) => r.kind === 'traditional' && r.enabled);
  form.western = (c?.reminders ?? []).some((r) => r.kind === 'western' && r.enabled);
  form.customs = (c?.reminders ?? [])
    .filter((r) => r.kind === 'custom')
    .map((r) => ({ title: r.title ?? '', frequency: r.frequency, dueDate: r.dueDate ?? '', color: r.color ?? taskColors[0] }));
  relationQuery.value = '';
  relationOpen.value = false;
  relationActive.value = 0;
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm();
  },
);

async function onAvatarChange(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  avatarUploading.value = true;
  try {
    form.avatarKey = await uploadContactAvatar(file);
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '头像上传失败');
  } finally {
    avatarUploading.value = false;
    input.value = '';
  }
}

function addPhone(): void {
  form.phones.push({ value: '', label: '私人', isPrimaryCall: false, isPrimarySms: false });
}
function removePhone(index: number): void {
  form.phones.splice(index, 1);
}
function setPrimaryCall(index: number): void {
  form.phones.forEach((p, i) => (p.isPrimaryCall = i === index));
}
function setPrimarySms(index: number): void {
  form.phones.forEach((p, i) => (p.isPrimarySms = i === index));
}
function addEmail(): void {
  form.emails.push('');
}
function removeEmail(index: number): void {
  form.emails.splice(index, 1);
}
function addCustom(): void {
  form.customs.push({ title: '', frequency: 'once', dueDate: '', color: taskColors[0] });
}
function removeCustom(index: number): void {
  form.customs.splice(index, 1);
}

function buildPayload(): ContactPayload {
  const items: ContactItem[] = [];
  for (const p of form.phones) {
    if (!p.value.trim()) continue;
    items.push({ kind: 'phone', label: p.label, value: p.value.trim(), isPrimaryCall: p.isPrimaryCall, isPrimarySms: p.isPrimarySms });
  }
  for (const e of form.emails) {
    if (!e.trim()) continue;
    items.push({ kind: 'email', label: '邮箱', value: e.trim() });
  }
  if (form.address.trim()) {
    items.push({ kind: 'address', label: '地址', value: form.address.trim() });
  }
  const reminders: ContactReminder[] = [
    { kind: 'spring_festival', frequency: '', dueDate: null, enabled: form.springFestival },
    { kind: 'traditional', frequency: '', dueDate: null, enabled: form.traditional },
    { kind: 'western', frequency: '', dueDate: null, enabled: form.western },
  ];
  for (const c of form.customs) {
    reminders.push({ kind: 'custom', title: c.title, frequency: c.frequency, dueDate: c.dueDate || null, color: c.color, enabled: true });
  }
  return {
    name: form.name,
    organization: form.organization || null,
    title: form.title || null,
    note: form.note || null,
    birthdayMonth: form.birthdayMonth ? Number(form.birthdayMonth) : null,
    birthdayDay: form.birthdayDay ? Number(form.birthdayDay) : null,
    birthYear: form.birthYear ? Number(form.birthYear) : null,
    showInCalendar: form.showInCalendar && hasFullBirthday.value,
    wechat: form.wechat || null,
    officePhone: form.officePhone || null,
    avatarKey: form.avatarKey,
    items,
    relations: form.relations.map((id) => ({ relatedContactId: id, relation: 'emergency' })),
    reminders,
  };
}

async function save(): Promise<void> {
  if (!form.name.trim()) {
    toast.error('请填写联系人姓名');
    return;
  }
  const month = form.birthdayMonth ? Number(form.birthdayMonth) : null;
  const day = form.birthdayDay ? Number(form.birthdayDay) : null;
  if ((month === null) !== (day === null)) {
    toast.error('生日月份与日期需同时填写');
    return;
  }
  saving.value = true;
  try {
    const payload = buildPayload();
    const saved = editing.value && props.contact ? await store.update(props.contact.id, payload) : await store.create(payload);
    toast.success(editing.value ? '联系人已更新' : '联系人已创建');
    emit('saved', saved);
    emit('update:modelValue', false);
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '保存失败，请稍后重试');
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <AppDialog :model-value="modelValue" :title="editing ? '编辑联系人' : '新增联系人'" size="lg" @update:model-value="emit('update:modelValue', $event)">
    <div class="cfd">
      <div class="cfd__head">
        <AppAvatar :name="form.name || '?'" :size="64" :src="avatarUrl(form.avatarKey)" />
        <div class="cfd__head-actions">
          <label class="cfd__upload">
            <AppButton variant="outline" size="sm" :loading="avatarUploading" @click.prevent>
              {{ avatarUploading ? '上传中…' : '更换头像' }}
            </AppButton>
            <input type="file" accept="image/*" hidden @change="onAvatarChange" />
          </label>
        </div>
      </div>

      <AppField label="姓名" required>
        <AppInput v-model="form.name" placeholder="联系人姓名" :maxlength="80" />
      </AppField>

      <div class="cfd__grid">
        <AppField label="单位 / 组织">
          <AppInput v-model="form.organization" placeholder="可留空" />
        </AppField>
        <AppField label="称谓 / 职位">
          <AppInput v-model="form.title" placeholder="可留空" />
        </AppField>
      </div>

      <div class="cfd__grid">
        <AppField label="微信号">
          <AppInput v-model="form.wechat" placeholder="可留空" />
        </AppField>
        <AppField label="办公电话">
          <AppInput v-model="form.officePhone" inputmode="tel" placeholder="可留空" />
        </AppField>
      </div>

      <fieldset class="cfd__section">
        <legend>生日</legend>
        <div class="cfd__birthday">
          <AppField label="月">
            <AppInput v-model="form.birthdayMonth" type="number" min="1" max="12" placeholder="月" inputmode="numeric" />
          </AppField>
          <AppField label="日">
            <AppInput v-model="form.birthdayDay" type="number" min="1" max="31" placeholder="日" inputmode="numeric" />
          </AppField>
          <AppField label="年份（可留空）">
            <AppInput v-model="form.birthYear" type="number" min="1900" :max="new Date().getFullYear()" placeholder="可留空" inputmode="numeric" />
          </AppField>
        </div>
      </fieldset>

      <fieldset class="cfd__section">
        <legend>手机号</legend>
        <div v-for="(p, i) in form.phones" :key="i" class="cfd__phone">
          <AppInput v-model="p.value" placeholder="手机号" inputmode="tel" />
          <AppSelect v-model="p.label" :options="phoneLabels.map((l) => ({ value: l, label: l }))" aria-label="号码类型" />
          <AppIconButton icon="close" label="删除号码" size="sm" @click="removePhone(i)" />
          <div class="cfd__phone-flags">
            <AppCheckbox :model-value="p.isPrimaryCall" label="默认联系号" @update:model-value="setPrimaryCall(i)" />
            <AppCheckbox :model-value="p.isPrimarySms" label="默认短信号" @update:model-value="setPrimarySms(i)" />
          </div>
        </div>
        <AppButton variant="ghost" size="sm" @click="addPhone">+ 添加手机号</AppButton>
      </fieldset>

      <fieldset class="cfd__section">
        <legend>邮箱</legend>
        <div v-for="(_, i) in form.emails" :key="i" class="cfd__row">
          <AppInput v-model="form.emails[i]" type="email" placeholder="邮箱地址" inputmode="email" />
          <AppIconButton icon="close" label="删除邮箱" size="sm" @click="removeEmail(i)" />
        </div>
        <AppButton variant="ghost" size="sm" @click="addEmail">+ 添加邮箱</AppButton>
      </fieldset>

      <AppField label="通信地址">
        <AppInput v-model="form.address" placeholder="可留空" />
      </AppField>

      <fieldset class="cfd__section">
        <legend>关联联系人</legend>
        <p v-if="!availableContacts.length" class="cfd__hint">暂无其他联系人可关联。</p>
        <div v-else ref="relationWrap" class="cfd__combo" @focusout="onRelationFocusOut">
          <div class="cfd__combo-box">
            <span v-for="id in form.relations" :key="id" class="cfd__chip">
              {{ relationName(id) }}
              <button
                type="button"
                class="cfd__chip-remove"
                :aria-label="`移除 ${relationName(id)}`"
                @click="removeRelation(id)"
              >
                <AppIcon name="close" :size="12" />
              </button>
            </span>
            <input
              ref="relationInput"
              v-model="relationQuery"
              class="cfd__combo-input"
              type="text"
              placeholder="输入姓名前缀搜索"
              autocomplete="off"
              @input="onRelationInput"
              @focus="onRelationInput"
              @keydown="onRelationKeydown"
            />
          </div>
          <ul v-if="relationOpen" class="cfd__combo-list" role="listbox">
            <li
              v-for="(c, i) in relationCandidates"
              :key="c.id"
              class="cfd__combo-option"
              :class="{ 'is-active': i === relationActive }"
              role="option"
              :aria-selected="i === relationActive"
              @mousedown.prevent="addRelation(c.id)"
              @mouseenter="relationActive = i"
            >
              {{ c.name }}
            </li>
            <li v-if="!relationCandidates.length" class="cfd__combo-empty">无匹配联系人</li>
          </ul>
        </div>
      </fieldset>

      <fieldset class="cfd__section">
        <legend>问候与提醒</legend>
        <AppSwitch
          v-model="form.showInCalendar"
          label="生日祝福"
          :description="hasFullBirthday ? '生日当天显示在家庭日历与首页生日列表' : '先填写上方生日月份与日期'"
          :disabled="!hasFullBirthday"
        />
        <AppSwitch v-model="form.springFestival" label="春节拜年" description="每年春节生成拜年待办" />
        <AppSwitch v-model="form.traditional" label="重要传统节日问候" description="元宵、清明、端午、中秋、冬至" />
        <AppSwitch v-model="form.western" label="重要西方节日问候" description="元旦、情人节、圣诞节" />

        <div class="cfd__customs">
          <p class="cfd__subtitle">其它事项</p>
          <div v-for="(c, i) in form.customs" :key="i" class="cfd__custom">
            <AppInput v-model="c.title" placeholder="待办事项（如：提醒体检）" />
            <div class="cfd__custom-row">
              <AppSelect
                v-model="c.frequency"
                :options="[
                  { value: 'once', label: '指定日期' },
                  { value: 'monthly', label: '每月' },
                  { value: 'weekly', label: '每周' },
                ]"
                aria-label="频率"
              />
              <AppInput v-model="c.dueDate" type="date" aria-label="日期" />
              <AppIconButton icon="close" label="删除事项" size="sm" @click="removeCustom(i)" />
            </div>
            <div class="cfd__colors" role="group" aria-label="待办颜色">
              <button
                v-for="color in taskColors"
                :key="color"
                type="button"
                class="cfd__swatch"
                :class="{ 'is-active': c.color === color }"
                :style="{ background: color }"
                :aria-label="`颜色 ${color}`"
                @click="c.color = color"
              />
            </div>
          </div>
          <AppButton variant="ghost" size="sm" @click="addCustom">+ 添加其它事项</AppButton>
        </div>
      </fieldset>

      <AppField label="备注">
        <textarea v-model="form.note" class="cfd__note" rows="3" placeholder="补充说明" maxlength="2000" />
      </AppField>
    </div>

    <template #footer>
      <AppButton variant="ghost" @click="emit('update:modelValue', false)">取消</AppButton>
      <AppButton :loading="saving" @click="save">{{ editing ? '保存' : '创建' }}</AppButton>
    </template>
  </AppDialog>
</template>

<style scoped>
.cfd {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}
.cfd__head {
  display: flex;
  align-items: center;
  gap: var(--at-space-4);
  margin-bottom: var(--at-space-2);
}
.cfd__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--at-space-3);
}
.cfd__section {
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-lg);
  padding: var(--at-space-4);
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
}
.cfd__section legend {
  padding: 0 var(--at-space-2);
  font-size: var(--at-text-caption);
  font-weight: 500;
  color: var(--at-text-secondary);
}
.cfd__birthday {
  display: grid;
  grid-template-columns: 1fr 1fr 1.6fr;
  gap: var(--at-space-3);
}
.cfd__phone {
  display: grid;
  grid-template-columns: 1fr 110px 36px;
  gap: var(--at-space-2);
  align-items: center;
}
.cfd__phone-flags {
  grid-column: 1 / -1;
  display: flex;
  gap: var(--at-space-4);
}
.cfd__row {
  display: grid;
  grid-template-columns: 1fr 36px;
  gap: var(--at-space-2);
  align-items: center;
}
.cfd__combo {
  position: relative;
}
.cfd__combo-box {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--at-space-2);
  min-height: 44px;
  padding: var(--at-space-2) var(--at-space-3);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  transition: border-color var(--at-duration-fast) var(--at-ease), box-shadow var(--at-duration-fast) var(--at-ease);
}
.cfd__combo-box:focus-within {
  border-color: var(--at-color-ink);
  box-shadow: 0 0 0 1px var(--at-color-ink);
}
.cfd__chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 4px 3px 10px;
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface-strong);
  color: var(--at-text-primary);
  font-size: var(--at-text-micro);
}
.cfd__chip-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px;
  border: none;
  background: none;
  color: var(--at-color-muted);
  cursor: pointer;
}
.cfd__chip-remove:hover {
  color: var(--at-text-primary);
}
.cfd__combo-input {
  flex: 1;
  min-width: 140px;
  height: 28px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--at-text-primary);
  font-size: var(--at-text-body-sm);
}
.cfd__combo-input:focus {
  outline: none;
}
.cfd__combo-input::placeholder {
  color: var(--at-color-muted-soft);
}
.cfd__combo-list {
  position: absolute;
  z-index: 20;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  max-height: 220px;
  margin: 0;
  padding: var(--at-space-1) 0;
  overflow-y: auto;
  list-style: none;
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  box-shadow: var(--at-shadow-sm);
}
.cfd__combo-option {
  padding: var(--at-space-2) var(--at-space-4);
  color: var(--at-text-primary);
  font-size: var(--at-text-body-sm);
  cursor: pointer;
}
.cfd__combo-option.is-active {
  background: var(--at-bg-surface-strong);
}
.cfd__combo-empty {
  padding: var(--at-space-2) var(--at-space-4);
  color: var(--at-color-muted);
  font-size: var(--at-text-micro);
}
.cfd__hint {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}
.cfd__subtitle {
  font-size: var(--at-text-caption);
  font-weight: 500;
  color: var(--at-text-secondary);
}
.cfd__custom {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  padding: var(--at-space-3);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
}
.cfd__custom-row {
  display: grid;
  grid-template-columns: 120px 1fr 36px;
  gap: var(--at-space-2);
  align-items: center;
}
.cfd__colors {
  display: flex;
  gap: var(--at-space-2);
}
.cfd__swatch {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
}
.cfd__swatch.is-active {
  border-color: var(--at-color-ink);
  box-shadow: 0 0 0 2px var(--at-bg-surface);
}
.cfd__note {
  width: 100%;
  padding: var(--at-space-3) var(--at-space-4);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  color: var(--at-text-primary);
  font-size: var(--at-text-body-sm);
  line-height: 1.55;
  resize: vertical;
}
.cfd__note:focus {
  outline: none;
  border-color: var(--at-color-ink);
}
@media (max-width: 639px) {
  .cfd__grid,
  .cfd__birthday {
    grid-template-columns: 1fr;
  }
}
</style>
