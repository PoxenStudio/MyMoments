<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import AppAvatar from '@/components/base/AppAvatar.vue';
import { avatarUrl } from '@/utils/avatar';
import AppButton from '@/components/base/AppButton.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppErrorState from '@/components/base/AppErrorState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import AudiencePicker from '@/components/business/AudiencePicker.vue';
import MentionRenderer from '@/components/business/MentionRenderer.vue';
import MentionTextarea from '@/components/business/MentionTextarea.vue';
import { useMessagesStore } from '@/stores/messages';
import { useAuthStore } from '@/stores/auth';
import { useFamilyStore } from '@/stores/family';
import { useContactsStore } from '@/stores/contacts';
import { useToastStore } from '@/stores/toast';
import type { MentionTarget, MessagePost, Visibility } from '@/types';
import { formatShortDate, formatTimestamp, todayISO } from '@/utils/chrono';

const auth = useAuthStore();
const family = useFamilyStore();
const contacts = useContactsStore();
const store = useMessagesStore();
const toast = useToastStore();
const route = useRoute();

const today = todayISO();
const filter = ref<'all' | 'mentions' | 'mine'>('all');
const search = ref('');
const composing = ref(false);
const saving = ref(false);
const highlightId = ref<number | null>(null);
const replyFor = ref<number | null>(null);
const replyBody = ref('');
const replyMentions = ref<MentionTarget[]>([]);
const replySaving = ref(false);

const draft = reactive({
  body: '',
  occurrenceDate: '',
  visibility: 'household' as Visibility,
  audienceIds: [] as number[],
  pinned: false,
});
const draftMentions = ref<MentionTarget[]>([]);

const editing = ref<MessagePost | null>(null);
const editForm = reactive({
  body: '',
  occurrenceDate: '',
  visibility: 'household' as Visibility,
  audienceIds: [] as number[],
  pinned: false,
});
const editMentions = ref<MentionTarget[]>([]);
const editSaving = ref(false);

const deleting = ref<MessagePost | null>(null);
const deleteLoading = ref(false);

const mentionMembers = computed(() => family.activeMembers.filter((m) => m.id !== auth.user?.id));
const mentionContacts = computed(() => contacts.items);
const filterTabs = [
  { key: 'all', label: '全部' },
  { key: 'mentions', label: '被提及' },
  { key: 'mine', label: '我的' },
] as const;
const visibilityOptions = [
  { value: 'household', label: '全家可见' },
  { value: 'selected', label: '指定成员' },
  { value: 'private', label: '仅自己' },
];

function visibilityLabel(v: Visibility): string {
  if (v === 'private') return '仅自己';
  if (v === 'selected') return '指定成员';
  return '全家可见';
}

function canManage(post: MessagePost): boolean {
  return post.authorId === auth.user?.id || auth.user?.role === 'admin';
}

function canEdit(post: MessagePost): boolean {
  return post.authorId === auth.user?.id || auth.user?.role === 'admin' || post.visibility === 'household';
}

async function reload(): Promise<void> {
  try {
    await store.load({
      q: search.value.trim() || undefined,
      filter: filter.value === 'all' ? undefined : filter.value,
    }, true);
  } catch {
    toast.error('留言加载失败，请稍后重试');
  }
}

let searchTimer: number | undefined;
watch([filter, search], () => {
  window.clearTimeout(searchTimer);
  searchTimer = window.setTimeout(() => { void reload(); }, 250);
});

onMounted(async () => {
  await family.load();
  void contacts.load();
  await reload();
  const open = Number(route.query.open);
  if (open) {
    await nextTick();
    const el = document.getElementById(`post-${open}`);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      highlightId.value = open;
      window.setTimeout(() => { highlightId.value = null; }, 2400);
    }
  }
  if (route.query.new === '1') openComposer();
});

function openComposer(): void {
  composing.value = true;
  void nextTick(() => {
    const el = document.querySelector<HTMLTextAreaElement>('.composer textarea');
    el?.focus();
  });
}

function closeComposer(): void {
  composing.value = false;
  draft.body = '';
  draft.occurrenceDate = '';
  draft.visibility = 'household';
  draft.audienceIds = [];
  draft.pinned = false;
  draftMentions.value = [];
}

async function submit(): Promise<void> {
  if (!draft.body.trim()) return;
  saving.value = true;
  try {
    await store.create({
      body: draft.body.trim(),
      occurrenceDate: draft.occurrenceDate || null,
      visibility: draft.visibility,
      audienceIds: draft.visibility === 'selected' ? draft.audienceIds : [],
      mentions: draftMentions.value,
      pinned: draft.pinned,
    });
    toast.success('留言已发布');
    closeComposer();
  } catch {
    toast.error('发布失败，请稍后重试');
  } finally {
    saving.value = false;
  }
}

