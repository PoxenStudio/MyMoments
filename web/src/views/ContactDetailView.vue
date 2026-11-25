<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppButton from '@/components/base/AppButton.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import AppErrorState from '@/components/base/AppErrorState.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';
import AppStatusTag from '@/components/base/AppStatusTag.vue';
import ContactFormDialog from '@/components/business/ContactFormDialog.vue';
import { ApiError } from '@/api/client';
import { useContactsStore, type ContactAssociations } from '@/stores/contacts';
import { useToastStore } from '@/stores/toast';
import type { Contact } from '@/types';
import { avatarUrl } from '@/utils/avatar';

const route = useRoute();
const router = useRouter();
const store = useContactsStore();
const toast = useToastStore();

const contact = ref<Contact | null>(null);
const loading = ref(true);
const notFound = ref(false);
const editOpen = ref(false);
const deleting = ref(false);
const deleteLoading = ref(false);
const associations = ref<ContactAssociations | null>(null);
const associationsLoading = ref(false);

const id = computed(() => Number(route.params.id));

function birthdayLabel(c: Contact): string {
  if (c.birthdayMonth == null || c.birthdayDay == null) return '';
  const base = `${c.birthdayMonth}月${c.birthdayDay}日`;
  return c.birthYear ? `${c.birthYear}年${base}` : base;
}

function itemLabel(kind: string): string {
  if (kind === 'phone') return '电话';
  if (kind === 'email') return '邮箱';
  if (kind === 'address') return '地址';
  return '其他';
}

function reminderKindLabel(kind: string): string {
  if (kind === 'spring_festival') return '春节拜年';
  if (kind === 'traditional') return '传统节日';
  if (kind === 'western') return '西方节日';
  return '自定义';
}

async function load(): Promise<void> {
  loading.value = true;
  try {
    contact.value = await store.get(id.value);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) notFound.value = true;
    else toast.error('联系人加载失败');
  } finally {
    loading.value = false;
  }
}

async function askDelete(): Promise<void> {
  associationsLoading.value = true;
  try {
    associations.value = await store.associations(id.value);
    deleting.value = true;
  } catch {
    toast.error('无法检查关联信息');
  } finally {
    associationsLoading.value = false;
  }
}

