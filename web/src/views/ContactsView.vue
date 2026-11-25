<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import AppButton from '@/components/base/AppButton.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppErrorState from '@/components/base/AppErrorState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppStatusTag from '@/components/base/AppStatusTag.vue';
import ContactFormDialog from '@/components/business/ContactFormDialog.vue';
import { ApiError } from '@/api/client';
import { useContactsStore, type BlockedContact, type ContactAssociations } from '@/stores/contacts';
import { useToastStore } from '@/stores/toast';
import type { Contact } from '@/types';
import { avatarUrl } from '@/utils/avatar';

const store = useContactsStore();
const toast = useToastStore();

const searchInput = ref('');
const formOpen = ref(false);
const editingContact = ref<Contact | null>(null);
const deleting = ref<Contact | null>(null);
const deleteLoading = ref(false);
const deleteAssociations = ref<ContactAssociations | null>(null);
const associationsLoading = ref(false);
const importing = ref(false);
const cleanupConfirmOpen = ref(false);
const cleanupRunning = ref(false);
const cleanupBlocked = ref<BlockedContact[]>([]);
const fileInput = ref<HTMLInputElement | null>(null);
const pinning = ref<number | null>(null);

const filtered = computed(() => {
  const q = searchInput.value.trim().toLowerCase();
  if (!q) return store.items;
  return store.items.filter((c) => {
    const hay = [
      c.name,
      c.organization ?? '',
      c.wechat ?? '',
      c.officePhone ?? '',
      ...c.items.map((i) => i.value),
    ]
      .join(' ')
      .toLowerCase();
    return hay.includes(q);
  });
});

function birthdayLabel(c: Contact): string {
  if (c.birthdayMonth == null || c.birthdayDay == null) return '';
  const base = `${c.birthdayMonth}月${c.birthdayDay}日`;
  return c.birthYear ? `${c.birthYear}年${base}` : base;
}

function primaryPhone(c: Contact): string {
  return c.items.find((i) => i.kind === 'phone' && i.isPrimaryCall)?.value ?? c.items.find((i) => i.kind === 'phone')?.value ?? '';
}

function reminderTags(c: Contact): string[] {
  const tags: string[] = [];
  if (c.reminders.some((r) => r.kind === 'spring_festival' && r.enabled)) tags.push('春节拜年');
  if (c.reminders.some((r) => r.kind === 'traditional' && r.enabled)) tags.push('传统节日');
  if (c.reminders.some((r) => r.kind === 'western' && r.enabled)) tags.push('西方节日');
  const customCount = c.reminders.filter((r) => r.kind === 'custom' && r.enabled).length;
  if (customCount) tags.push(`${customCount} 项待办`);
  return tags;
}

function openCreate(): void {
  editingContact.value = null;
  formOpen.value = true;
}

async function togglePin(c: Contact): Promise<void> {
  if (pinning.value !== null) return;
  pinning.value = c.id;
  try {
    await store.setPinned(c.id, !c.pinned);
    toast.success(c.pinned ? `已取消置顶「${c.name}」` : `已置顶「${c.name}」`);
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '操作失败，请稍后重试');
  } finally {
    pinning.value = null;
  }
}

function openEdit(c: Contact): void {
  editingContact.value = c;
  formOpen.value = true;
}

function askDelete(c: Contact): void {
  deleting.value = c;
  deleteAssociations.value = null;
  associationsLoading.value = true;
  void store.associations(c.id)
    .then((a) => { deleteAssociations.value = a; })
    .catch(() => { deleteAssociations.value = null; })
    .finally(() => { associationsLoading.value = false; });
}

async function confirmDelete(): Promise<void> {
  if (!deleting.value) return;
  deleteLoading.value = true;
  try {
    await store.remove(deleting.value.id);
    toast.success('联系人已删除');
    deleting.value = null;
    deleteAssociations.value = null;
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除失败，请稍后重试');
  } finally {
    deleteLoading.value = false;
  }
}

function openCleanup(): void {
  cleanupBlocked.value = [];
  cleanupConfirmOpen.value = true;
}

async function confirmCleanup(): Promise<void> {
  cleanupRunning.value = true;
  try {
    const result = await store.cleanup();
    if (result.blocked.length) {
      cleanupBlocked.value = result.blocked;
      toast.error(`${result.blocked.length} 位联系人仍有关联，未执行清理`);
    } else {
      toast.success(`已清理 ${result.deleted} 位联系人`);
      cleanupConfirmOpen.value = false;
    }
    await store.load({}, true);
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '清理失败，请稍后重试');
  } finally {
    cleanupRunning.value = false;
  }
}

