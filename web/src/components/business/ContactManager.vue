<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import type { Contact } from '@/types';
import { ApiError } from '@/api/client';
import { useAuthStore } from '@/stores/auth';
import { useContactsStore } from '@/stores/contacts';
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
import AppSelect from '@/components/base/AppSelect.vue';
import AppStatusTag from '@/components/base/AppStatusTag.vue';
import AppTextarea from '@/components/base/AppTextarea.vue';

const contacts = useContactsStore();
const auth = useAuthStore();
const toast = useToastStore();

const query = ref('');
const loadError = ref('');
const formOpen = ref(false);
const editing = ref<Contact | null>(null);
const saving = ref(false);
const formError = ref('');
const deleteTarget = ref<Contact | null>(null);
const deleting = ref(false);

interface ItemForm {
  kind: 'phone' | 'email' | 'address';
  label: string;
  value: string;
}

const form = reactive({
  name: '',
  organization: '',
  title: '',
  category: '',
  birthdayOn: '',
  ownerScope: 'household' as 'household' | 'private',
  note: '',
  items: [] as ItemForm[],
});

const kindOptions = [
  { value: 'phone', label: '电话' },
  { value: 'email', label: '邮箱' },
  { value: 'address', label: '地址' },
];

const scopeOptions = [
  { value: 'household', label: '全家可见' },
  { value: 'private', label: '仅自己可见' },
];

const categories = computed(() => {
  const set = new Set<string>();
  for (const item of contacts.items) {
    if (item.category) set.add(item.category);
  }
  return [...set].sort((a, b) => a.localeCompare(b, 'zh-CN'));
});

const filteredContacts = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return contacts.items;
  return contacts.items.filter((contact) => {
    const haystack = [contact.name, contact.organization, contact.title, contact.category, contact.note]
      .filter(Boolean)
      .join(' ')
      .toLowerCase();
    return haystack.includes(q) || contact.items.some((item) => item.value.toLowerCase().includes(q));
  });
});

function canDelete(contact: Contact): boolean {
  return contact.createdBy === auth.user?.id || auth.isAdmin;
}

async function load(): Promise<void> {
  loadError.value = '';
  try {
    await contacts.load({}, true);
  } catch (err) {
    loadError.value = err instanceof ApiError ? err.message : '通讯录加载失败';
  }
}

function openCreate(): void {
  editing.value = null;
  Object.assign(form, {
    name: '', organization: '', title: '', category: '', birthdayOn: '',
    ownerScope: 'household', note: '', items: [] as ItemForm[],
  });
  formError.value = '';
  formOpen.value = true;
}

function openEdit(contact: Contact): void {
  editing.value = contact;
  Object.assign(form, {
    name: contact.name,
    organization: contact.organization ?? '',
    title: contact.title ?? '',
    category: contact.category ?? '',
    birthdayOn: contact.birthdayOn ?? '',
    ownerScope: contact.ownerScope,
    note: contact.note ?? '',
    items: contact.items.map((item) => ({ kind: item.kind as ItemForm['kind'], label: item.label ?? '', value: item.value })),
  });
  formError.value = '';
  formOpen.value = true;
}

function addItem(kind: ItemForm['kind'] = 'phone'): void {
  form.items.push({ kind, label: '', value: '' });
}

function removeItem(index: number): void {
  form.items.splice(index, 1);
}

async function submit(): Promise<void> {
  formError.value = '';
  if (!form.name.trim()) {
    formError.value = '请填写姓名';
    return;
  }
  const items = form.items
    .filter((item) => item.value.trim())
    .map((item) => ({ kind: item.kind, label: item.label.trim() || null, value: item.value.trim() }));
  saving.value = true;
  try {
    const payload = {
      name: form.name.trim(),
      organization: form.organization.trim() || null,
      title: form.title.trim() || null,
      category: form.category.trim() || null,
      birthdayOn: form.birthdayOn || null,
      ownerScope: form.ownerScope,
      note: form.note.trim() || null,
      items,
    };
    if (editing.value) {
      await contacts.update(editing.value.id, payload);
      toast.success('联系人已更新');
    } else {
      await contacts.create(payload);
      toast.success('联系人已添加');
    }
    formOpen.value = false;
  } catch (err) {
    formError.value = err instanceof ApiError ? err.message : '保存联系人失败';
  } finally {
    saving.value = false;
  }
}

async function confirmDelete(): Promise<void> {
  if (!deleteTarget.value) return;
  deleting.value = true;
  try {
    await contacts.remove(deleteTarget.value.id);
    toast.success('联系人已删除');
    deleteTarget.value = null;
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除联系人失败');
  } finally {
    deleting.value = false;
  }
}

function kindIcon(kind: string): 'messages' | 'link' | 'map-pin' {
  if (kind === 'email') return 'link';
  if (kind === 'address') return 'map-pin';
  return 'messages';
}

