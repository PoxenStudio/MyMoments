<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import AppButton from '@/components/base/AppButton.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppErrorState from '@/components/base/AppErrorState.vue';
import AppField from '@/components/base/AppField.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import AppSwitch from '@/components/base/AppSwitch.vue';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AudiencePicker from '@/components/business/AudiencePicker.vue';
import MentionRenderer from '@/components/business/MentionRenderer.vue';
import MentionTextarea from '@/components/business/MentionTextarea.vue';
import { useNotesStore } from '@/stores/notes';
import { useAuthStore } from '@/stores/auth';
import { useFamilyStore } from '@/stores/family';
import { useContactsStore } from '@/stores/contacts';
import { useToastStore } from '@/stores/toast';
import type { MentionTarget, Note, Visibility } from '@/types';
import { formatShortDate, formatTimestamp, todayISO } from '@/utils/chrono';

const auth = useAuthStore();
const family = useFamilyStore();
const contacts = useContactsStore();
const store = useNotesStore();
const toast = useToastStore();
const route = useRoute();

const today = todayISO();
const search = ref('');
const category = ref('');
const dialogOpen = ref(false);
const editingId = ref<number | null>(null);
const saving = ref(false);
const deleting = ref<Note | null>(null);
const deleteLoading = ref(false);
const mentions = ref<MentionTarget[]>([]);

const form = reactive({
  title: '',
  body: '',
  color: 'default',
  category: '',
  memoryDate: '',
  visibility: 'household' as Visibility,
  audienceIds: [] as number[],
  pinned: false,
  items: [] as { text: string; checked: boolean }[],
});

const colors = [
  { value: 'default', label: '白' },
  { value: 'sand', label: '沙' },
  { value: 'mint', label: '薄荷' },
  { value: 'peach', label: '蜜桃' },
  { value: 'lavender', label: '薰衣草' },
  { value: 'sky', label: '天空' },
  { value: 'rose', label: '玫瑰' },
];
const visibilityOptions = [
  { value: 'household', label: '全家可见' },
  { value: 'selected', label: '指定成员' },
  { value: 'private', label: '仅自己' },
];

const mentionMembers = computed(() => family.activeMembers.filter((m) => m.id !== auth.user?.id));
const mentionContacts = computed(() => contacts.items);
const categories = computed(() => {
  const set = new Set<string>();
  for (const note of store.items) {
    if (note.category) set.add(note.category);
  }
  return [...set];
});
const editingNote = computed(() =>
  editingId.value === null ? null : store.items.find((n) => n.id === editingId.value) ?? null,
);

function canManage(note: Note): boolean {
  return note.authorId === auth.user?.id || auth.user?.role === 'admin';
}

async function reload(): Promise<void> {
  try {
    await store.load({
      q: search.value.trim() || undefined,
      category: category.value || undefined,
    }, true);
  } catch {
    toast.error('便签加载失败，请稍后重试');
  }
}

let searchTimer: number | undefined;
watch([search, category], () => {
  window.clearTimeout(searchTimer);
  searchTimer = window.setTimeout(() => { void reload(); }, 250);
});

onMounted(async () => {
  await family.load();
  void contacts.load();
  await reload();
  const open = Number(route.query.open);
  if (open) {
    const note = store.items.find((n) => n.id === open);
    if (note) openEdit(note);
  }
  if (route.query.new === '1') openCreate();
});

function resetForm(): void {
  form.title = '';
  form.body = '';
  form.color = 'default';
  form.category = '';
  form.memoryDate = '';
  form.visibility = 'household';
  form.audienceIds = [];
  form.pinned = false;
  form.items = [];
  mentions.value = [];
}

function openCreate(): void {
  editingId.value = null;
  resetForm();
  dialogOpen.value = true;
  void nextTick(() => {
    document.querySelector<HTMLInputElement>('.note-dialog input')?.focus();
  });
}

function openEdit(note: Note): void {
  editingId.value = note.id;
  resetForm();
  form.title = note.title ?? '';
  form.body = note.body ?? '';
  form.color = note.color ?? 'default';
  form.category = note.category ?? '';
  form.memoryDate = note.memoryDate ?? '';
  form.visibility = note.visibility;
  form.audienceIds = [...note.audienceIds];
  form.pinned = note.pinned;
  form.items = note.items.map((it) => ({ text: it.text, checked: it.checked }));
  mentions.value = [...note.mentions];
  dialogOpen.value = true;
}

function addItem(): void {
  form.items.push({ text: '', checked: false });
}

function removeItem(index: number): void {
  form.items.splice(index, 1);
}