function blockedSummary(b: BlockedContact): string {
  const parts = [];
  if (b.counts.tasks) parts.push(`${b.counts.tasks} 待办`);
  if (b.counts.messages) parts.push(`${b.counts.messages} 留言`);
  if (b.counts.notes) parts.push(`${b.counts.notes} 便签`);
  return parts.join('、');
}

function triggerImport(): void {
  fileInput.value?.click();
}

async function onImport(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  importing.value = true;
  try {
    const result = await store.importFile(file);
    toast.success(`已导入 ${result.created} 位联系人`);
    await store.load({}, true);
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '导入失败，请确认文件为 vCard 格式');
  } finally {
    importing.value = false;
    input.value = '';
  }
}

onMounted(async () => {
  try {
    await store.load();
  } catch {
    // 错误由 AppErrorState 呈现
  }
});
</script>

<template>
  <div class="at-page">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">通讯录</p>
        <h1 class="at-page__title">联系人</h1>
        <p class="at-page__subtitle">记录家人常联系的人，电话、微信、生日和问候提醒都集中在这里。</p>
      </div>
      <div class="at-page__actions">
        <input ref="fileInput" type="file" accept=".vcf,text/vcard" hidden @change="onImport" />
        <AppButton variant="outline" :loading="importing" @click="triggerImport">导入 vCard</AppButton>
        <AppButton @click="openCreate">新增联系人</AppButton>
      </div>
    </header>

    <div class="toolbar">
      <label class="search">
        <AppIcon name="search" :size="16" />
        <input v-model="searchInput" type="search" placeholder="搜索联系人" aria-label="搜索联系人" />
      </label>
    </div>

    <AppErrorState v-if="store.error" kind="server" compact @retry="store.load({}, true)" />
    <p v-else-if="store.loading && !store.items.length" class="at-faint loading">正在加载联系人…</p>
    <AppEmptyState
      v-else-if="!store.items.length"
      icon="contacts"
      title="还没有联系人"
      description="手动新增一位联系人，或从 vCard 文件批量导入。"
    >
      <AppButton @click="openCreate">新增联系人</AppButton>
    </AppEmptyState>
    <AppEmptyState v-else-if="!filtered.length" icon="search" title="没有匹配的联系人" description="换个关键词试试。" />

    <div v-else class="contacts-grid">
      <article
        v-for="c in filtered"
        :key="c.id"
        class="contact at-card at-card--hover"
        :class="{ 'is-pinned': c.pinned }"
        tabindex="0"
        role="button"
        @click="openEdit(c)"
        @keydown.enter="openEdit(c)"
      >
        <div class="contact__top">
          <AppAvatar :name="c.name" :size="48" :src="avatarUrl(c.avatarKey)" />
          <div class="contact__meta">
            <h3 class="contact__name">{{ c.name }}</h3>
            <p v-if="c.title || c.organization" class="contact__org">{{ [c.title, c.organization].filter(Boolean).join(' · ') }}</p>
          </div>
          <AppIconButton
            :icon="c.pinned ? 'pin-off' : 'pin'"
            :label="c.pinned ? '取消置顶' : '置顶'"
            size="sm"
            :active="c.pinned"
            :disabled="pinning === c.id"
            @click.stop="togglePin(c)"
          />
          <AppIconButton icon="trash" label="删除联系人" size="sm" @click.stop="askDelete(c)" />
        </div>

        <ul class="contact__info">
          <li v-if="primaryPhone(c)" class="contact__line">
            <span class="contact__k">手机</span>
            <span>{{ primaryPhone(c) }}</span>
          </li>
          <li v-if="c.wechat" class="contact__line">
            <span class="contact__k">微信</span>
            <span>{{ c.wechat }}</span>
          </li>
          <li v-if="c.officePhone" class="contact__line">
            <span class="contact__k">办公</span>
            <span>{{ c.officePhone }}</span>
          </li>
          <li v-if="birthdayLabel(c)" class="contact__line">
            <span class="contact__k">生日</span>
            <span>{{ birthdayLabel(c) }}</span>
            <AppStatusTag v-if="c.showInCalendar" tone="brand">日历</AppStatusTag>
          </li>
        </ul>

        <div v-if="reminderTags(c).length" class="contact__tags">
          <span v-for="tag in reminderTags(c)" :key="tag" class="contact__tag">{{ tag }}</span>
        </div>
      </article>
    </div>

    <footer v-if="store.items.length" class="cleanup-bar">
      <p class="cleanup-bar__hint">删除所有未被待办、留言、便签关联的联系人。</p>
      <AppButton variant="ghost" @click="openCleanup">清理联系人</AppButton>
    </footer>

    <ContactFormDialog v-model="formOpen" :contact="editingContact" @saved="store.load({}, true)" />

    <AppConfirmDialog
      :model-value="deleting !== null"
      :title="'删除联系人'"
      :message="deleting && deleteAssociations && (deleteAssociations.tasks || deleteAssociations.messages || deleteAssociations.notes)
        ? `「${deleting.name}」已被 ${deleteAssociations.tasks} 条待办、${deleteAssociations.messages} 条留言、${deleteAssociations.notes} 条便签关联。删除将解除这些关联并关闭/完成相关内容，此操作不可撤销。`
        : deleting ? `确定删除「${deleting.name}」吗？此操作不可撤销。` : ''"
      confirm-text="删除"
      danger
      :loading="deleteLoading || associationsLoading"
      @update:model-value="(v) => { if (!v) deleting = null; }"
      @confirm="confirmDelete"
    />

    <AppConfirmDialog
      :model-value="cleanupConfirmOpen"
      title="清理联系人"
      message="将删除所有未被任何待办、留言、便签关联的联系人，此操作不可撤销。"
      confirm-text="开始清理"
      danger
      :loading="cleanupRunning"
      @update:model-value="cleanupConfirmOpen = $event"
      @confirm="confirmCleanup"
    />

    <AppDialog
      :model-value="cleanupBlocked.length > 0"
      title="无法清理"
      size="sm"
      @update:model-value="cleanupBlocked = []"
    >
      <p class="blocked-note">以下联系人仍被待办、留言或便签关联，不能清理：</p>
      <ul class="blocked-list">
        <li v-for="b in cleanupBlocked" :key="b.id" class="blocked-item">
          <span class="blocked-item__name">{{ b.name }}</span>
          <span class="blocked-item__counts">{{ blockedSummary(b) }}</span>
        </li>
      </ul>
      <template #footer>
        <AppButton @click="cleanupBlocked = []">知道了</AppButton>
      </template>
    </AppDialog>
  </div>