function startReply(post: MessagePost): void {
  replyFor.value = replyFor.value === post.id ? null : post.id;
  replyBody.value = '';
  replyMentions.value = [];
}

async function submitReply(post: MessagePost): Promise<void> {
  if (!replyBody.value.trim()) return;
  replySaving.value = true;
  try {
    await store.reply(post.id, replyBody.value.trim(), replyMentions.value);
    toast.success('回复已发送');
    replyFor.value = null;
    replyBody.value = '';
    replyMentions.value = [];
  } catch {
    toast.error('回复失败，请稍后重试');
  } finally {
    replySaving.value = false;
  }
}

function startEdit(post: MessagePost): void {
  editing.value = post;
  editForm.body = post.body;
  editForm.occurrenceDate = post.occurrenceDate ?? '';
  editForm.visibility = post.visibility;
  editForm.audienceIds = [...post.audienceIds];
  editForm.pinned = post.pinned;
  editMentions.value = [...post.mentions];
}

async function submitEdit(): Promise<void> {
  if (!editing.value || !editForm.body.trim()) return;
  editSaving.value = true;
  try {
    await store.update(editing.value.id, {
      body: editForm.body.trim(),
      occurrenceDate: editForm.occurrenceDate || null,
      visibility: editForm.visibility,
      audienceIds: editForm.visibility === 'selected' ? editForm.audienceIds : [],
      mentions: editMentions.value,
      pinned: editForm.pinned,
    });
    toast.success('留言已更新');
    editing.value = null;
  } catch {
    toast.error('更新失败，请稍后重试');
  } finally {
    editSaving.value = false;
  }
}

async function togglePin(post: MessagePost): Promise<void> {
  try {
    await store.setPinned(post.id, !post.pinned);
  } catch {
    toast.error('操作失败，请稍后重试');
  }
}