async function submit(): Promise<void> {
  const items = form.items.filter((it) => it.text.trim()).map((it) => ({ text: it.text.trim(), checked: it.checked }));
  if (!form.title.trim() && !form.body.trim() && items.length === 0) {
    toast.error('请填写标题、正文或检查清单');
    return;
  }
  saving.value = true;
  const payload = {
    title: form.title.trim() || null,
    body: form.body.trim() || null,
    color: form.color || null,
    category: form.category.trim() || null,
    memoryDate: form.memoryDate || null,
    visibility: form.visibility,
    audienceIds: form.visibility === 'selected' ? form.audienceIds : [],
    mentions: mentions.value,
    pinned: form.pinned,
    items,
  };
  try {
    if (editingId.value) {
      await store.update(editingId.value, payload);
      toast.success('便签已更新');
    } else {
      await store.create(payload);
      toast.success('便签已创建');
    }
    dialogOpen.value = false;
  } catch {
    toast.error('保存失败，请稍后重试');
  } finally {
    saving.value = false;
  }
}

async function togglePin(note: Note): Promise<void> {
  try {
    await store.setPinned(note.id, !note.pinned);
  } catch {
    toast.error('操作失败，请稍后重试');
  }
}

async function confirmDelete(): Promise<void> {
  if (!deleting.value) return;
  deleteLoading.value = true;
  try {
    await store.remove(deleting.value.id);
    toast.success('便签已删除');
    deleting.value = null;
    dialogOpen.value = false;
  } catch {
    toast.error('删除失败，请稍后重试');
  } finally {
    deleteLoading.value = false;
  }
}
</script>