</template>

<style scoped>
.toolbar {
  margin-bottom: var(--at-space-4);
}
.search {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  max-width: 360px;
  height: 44px;
  padding: 0 var(--at-space-4);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-pill);
  color: var(--at-color-muted);
}
.search input {
  flex: 1;
  border: 0;
  background: transparent;
  color: var(--at-text-primary);
  font-size: var(--at-text-body-sm);
}
.search input:focus {
  outline: none;
}
.contacts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--at-space-4);
}
.contact {
  padding: var(--at-space-4);
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
}
.contact.is-pinned {
  border-color: var(--at-color-hairline-strong);
}
.contact__top {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
}
.contact__meta {
  flex: 1;
  min-width: 0;
}
.contact__name {
  font-size: var(--at-text-body-sm);
  font-weight: 500;
}
.contact__org {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.contact__info {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
}
.contact__line {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  font-size: var(--at-text-caption);
  color: var(--at-text-secondary);
}
.contact__k {
  flex: none;
  width: 40px;
  color: var(--at-color-muted);
}
.contact__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}
.contact__tag {
  font-size: 12px;
  color: var(--at-color-primary);
  background: color-mix(in srgb, var(--at-color-primary) 10%, transparent);
  border-radius: var(--at-radius-pill);
  padding: 2px 10px;
}
.loading {
  padding: var(--at-space-6) 0;
  text-align: center;
}
.cleanup-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
  margin-top: var(--at-space-6);
  padding-top: var(--at-space-4);
  border-top: 1px solid var(--at-color-hairline);
}
.cleanup-bar__hint {
  color: var(--at-color-muted);
  font-size: var(--at-text-micro);
}
.blocked-note {
  color: var(--at-text-secondary);
  font-size: var(--at-text-body-sm);
}
.blocked-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  margin-top: var(--at-space-3);
}
.blocked-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
  padding: var(--at-space-2) var(--at-space-3);
  background: var(--at-bg-surface-strong);
  border-radius: var(--at-radius-md);
}
.blocked-item__name {
  font-weight: 600;
}
.blocked-item__counts {
  color: var(--at-color-muted);
  font-size: var(--at-text-micro);
}
</style>