async function confirmDelete(): Promise<void> {
  if (!deleting.value) return;
  deleteLoading.value = true;
  try {
    await store.remove(deleting.value.id);
    toast.success('留言已删除');
    deleting.value = null;
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
        <p class="at-caption-upper">交流</p>
        <h1 class="at-page__title">留言板</h1>
        <p class="at-page__subtitle">家人的留言、好消息和提醒，都留在这里。</p>
      </div>
      <div class="at-page__actions">
        <AppButton @click="composing ? closeComposer() : openComposer()">
          {{ composing ? '收起' : '写留言' }}
        </AppButton>
      </div>
    </header>

    <section v-if="composing" class="composer at-card">
      <div class="composer__head">
        <AppAvatar :name="auth.user?.displayName ?? ''" :size="34" :src="avatarUrl(auth.user?.avatarKey)" />
        <div>
          <strong>{{ auth.user?.displayName }}</strong>
          <p class="at-faint">写点什么给家里人</p>
        </div>
      </div>
      <MentionTextarea
        v-model="draft.body"
        v-model:mentions="draftMentions"
        :members="mentionMembers"
        :contacts="mentionContacts"
        placeholder="分享一件小事，输入 @ 可以提及家人或联系人"
        :rows="4"
        :maxlength="5000"
      />
      <div class="composer__row">
        <label class="inline-field">
          <span class="at-caption-upper">发生日期</span>
          <input v-model="draft.occurrenceDate" type="date" class="inline-field__input" :max="today" />
        </label>
        <label class="inline-field">
          <span class="at-caption-upper">可见范围</span>
          <AppSelect v-model="draft.visibility" :options="visibilityOptions" aria-label="可见范围" />
        </label>
        <label class="inline-check">
          <input v-model="draft.pinned" type="checkbox" />
          <span>置顶</span>
        </label>
      </div>
      <AudiencePicker
        v-if="draft.visibility === 'selected'"
        v-model="draft.audienceIds"
        :members="mentionMembers"
      />
      <div class="composer__footer">
        <span class="at-faint">支持 @ 提及，家人会收到通知</span>
        <div class="composer__buttons">
          <AppButton variant="ghost" @click="closeComposer">取消</AppButton>
          <AppButton :loading="saving" :disabled="!draft.body.trim()" @click="submit">发布</AppButton>
        </div>
      </div>
    </section>

    <div class="toolbar">
      <div class="toolbar__tabs" role="tablist" aria-label="留言筛选">
        <button
          v-for="tab in filterTabs"
          :key="tab.key"
          type="button"
          role="tab"
          class="toolbar__tab"
          :class="{ 'is-active': filter === tab.key }"
          :aria-selected="filter === tab.key"
          @click="filter = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>
      <label class="search">
        <AppIcon name="search" :size="16" />
        <input v-model="search" type="search" placeholder="搜索留言" aria-label="搜索留言" />
      </label>
    </div>

    <AppErrorState v-if="store.error" kind="server" compact @retry="reload" />
    <p v-else-if="store.loading && !store.items.length" class="at-faint loading">正在加载留言…</p>
    <AppEmptyState
      v-else-if="!store.items.length"
      icon="messages"
      title="还没有留言"
      description="写下第一句话，家人登录后就能看到。"
    >
      <AppButton @click="openComposer">写留言</AppButton>
    </AppEmptyState>

    <ul v-else class="feed">
      <li
        v-for="post in store.items"
        :id="`post-${post.id}`"
        :key="post.id"
        class="post at-card"
        :class="{ 'is-highlight': highlightId === post.id }"
      >
        <header class="post__head">
          <AppAvatar :name="post.authorName ?? ''" :size="38" />
          <div class="post__who">
            <div class="post__name-line">
              <strong>{{ post.authorName }}</strong>
              <span v-if="post.pinned" class="post__pin"><AppIcon name="pin" :size="12" />置顶</span>
              <span class="post__visibility">{{ visibilityLabel(post.visibility) }}</span>
            </div>
            <p class="at-faint">
              {{ formatTimestamp(post.createdAt) }}
              <template v-if="post.occurrenceDate"> · 记于 {{ formatShortDate(post.occurrenceDate) }}</template>
              <template v-if="post.editedAt"> · 已编辑</template>
              <template v-if="post.closedAt"> · 已关闭</template>
            </p>
          </div>
          <div class="post__actions">
            <AppIconButton
              icon="pin"
              :label="post.pinned ? '取消置顶' : '置顶'"
              size="sm"
              :active="post.pinned"
              @click="togglePin(post)"
            />
            <AppIconButton v-if="canEdit(post)" icon="edit" label="编辑" size="sm" @click="startEdit(post)" />
            <AppIconButton
              v-if="canManage(post)"
              icon="trash"
              label="删除"
              size="sm"
              @click="deleting = post"
            />
          </div>
        </header>

        <p class="post__body">
          <MentionRenderer :text="post.body" :mentions="post.mentions" :contacts="mentionContacts" :members="family.activeMembers" />
        </p>

        <div class="post__footer">
          <button type="button" class="post__reply-btn" @click="startReply(post)">
            <AppIcon name="messages" :size="15" />
            {{ post.replies.length ? `${post.replies.length} 条回复` : '回复' }}
          </button>
        </div>

        <ol v-if="post.replies.length" class="replies">
          <li v-for="reply in post.replies" :key="reply.id" class="reply">
            <AppAvatar :name="reply.authorName ?? ''" :size="26" />
            <div class="reply__body">
              <p class="reply__meta">
                <strong>{{ reply.authorName }}</strong>
                <span class="at-faint">{{ formatTimestamp(reply.createdAt) }}</span>
              </p>
              <p class="reply__text">{{ reply.body }}</p>
            </div>
          </li>
        </ol>

        <div v-if="replyFor === post.id" class="reply-box">
          <MentionTextarea
            v-model="replyBody"
            v-model:mentions="replyMentions"
            :members="mentionMembers"
            :contacts="mentionContacts"
            placeholder="写下回复，输入 @ 可以提及家人或联系人"
            :rows="2"
            :maxlength="2000"
          />
          <div class="reply-box__actions">
            <AppButton variant="ghost" size="sm" @click="replyFor = null">取消</AppButton>
            <AppButton size="sm" :loading="replySaving" :disabled="!replyBody.trim()" @click="submitReply(post)">
              发送
            </AppButton>
          </div>
        </div>
      </li>
    </ul>

    <AppDialog
      :model-value="editing !== null"
      title="编辑留言"
      size="md"
      @update:model-value="editing = null"
    >
      <div class="dialog-form">
        <MentionTextarea
          v-model="editForm.body"
          v-model:mentions="editMentions"
          :members="mentionMembers"
          :contacts="mentionContacts"
          :rows="5"
          :maxlength="5000"
        />
        <div class="composer__row">
          <label class="inline-field">
            <span class="at-caption-upper">发生日期</span>
            <input v-model="editForm.occurrenceDate" type="date" class="inline-field__input" :max="today" />
          </label>
          <label class="inline-field">
            <span class="at-caption-upper">可见范围</span>
            <AppSelect v-model="editForm.visibility" :options="visibilityOptions" aria-label="可见范围" />
          </label>
          <label class="inline-check">
            <input v-model="editForm.pinned" type="checkbox" />
            <span>置顶</span>
          </label>
        </div>
        <AudiencePicker
          v-if="editForm.visibility === 'selected'"
          v-model="editForm.audienceIds"
          :members="mentionMembers"
        />
      </div>
      <template #footer>
        <AppButton variant="ghost" @click="editing = null">取消</AppButton>
        <AppButton :loading="editSaving" :disabled="!editForm.body.trim()" @click="submitEdit">保存</AppButton>
      </template>
    </AppDialog>

    <AppConfirmDialog
      :model-value="deleting !== null"
      title="删除留言"
      message="删除后，这条留言及其回复和提及通知都会消失，且无法恢复。"
      confirm-text="删除"
      danger
      :loading="deleteLoading"
      @update:model-value="deleting = null"
      @confirm="confirmDelete"
    />
  </div>
</template>

<style scoped>
.composer {
  padding: var(--at-space-5);
  margin-bottom: var(--at-space-5);
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  border-radius: var(--at-radius-xl);
}

.composer__head {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
}

.composer__head p {
  font-size: var(--at-text-micro);
  margin-top: 2px;
}

.composer__row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: var(--at-space-4);
}

.inline-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.inline-field__input {
  height: var(--at-control-height);
  padding: 0 var(--at-space-3);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface);
  color: var(--at-text-primary);
  font-size: var(--at-text-caption);
}