<template>
  <div class="at-page">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">记录</p>
        <h1 class="at-page__title">便签</h1>
        <p class="at-page__subtitle">写给家人的话、要办的事和重要的日子，都可以记在这里。</p>
      </div>
      <div class="at-page__actions">
        <AppButton @click="openCreate">新建便签</AppButton>
      </div>
    </header>

    <div class="toolbar">
      <div class="toolbar__chips" role="group" aria-label="按分类筛选">
        <button
          type="button"
          class="chip"
          :class="{ 'is-active': category === '' }"
          @click="category = ''"
        >
          全部
        </button>
        <button
          v-for="item in categories"
          :key="item"
          type="button"
          class="chip"
          :class="{ 'is-active': category === item }"
          @click="category = item"
        >
          {{ item }}
        </button>
      </div>
      <label class="search">
        <AppIcon name="search" :size="16" />
        <input v-model="search" type="search" placeholder="搜索便签" aria-label="搜索便签" />
      </label>
    </div>

    <AppErrorState v-if="store.error" kind="server" compact @retry="reload" />
    <p v-else-if="store.loading && !store.items.length" class="at-faint loading">正在加载便签…</p>
    <AppEmptyState
      v-else-if="!store.items.length"
      icon="notes"
      title="还没有便签"
      description="写一句话、列一份清单，或记下一个重要的日子。"
    >
      <AppButton @click="openCreate">新建便签</AppButton>
    </AppEmptyState>

    <div v-else class="notes-grid">
      <article
        v-for="note in store.items"
        :key="note.id"
        class="note at-card at-card--hover"
        :class="[`note--${note.color ?? 'default'}`, { 'is-pinned': note.pinned }]"
        tabindex="0"
        role="button"
        @click="openEdit(note)"
        @keydown.enter="openEdit(note)"
      >
        <div class="note__top">
          <span v-if="note.category" class="note__category">{{ note.category }}</span>
          <span v-else class="note__category note__category--ghost">便签</span>
          <AppIconButton
            icon="pin"
            :label="note.pinned ? '取消置顶' : '置顶'"
            size="sm"
            :active="note.pinned"
            @click.stop="togglePin(note)"
          />
        </div>
        <h3 v-if="note.title" class="note__title">{{ note.title }}</h3>
        <p v-if="note.body" class="note__body">
          <MentionRenderer :text="note.body" :mentions="note.mentions" :contacts="mentionContacts" :members="family.activeMembers" />
        </p>
        <ul v-if="note.items.length" class="note__items">
          <li v-for="item in note.items.slice(0, 5)" :key="item.id" :class="{ 'is-checked': item.checked }">
            <span class="note__box"><AppIcon v-if="item.checked" name="check" :size="11" /></span>
            <span>{{ item.text }}</span>
          </li>
          <li v-if="note.items.length > 5" class="note__more">还有 {{ note.items.length - 5 }} 项</li>
        </ul>
        <footer class="note__footer">
          <AppAvatar :name="note.authorName ?? ''" :size="22" />
          <span class="note__author">{{ note.authorName }}</span>
          <span class="note__time">{{ formatTimestamp(note.updatedAt, { dateOnly: true }) }}</span>
          <span v-if="note.closedAt" class="note__closed">已关闭</span>
        </footer>
        <p v-if="note.memoryDate" class="note__memory">
          回忆 · {{ formatShortDate(note.memoryDate) }}
        </p>
      </article>
    </div>

    <AppDialog
      :model-value="dialogOpen"
      :title="editingId ? '编辑便签' : '新建便签'"
      size="lg"
      @update:model-value="dialogOpen = false"
    >
      <div class="note-dialog">
        <AppField label="标题" for-id="note-title">
          <AppInput id="note-title" v-model="form.title" placeholder="给这张便签起个名字（可选）" :maxlength="120" />
        </AppField>
        <AppField label="正文" hint="输入 @ 可以提及家人，他们会收到通知。">
          <MentionTextarea
            v-model="form.body"
            v-model:mentions="mentions"
            :members="mentionMembers"
            :contacts="mentionContacts"
            placeholder="写点什么…"
            :rows="5"
            :maxlength="10000"
          />
        </AppField>

        <div class="note-dialog__grid">
          <AppField label="颜色">
            <div class="swatches" role="group" aria-label="便签颜色">
              <button
                v-for="c in colors"
                :key="c.value"
                type="button"
                class="swatch"
                :class="[`swatch--${c.value}`, { 'is-active': form.color === c.value }]"
                :aria-label="c.label"
                :aria-pressed="form.color === c.value"
                @click="form.color = c.value"
              />
            </div>
          </AppField>
          <AppField label="分类" for-id="note-category">
            <AppInput id="note-category" v-model="form.category" placeholder="如：出行、菜谱" :maxlength="40" />
          </AppField>
          <AppField label="回忆日期" for-id="note-memory">
            <AppInput id="note-memory" v-model="form.memoryDate" type="date" :max="today" />
          </AppField>
          <AppField label="可见范围">
            <AppSelect v-model="form.visibility" :options="visibilityOptions" aria-label="可见范围" />
          </AppField>
        </div>

        <AudiencePicker
          v-if="form.visibility === 'selected'"
          v-model="form.audienceIds"
          :members="mentionMembers"
        />

        <AppField label="检查清单">
          <ul class="checklist">
            <li v-for="(item, index) in form.items" :key="index" class="checklist__row">
              <label class="checklist__check">
                <input v-model="item.checked" type="checkbox" />
              </label>
              <input v-model="item.text" class="checklist__input" placeholder="清单项" :maxlength="200" />
              <AppIconButton icon="close" label="移除清单项" size="sm" @click="removeItem(index)" />
            </li>
          </ul>
          <AppButton variant="ghost" size="sm" @click="addItem">添加清单项</AppButton>
        </AppField>

        <AppSwitch v-model="form.pinned" label="置顶便签" description="置顶后会显示在便签墙最前面。" />
      </div>

      <template #footer>
        <AppButton
          v-if="editingNote && canManage(editingNote)"
          variant="ghost"
          @click="deleting = editingNote"
        >
          删除
        </AppButton>
        <span class="dialog-spacer" />
        <AppButton variant="ghost" @click="dialogOpen = false">取消</AppButton>
        <AppButton :loading="saving" @click="submit">保存</AppButton>
      </template>
    </AppDialog>

    <AppConfirmDialog
      :model-value="deleting !== null"
      title="删除便签"
      message="删除后这张便签和它的提及通知都会消失，且无法恢复。"
      confirm-text="删除"
      danger
      :loading="deleteLoading"
      @update:model-value="deleting = null"
      @confirm="confirmDelete"
    />
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
  margin-bottom: var(--at-space-4);
}

.toolbar__chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.chip {
  padding: 7px 14px;
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface);
  color: var(--at-color-muted);
  font-size: var(--at-text-micro);
  transition: background var(--at-duration-fast) var(--at-ease), color var(--at-duration-fast) var(--at-ease);
}

.chip:hover {
  background: var(--at-bg-surface-strong);
}

.chip.is-active {
  background: var(--at-color-ink);
  border-color: var(--at-color-ink);
  color: var(--at-text-on-primary);
}

.search {
  display: inline-flex;
  align-items: center;
  gap: var(--at-space-2);
  padding: 0 var(--at-space-3);
  height: var(--at-control-height);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface);
  color: var(--at-color-muted);
}

.search input {
  border: 0;
  outline: none;
  background: transparent;
  font-size: var(--at-text-caption);
  min-width: 160px;
  color: var(--at-text-primary);
}