async function confirmDelete(): Promise<void> {
  deleteLoading.value = true;
  try {
    await store.remove(id.value);
    toast.success('联系人已删除');
    await router.push({ name: 'contacts' });
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除失败，请稍后重试');
  } finally {
    deleteLoading.value = false;
    deleting.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div class="at-page">
    <div class="detail-head">
      <AppIconButton icon="arrow-left" label="返回联系人" @click="router.push({ name: 'contacts' })" />
      <div class="detail-head__spacer" />
      <AppButton v-if="contact" variant="outline" @click="editOpen = true">编辑</AppButton>
      <AppButton v-if="contact" variant="danger" @click="askDelete">删除</AppButton>
    </div>

    <AppErrorState v-if="notFound" kind="notfound" compact />
    <p v-else-if="loading" class="at-faint loading">正在加载联系人…</p>

    <template v-else-if="contact">
      <header class="profile at-card">
        <AppAvatar :name="contact.name" :size="72" :src="avatarUrl(contact.avatarKey)" />
        <div class="profile__meta">
          <h1 class="profile__name">{{ contact.name }}</h1>
          <p v-if="contact.title || contact.organization" class="profile__org">{{ [contact.title, contact.organization].filter(Boolean).join(' · ') }}</p>
          <div v-if="contact.category" class="profile__tags">
            <AppStatusTag tone="brand">{{ contact.category }}</AppStatusTag>
            <AppStatusTag v-if="contact.showInCalendar" tone="brand">日历</AppStatusTag>
          </div>
        </div>
      </header>

      <section class="panel at-card">
        <h2 class="panel__title">联系方式</h2>
        <ul class="lines">
          <li v-if="!contact.items.length" class="lines__empty">暂无联系方式</li>
          <li v-for="(it, i) in contact.items" :key="i" class="line">
            <span class="line__k">{{ it.label ?? itemLabel(it.kind) }}</span>
            <span class="line__v">{{ it.value }}</span>
            <AppStatusTag v-if="it.isPrimaryCall" tone="brand">主叫</AppStatusTag>
            <AppStatusTag v-if="it.isPrimarySms" tone="brand">短信</AppStatusTag>
          </li>
          <li v-if="contact.wechat" class="line"><span class="line__k">微信</span><span class="line__v">{{ contact.wechat }}</span></li>
          <li v-if="contact.officePhone" class="line"><span class="line__k">办公</span><span class="line__v">{{ contact.officePhone }}</span></li>
        </ul>
      </section>

      <section class="panel at-card">
        <h2 class="panel__title">重要信息</h2>
        <ul class="lines">
          <li v-if="birthdayLabel(contact)" class="line"><span class="line__k">生日</span><span class="line__v">{{ birthdayLabel(contact) }}</span></li>
          <li v-if="contact.note" class="line line--stack"><span class="line__k">备注</span><span class="line__v">{{ contact.note }}</span></li>
        </ul>
      </section>

      <section v-if="contact.reminders.length" class="panel at-card">
        <h2 class="panel__title">问候与待办</h2>
        <ul class="lines">
          <li v-for="r in contact.reminders" :key="r.id" class="line">
            <span class="line__k">{{ reminderKindLabel(r.kind) }}</span>
            <span class="line__v">{{ r.title || r.frequency || '—' }}</span>
            <AppStatusTag :tone="r.enabled ? 'brand' : 'neutral'">{{ r.enabled ? '启用' : '停用' }}</AppStatusTag>
          </li>
        </ul>
      </section>

      <section v-if="contact.relations.length" class="panel at-card">
        <h2 class="panel__title">关联</h2>
        <ul class="lines">
          <li v-for="(r, i) in contact.relations" :key="i" class="line">
            <span class="line__k">{{ r.relation || '关联' }}</span>
            <span class="line__v">{{ r.relatedName }}</span>
          </li>
        </ul>
      </section>
    </template>

    <ContactFormDialog v-model="editOpen" :contact="contact" @saved="load" />

    <AppConfirmDialog
      :model-value="deleting"
      title="删除联系人"
      :message="associations && (associations.tasks || associations.messages || associations.notes)
        ? `「${contact?.name}」已被 ${associations.tasks} 条待办、${associations.messages} 条留言、${associations.notes} 条便签关联。删除将解除这些关联并关闭/完成相关内容，且无法恢复。`
        : `确定删除「${contact?.name}」吗？此操作不可撤销。`"
      confirm-text="删除"
      danger
      :loading="deleteLoading || associationsLoading"
      @update:model-value="deleting = false"
      @confirm="confirmDelete"
    />
  </div>
</template>

<style scoped>
.detail-head {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  margin-bottom: var(--at-space-4);
}
.detail-head__spacer {
  flex: 1;
}
.loading {
  padding: var(--at-space-6) 0;
  text-align: center;
}
.profile {
  display: flex;
  align-items: center;
  gap: var(--at-space-4);
  padding: var(--at-space-5);
  margin-bottom: var(--at-space-4);
}
.profile__meta {
  min-width: 0;
}
.profile__name {
  font-family: var(--at-font-display);
  font-size: var(--at-text-title);
  font-weight: 400;
}
.profile__org {
  margin-top: 4px;
  color: var(--at-color-muted);
  font-size: var(--at-text-caption);
}
.profile__tags {
  display: flex;
  gap: var(--at-space-2);
  margin-top: var(--at-space-2);
}
.panel {
  padding: var(--at-space-5);
  margin-bottom: var(--at-space-4);
}
.panel__title {
  font-size: var(--at-text-caption);
  letter-spacing: 0.4px;
  text-transform: uppercase;
  color: var(--at-color-muted);
  margin-bottom: var(--at-space-3);
}
.lines {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  list-style: none;
}
.lines__empty {
  color: var(--at-color-muted);
  font-size: var(--at-text-caption);
}
.line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--at-space-2);
  font-size: var(--at-text-caption);
  color: var(--at-text-secondary);
}
.line--stack {
  align-items: flex-start;
  flex-direction: column;
}
.line__k {
  flex: none;
  width: 56px;
  color: var(--at-color-muted);
}
.line__v {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
</style>