.inline-check {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: var(--at-control-height);
  font-size: var(--at-text-caption);
  color: var(--at-color-body);
}

.composer__footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
  font-size: var(--at-text-micro);
}

.composer__buttons {
  display: flex;
  gap: var(--at-space-2);
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
  margin-bottom: var(--at-space-4);
}

.toolbar__tabs {
  display: inline-flex;
  padding: 3px;
  background: var(--at-bg-surface-strong);
  border-radius: var(--at-radius-pill);
}

.toolbar__tab {
  padding: 7px 16px;
  border-radius: var(--at-radius-pill);
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  transition: background var(--at-duration-fast) var(--at-ease), color var(--at-duration-fast) var(--at-ease);
}

.toolbar__tab.is-active {
  background: var(--at-bg-surface);
  color: var(--at-text-primary);
  box-shadow: var(--at-shadow-panel);
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

.feed {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  list-style: none;
}

.post {
  padding: var(--at-space-5);
  border-radius: var(--at-radius-xl);
  transition: border-color var(--at-duration-base) var(--at-ease);
}

.post.is-highlight {
  border-color: var(--at-color-ink);
  box-shadow: var(--at-shadow-card-hover), inset 0 1px 0 var(--at-card-highlight);
}

.post__head {
  display: flex;
  align-items: flex-start;
  gap: var(--at-space-3);
}

.post__who {
  min-width: 0;
  flex: 1;
}

.post__name-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--at-space-2);
  font-size: var(--at-text-caption);
}

.post__pin {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 8px;
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface-strong);
  font-size: 11px;
  color: var(--at-color-body-strong);
}

.post__visibility {
  font-size: 11px;
  color: var(--at-color-muted);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-pill);
  padding: 1px 8px;
}

.post__who .at-faint {
  margin-top: 2px;
  font-size: var(--at-text-micro);
}

.post__actions {
  display: flex;
  gap: 2px;
}

.post__body {
  margin: var(--at-space-3) 0;
  font-size: var(--at-text-body-sm);
  line-height: 1.75;
  color: var(--at-color-body-strong);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.post__footer {
  display: flex;
  align-items: center;
}

.post__reply-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.post__reply-btn:hover {
  color: var(--at-text-primary);
}

.replies {
  margin-top: var(--at-space-3);
  padding-top: var(--at-space-3);
  border-top: 1px solid var(--at-color-hairline-soft);
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  list-style: none;
}

.reply {
  display: flex;
  gap: var(--at-space-2);
}

.reply__body {
  flex: 1;
  min-width: 0;
  padding: var(--at-space-2) var(--at-space-3);
  background: var(--at-bg-canvas-soft);
  border-radius: var(--at-radius-md);
}

.reply__meta {
  display: flex;
  align-items: baseline;
  gap: var(--at-space-2);
  font-size: var(--at-text-micro);
}

.reply__text {
  margin-top: 2px;
  font-size: var(--at-text-caption);
  line-height: 1.6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.reply-box {
  margin-top: var(--at-space-3);
}

.reply-box__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--at-space-2);
  margin-top: var(--at-space-2);
}

.dialog-form {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
}

@media (max-width: 639px) {
  .at-page__header {
    align-items: flex-start;
  }

  .post {
    padding: var(--at-space-4);
  }

  .search {
    flex: 1;
  }

  .search input {
    width: 100%;
    min-width: 0;
  }

  .toolbar {
    align-items: stretch;
    flex-direction: column-reverse;
  }

  .toolbar__tabs {
    align-self: flex-start;
  }
}
</style>