.loading {
  padding: var(--at-space-6) 0;
  text-align: center;
  font-size: var(--at-text-caption);
}

.notes-grid {
  column-count: 3;
  column-gap: var(--at-space-4);
}

.note {
  break-inside: avoid;
  margin-bottom: var(--at-space-4);
  padding: var(--at-space-4);
  background: var(--at-card-default);
  cursor: pointer;
}

.note:hover,
.note:focus-visible {
  outline: none;
}

.note.is-pinned {
  box-shadow: var(--at-shadow-card), inset 0 0 0 1px var(--at-color-hairline-strong);
}

.note.is-pinned:hover,
.note.is-pinned:focus-visible {
  box-shadow: var(--at-shadow-card-hover), inset 0 0 0 1px var(--at-color-hairline-strong);
}

.note--soft { background: var(--at-card-soft); }
.note--sand { background: var(--at-card-sand); }
.note--mint { background: var(--at-card-mint); }
.note--peach { background: var(--at-card-peach); }
.note--lavender { background: var(--at-card-lavender); }
.note--sky { background: var(--at-card-sky); }
.note--rose { background: var(--at-card-rose); }

.note__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-2);
}

.note__category {
  font-size: 11px;
  letter-spacing: 0.4px;
  color: var(--at-color-muted);
  text-transform: uppercase;
}

.note__category--ghost {
  opacity: 0.45;
}

.note__title {
  margin-top: var(--at-space-2);
  font-family: var(--at-font-display);
  font-weight: 400;
  font-size: var(--at-text-title-sm);
  line-height: 1.35;
}

.note__body {
  margin-top: var(--at-space-2);
  font-size: var(--at-text-caption);
  line-height: 1.7;
  color: var(--at-text-secondary);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  display: -webkit-box;
  -webkit-line-clamp: 6;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.note__items {
  margin-top: var(--at-space-3);
  display: flex;
  flex-direction: column;
  gap: 6px;
  list-style: none;
}

.note__items li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: var(--at-text-micro);
  color: var(--at-text-secondary);
}

.note__box {
  flex: 0 0 auto;
  display: grid;
  place-items: center;
  width: 14px;
  height: 14px;
  margin-top: 2px;
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: 3px;
  background: var(--at-bg-surface);
}

.note__items li.is-checked {
  color: var(--at-color-muted);
  text-decoration: line-through;
}

.note__items li.is-checked .note__box {
  background: var(--at-color-ink);
  border-color: var(--at-color-ink);
  color: var(--at-text-on-primary);
}

.note__more {
  color: var(--at-color-muted);
}

.note__footer {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: var(--at-space-3);
  font-size: 11px;
  color: var(--at-color-muted);
}

.note__author {
  color: var(--at-text-secondary);
}

.note__time {
  margin-left: auto;
}

.note__closed {
  flex: none;
  font-size: 11px;
  color: var(--at-color-muted);
  padding: 1px 8px;
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-pill);
}

.note__memory {
  margin-top: var(--at-space-1);
  font-size: 11px;
  color: var(--at-color-muted);
}

.note-dialog {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.note-dialog__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--at-space-3);
}

.swatches {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.swatch {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  border: 1px solid rgba(12, 10, 9, 0.12);
  transition: transform var(--at-duration-fast) var(--at-ease), box-shadow var(--at-duration-fast) var(--at-ease);
}

.swatch--default { background: var(--at-card-default); }
.swatch--sand { background: var(--at-card-sand); }
.swatch--mint { background: var(--at-card-mint); }
.swatch--peach { background: var(--at-card-peach); }
.swatch--lavender { background: var(--at-card-lavender); }
.swatch--sky { background: var(--at-card-sky); }
.swatch--rose { background: var(--at-card-rose); }

.swatch.is-active {
  transform: scale(1.12);
  box-shadow: 0 0 0 2px var(--at-color-ink);
}

.checklist {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  margin-bottom: var(--at-space-2);
  list-style: none;
}

.checklist__row {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
}

.checklist__check {
  display: grid;
  place-items: center;
}

.checklist__input {
  flex: 1;
  height: 36px;
  padding: 0 var(--at-space-3);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface);
  font-size: var(--at-text-caption);
  color: var(--at-text-primary);
}

.dialog-spacer {
  flex: 1;
}

@media (max-width: 1023px) {
  .notes-grid {
    column-count: 2;
  }
}

@media (max-width: 639px) {
  .notes-grid {
    column-count: 1;
  }

  .note-dialog__grid {
    grid-template-columns: 1fr;
  }

  .search {
    flex: 1;
  }

  .search input {
    width: 100%;
    min-width: 0;
  }
}
</style>