function kindLabel(kind: string): string {
  if (kind === 'email') return '邮箱';
  if (kind === 'address') return '地址';
  return '电话';
}

function itemHref(kind: string, value: string): string {
  if (kind === 'email') return `mailto:${value}`;
  if (kind === 'phone') return `tel:${value.replace(/\s+/g, '')}`;
  return '';
}

function birthdayText(contact: Contact): string {
  if (!contact.birthdayOn) return '';
  if (contact.birthdayOn.startsWith('0000')) return contact.birthdayOn.slice(5).replace('-', ' 月 ') + ' 日';
  return contact.birthdayOn;
}

onMounted(load);
</script>

<template>
  <section class="contacts">
    <div class="contacts__bar">
      <AppInput v-model="query" type="search" placeholder="搜索姓名、组织或电话" aria-label="搜索联系人" />
      <AppButton variant="outline" @click="openCreate"><AppIcon name="plus" :size="16" />添加联系人</AppButton>
    </div>

    <p v-if="loadError" class="contacts__error" role="alert">{{ loadError }}</p>
    <div v-else-if="contacts.loading && !contacts.items.length" class="contacts__loading">正在加载通讯录…</div>
    <AppEmptyState
      v-else-if="!filteredContacts.length"
      :icon="query ? 'search' : 'user'"
      :title="query ? '没有匹配的联系人' : '通讯录还是空的'"
      :description="query ? '换个关键词试试。' : '记下常联系的人，需要时不用再翻手机。'"
    >
      <AppButton v-if="!query" variant="outline" @click="openCreate">添加第一位联系人</AppButton>
    </AppEmptyState>
    <template v-else>
      <p class="contacts__count">共 {{ filteredContacts.length }} 位联系人<span v-if="categories.length"> · {{ categories.join(' / ') }}</span></p>
      <ul class="contacts__grid">
        <li v-for="contact in filteredContacts" :key="contact.id" class="contact">
          <header class="contact__head">
            <AppAvatar :name="contact.name" :size="42" />
            <div class="contact__who">
              <div class="contact__name-row">
                <span class="contact__name">{{ contact.name }}</span>
                <AppStatusTag v-if="contact.ownerScope === 'private'" tone="warning">仅自己</AppStatusTag>
                <AppStatusTag v-if="contact.category" tone="neutral">{{ contact.category }}</AppStatusTag>
              </div>
              <p v-if="contact.organization || contact.title" class="contact__org">
                {{ [contact.organization, contact.title].filter(Boolean).join(' · ') }}
              </p>
            </div>
            <div class="contact__actions">
              <AppIconButton icon="edit" label="编辑联系人" size="sm" @click="openEdit(contact)" />
              <AppIconButton v-if="canDelete(contact)" icon="trash" label="删除联系人" size="sm" @click="deleteTarget = contact" />
            </div>
          </header>
          <div v-if="contact.items.length" class="contact__items">
            <component
              :is="itemHref(item.kind, item.value) ? 'a' : 'span'"
              v-for="item in contact.items"
              :key="item.id"
              class="contact__item"
              :href="itemHref(item.kind, item.value) || undefined"
            >
              <AppIcon :name="kindIcon(item.kind)" :size="14" />
              <span class="contact__item-label">{{ item.label || kindLabel(item.kind) }}</span>
              <span class="contact__item-value">{{ item.value }}</span>
            </component>
          </div>
          <p v-if="contact.birthdayOn" class="contact__birthday">
            <AppIcon name="calendar" :size="14" />生日 {{ birthdayText(contact) }}
          </p>
          <p v-if="contact.note" class="contact__note">{{ contact.note }}</p>
        </li>
      </ul>
    </template>

    <AppDialog v-model="formOpen" :title="editing ? '编辑联系人' : '添加联系人'" size="lg">
      <form class="at-stack" @submit.prevent="submit">
        <div class="contacts__form-grid">
          <AppField label="姓名" for-id="contact-name" required>
            <AppInput id="contact-name" v-model="form.name" placeholder="例如：王老师" :maxlength="80" />
          </AppField>
          <AppField label="分类" for-id="contact-category" hint="如学校、亲戚、邻居">
            <AppInput id="contact-category" v-model="form.category" placeholder="例如：学校" />
          </AppField>
        </div>
        <div class="contacts__form-grid">
          <AppField label="组织" for-id="contact-org">
            <AppInput id="contact-org" v-model="form.organization" placeholder="例如：实验小学" />
          </AppField>
          <AppField label="职务" for-id="contact-title">
            <AppInput id="contact-title" v-model="form.title" placeholder="例如：班主任" />
          </AppField>
        </div>
        <div class="contacts__form-grid">
          <AppField label="生日" for-id="contact-birthday" hint="可留空">
            <AppInput id="contact-birthday" v-model="form.birthdayOn" type="date" />
          </AppField>
          <AppField label="可见范围" for-id="contact-scope">
            <AppSelect id="contact-scope" v-model="form.ownerScope" :options="scopeOptions" aria-label="可见范围" />
          </AppField>
        </div>

        <AppField label="联系方式">
          <div class="contacts__items-editor">
            <div v-for="(item, index) in form.items" :key="index" class="contacts__item-row">
              <AppSelect v-model="item.kind" :options="kindOptions" aria-label="联系项类型" />
              <AppInput v-model="item.label" placeholder="标签" aria-label="联系项标签" />
              <AppInput v-model="item.value" placeholder="号码 / 邮箱 / 地址" aria-label="联系项内容" />
              <AppIconButton icon="trash" label="移除" size="sm" @click="removeItem(index)" />
            </div>
            <div class="contacts__add-row">
              <AppButton variant="ghost" size="sm" @click="addItem('phone')"><AppIcon name="plus" :size="14" />电话</AppButton>
              <AppButton variant="ghost" size="sm" @click="addItem('email')"><AppIcon name="plus" :size="14" />邮箱</AppButton>
              <AppButton variant="ghost" size="sm" @click="addItem('address')"><AppIcon name="plus" :size="14" />地址</AppButton>
            </div>
          </div>
        </AppField>

        <AppField label="备注" for-id="contact-note">
          <AppTextarea id="contact-note" v-model="form.note" :rows="3" placeholder="记下需要留意的细节…" :maxlength="2000" />
        </AppField>
        <p v-if="formError" class="contacts__error" role="alert"><AppIcon name="alert" :size="16" />{{ formError }}</p>
      </form>
      <template #footer>
        <AppButton variant="ghost" @click="formOpen = false">取消</AppButton>
        <AppButton :loading="saving" @click="submit">保存</AppButton>
      </template>
    </AppDialog>

    <AppConfirmDialog
      :model-value="deleteTarget !== null"
      title="删除联系人"
      :message="`确定删除「${deleteTarget?.name}」吗？关联的生日记录会保留，但会解除与联系人的关联。`"
      danger
      confirm-text="删除"
      :loading="deleting"
      @update:model-value="deleteTarget = null"
      @confirm="confirmDelete"
    />
  </section>
</template>

<style scoped>
.contacts {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.contacts__bar {
  display: flex;
  gap: var(--at-space-3);
  align-items: center;
}

.contacts__bar :deep(.at-input) {
  flex: 1;
}

.contacts__count {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.contacts__loading {
  padding: var(--at-space-6) 0;
  color: var(--at-text-secondary);
  font-size: var(--at-text-body-sm);
}

.contacts__grid {
  list-style: none;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--at-space-3);
}

.contact {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  padding: var(--at-space-4);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-xl);
  min-width: 0;
}

.contact__head {
  display: flex;
  align-items: flex-start;
  gap: var(--at-space-3);
}

.contact__who {
  flex: 1;
  min-width: 0;
}

.contact__name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.contact__name {
  font-size: var(--at-text-body);
  font-weight: 500;
}

.contact__org {
  margin-top: 2px;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.contact__actions {
  display: flex;
  gap: 2px;
}

.contact__items {
  display: flex;
  flex-direction: column;
  gap: 5px;
  border-top: 1px solid var(--at-color-hairline-soft);
  padding-top: var(--at-space-3);
}

.contact__item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--at-text-caption);
  color: var(--at-text-secondary);
  min-width: 0;
}

a.contact__item {
  color: var(--at-color-brand);
}

.contact__item-label {
  flex: none;
  color: var(--at-color-muted);
  font-size: var(--at-text-micro);
}

.contact__item-value {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.contact__birthday {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.contact__note {
  font-size: var(--at-text-micro);
  color: var(--at-text-secondary);
  line-height: 1.6;
  border-top: 1px dashed var(--at-color-hairline);
  padding-top: var(--at-space-2);
  white-space: pre-wrap;
}

.contacts__form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--at-space-3);
}

.contacts__items-editor {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
}

.contacts__item-row {
  display: grid;
  grid-template-columns: 96px 120px 1fr auto;
  gap: var(--at-space-2);
  align-items: center;
}

.contacts__add-row {
  display: flex;
  gap: var(--at-space-2);
  flex-wrap: wrap;
}

.contacts__error {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--at-color-danger);
  font-size: 13px;
}

@media (max-width: 1023px) {
  .contacts__grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 639px) {
  .contacts__bar {
    flex-direction: column;
    align-items: stretch;
  }

  .contacts__form-grid {
    grid-template-columns: 1fr;
  }

  .contacts__item-row {
    grid-template-columns: 1fr 1fr;
  }

  .contacts__item-row :deep(.at-input:last-of-type) {
    grid-column: 1 / -1;
  }
}
</style>
