<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { Task, TaskMedia } from '@/types';
import { useAuthStore } from '@/stores/auth';
import { useFamilyStore } from '@/stores/family';
import { useTasksStore } from '@/stores/tasks';
import { useToastStore } from '@/stores/toast';
import { ApiError, api } from '@/api/client';
import { dueLabel, priorityLabels, recurrenceSummary, statusLabels, visibilityLabels } from '@/utils/task';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppButton from '@/components/base/AppButton.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import AppStatusTag from '@/components/base/AppStatusTag.vue';

const props = defineProps<{
  modelValue: boolean;
  task: Task | null;
  today: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  changed: [task: Task];
  edit: [task: Task];
  remove: [task: Task];
  complete: [task: Task, actualUserId: number];
  undo: [task: Task];
}>();

const auth = useAuthStore();
const family = useFamilyStore();
const tasks = useTasksStore();
const toast = useToastStore();

const media = ref<TaskMedia[]>([]);
const mediaLoading = ref(false);
const uploading = ref(false);
const uploadProgress = ref(0);
const uploadError = ref('');
const lightbox = ref<TaskMedia | null>(null);
const confirmDelete = ref(false);
const completeOpen = ref(false);
const actualUserId = ref('');
const togglingSubtask = ref<number | null>(null);
const fileInput = ref<HTMLInputElement | null>(null);
let xhr: XMLHttpRequest | null = null;

const completed = computed(() => props.task?.status === 'completed');
const due = computed(() => (props.task ? dueLabel(props.task, props.today) : { text: '', tone: 'muted' as const }));
const repeat = computed(() => recurrenceSummary(props.task?.recurrenceRule));
const doneSubtasks = computed(() => props.task?.subtasks.filter((s) => s.done).length ?? 0);
const activeMembers = computed(() => family.members.filter((m) => m.status === 'active'));
const actualOptions = computed(() => activeMembers.value.map((m) => ({ value: String(m.id), label: m.displayName })));
const canEdit = computed(() => Boolean(props.task && !props.task.archivedAt));

watch(() => props.modelValue, (open) => {
  if (!open) {
    abortUpload();
    lightbox.value = null;
    confirmDelete.value = false;
    completeOpen.value = false;
    uploadError.value = '';
    return;
  }
  actualUserId.value = String(auth.user?.id ?? '');
  void loadMedia();
});

watch(() => props.task?.id, () => {
  if (props.modelValue) {
    completeOpen.value = false;
    void loadMedia();
  }
});

async function loadMedia(): Promise<void> {
  if (!props.task) return;
  mediaLoading.value = true;
  try {
    const res = await api.get<{ items: TaskMedia[] }>(`/tasks/${props.task.id}/media`);
    media.value = res.items;
  } catch {
    media.value = [];
  } finally {
    mediaLoading.value = false;
  }
}

async function toggleSubtask(subtaskId: number, done: boolean): Promise<void> {
  if (!props.task) return;
  togglingSubtask.value = subtaskId;
  try {
    const updated = await tasks.toggleSubtask(props.task.id, subtaskId, done);
    emit('changed', updated);
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '更新子任务失败');
  } finally {
    togglingSubtask.value = null;
  }
}

function pickFile(): void {
  fileInput.value?.click();
}

function onFileChange(event: Event): void {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (file) upload(file);
}

function upload(file: File): void {
  if (!props.task) return;
  uploadError.value = '';
  uploading.value = true;
  uploadProgress.value = 0;
  const form = new FormData();
  form.append('file', file);
  xhr = new XMLHttpRequest();
  xhr.open('POST', `/api/v1/tasks/${props.task.id}/media`);
  xhr.upload.onprogress = (event) => {
    if (event.lengthComputable) uploadProgress.value = Math.round((event.loaded / event.total) * 100);
  };
  xhr.onload = () => {
    const status = xhr?.status ?? 0;
    const text = xhr?.responseText ?? '';
    uploading.value = false;
    xhr = null;
    if (status === 201 || status === 200) {
      toast.success('附件已上传');
      void loadMedia();
      if (props.task) emit('changed', props.task);
      return;
    }
    uploadError.value = parseXHRError(text) || '上传失败，请稍后重试';
  };
  xhr.onerror = () => {
    uploading.value = false;
    xhr = null;
    uploadError.value = '网络连接失败，请检查网络后重试';
  };
  xhr.onabort = () => {
    uploading.value = false;
    xhr = null;
  };
  xhr.send(form);
}

function parseXHRError(text: string): string {
  try {
    const parsed = JSON.parse(text) as { error?: { message?: string } };
    return parsed.error?.message ?? '';
  } catch {
    return '';
  }
}

function abortUpload(): void {
  if (xhr) {
    xhr.abort();
    xhr = null;
    uploading.value = false;
  }
}

async function removeMedia(item: TaskMedia): Promise<void> {
  try {
    await api.del(`/media/${item.id}`);
    media.value = media.value.filter((m) => m.id !== item.id);
    toast.success('附件已删除');
    if (props.task) emit('changed', props.task);
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除附件失败');
  }
}

function mediaURL(item: TaskMedia, thumb = false): string {
  return `/api/v1/media/${item.id}${thumb ? '/thumbnail' : ''}`;
}

function confirmComplete(): void {
  if (!props.task) return;
  const actual = Number(actualUserId.value) || auth.user?.id || 0;
  emit('complete', props.task, actual);
  completeOpen.value = false;
}

function requestDelete(): void {
  if (!props.task) return;
  emit('remove', props.task);
  confirmDelete.value = false;
}
</script>

<template>
  <AppDialog
    :model-value="modelValue"
    :title="task?.title ?? '待办详情'"
    size="lg"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-if="task" class="detail">
      <div class="detail__tags">
        <AppStatusTag :tone="completed ? 'success' : task.status === 'in_progress' ? 'brand' : 'neutral'">
          {{ statusLabels[task.status] }}
        </AppStatusTag>
        <AppStatusTag v-if="task.priority === 'high' || task.priority === 'urgent'" tone="danger">
          {{ priorityLabels[task.priority] }}
        </AppStatusTag>
        <AppStatusTag v-if="task.archivedAt" tone="neutral">已归档</AppStatusTag>
        <AppStatusTag v-if="task.visibility !== 'household'" tone="brand">{{ visibilityLabels[task.visibility] }}</AppStatusTag>
      </div>

      <dl class="detail__meta">
        <div>
          <dt>截止</dt>
          <dd :class="`is-${due.tone}`">{{ task.dueDate ? `${task.dueDate}${task.dueTime ? ` ${task.dueTime}` : ''}` : '未设置' }}</dd>
        </div>
        <div>
          <dt>指派</dt>
          <dd class="detail__people">
            <template v-if="task.assignees.length">
              <span v-for="person in task.assignees" :key="person.id" class="detail__person">
                <AppAvatar :name="person.displayName" :size="22" />{{ person.displayName }}
              </span>
            </template>
            <span v-else class="muted">未指派</span>
          </dd>
        </div>
        <div v-if="repeat">
          <dt>重复</dt>
          <dd>{{ repeat }}</dd>
        </div>
        <div v-if="task.templateId">
          <dt>模板</dt>
          <dd>来自模板</dd>
        </div>
      </dl>

      <p v-if="task.description" class="detail__desc">{{ task.description }}</p>

      <section v-if="task.subtasks.length" class="detail__section">
        <div class="detail__section-head">
          <h3>子任务</h3>
          <span class="muted">{{ doneSubtasks }}/{{ task.subtasks.length }}</span>
        </div>
        <label v-for="subtask in task.subtasks" :key="subtask.id" class="subtask">
          <input
            type="checkbox"
            class="subtask__input"
            :checked="subtask.done"
            :disabled="togglingSubtask === subtask.id"
            @change="toggleSubtask(subtask.id, ($event.target as HTMLInputElement).checked)"
          />
          <span class="subtask__box"><AppIcon name="check" :size="12" /></span>
          <span :class="{ 'is-done': subtask.done }">{{ subtask.title }}</span>
        </label>
      </section>

      <section v-if="task.completion" class="detail__section detail__completion">
        <div class="detail__section-head"><h3>完成记录</h3></div>
        <p>
          {{ task.completion.doneByName || '家人' }} 勾选
          <template v-if="task.completion.actualName && task.completion.actualName !== task.completion.doneByName">
            · 实际由 {{ task.completion.actualName }} 完成
          </template>
          · {{ new Date(task.completion.completedAt * 1000).toLocaleString('zh-CN', { hour12: false }) }}
        </p>
      </section>

      <section class="detail__section">
        <div class="detail__section-head">
          <h3>随手记录</h3>
          <span class="muted">{{ media.length }} 个附件</span>
        </div>
        <div v-if="mediaLoading" class="muted">加载中…</div>
        <template v-else>
          <div v-if="media.length" class="media">
            <figure v-for="item in media" :key="item.id" class="media__item">
              <button
                v-if="item.mediaType === 'image'"
                type="button"
                class="media__thumb"
                @click="lightbox = item"
              >
                <img :src="mediaURL(item, true)" :alt="item.originalName ?? '图片附件'" loading="lazy" />
              </button>
              <audio v-else-if="item.mediaType === 'audio'" controls :src="mediaURL(item)" class="media__audio" />
              <video v-else controls :src="mediaURL(item)" class="media__video" />
              <figcaption>
                <span class="media__name">{{ item.originalName ?? '附件' }}</span>
                <button type="button" class="media__remove" aria-label="删除附件" @click="removeMedia(item)">
                  <AppIcon name="trash" :size="13" />
                </button>
              </figcaption>
            </figure>
          </div>
          <AppEmptyState
            v-else-if="!canEdit"
            title="没有附件"
            description="完成或归档的待办不能再添加记录"
            icon="image"
          />
          <div v-else class="upload">
            <input ref="fileInput" type="file" accept="image/*,audio/*,video/*" class="upload__input" @change="onFileChange" />
            <AppButton variant="outline" size="sm" :disabled="uploading" @click="pickFile">
              <AppIcon name="upload" :size="15" />上传照片 / 语音 / 视频
            </AppButton>
            <div v-if="uploading" class="upload__progress">
              <span :style="{ width: `${uploadProgress}%` }" />
            </div>
            <button v-if="uploading" type="button" class="link-btn" @click="abortUpload">取消上传</button>
            <p v-if="uploadError" class="upload__error" role="alert">{{ uploadError }}</p>
          </div>
        </template>
      </section>

      <div v-if="completeOpen" class="complete-box">
        <p class="complete-box__title">确认完成</p>
        <AppSelect v-model="actualUserId" :options="actualOptions" aria-label="实际完成者" />
        <p class="complete-box__hint">
          勾选者为你本人；可选择实际完成者。{{ repeat ? '完成后将自动生成下一次待办。' : '' }}
        </p>
        <div class="complete-box__actions">
          <AppButton variant="ghost" size="sm" @click="completeOpen = false">取消</AppButton>
          <AppButton size="sm" @click="confirmComplete">确认完成</AppButton>
        </div>
      </div>
    </div>

    <template #footer>
      <AppButton variant="ghost" @click="emit('update:modelValue', false)">关闭</AppButton>
      <div class="detail__footer">
        <AppButton v-if="!completed && !task?.archivedAt" variant="outline" @click="completeOpen = true">标记完成</AppButton>
        <AppButton v-if="completed" variant="outline" @click="task && emit('undo', task)">撤销完成</AppButton>
        <AppButton v-if="canEdit" variant="ghost" @click="task && emit('edit', task)">编辑</AppButton>
        <AppButton v-if="canEdit" variant="danger" @click="confirmDelete = true">删除</AppButton>
      </div>
    </template>
  </AppDialog>

  <AppConfirmDialog
    v-model="confirmDelete"
    title="删除待办"
    :message="task?.seriesId ? '这会删除当前待办；已完成的历史记录会保留。' : '删除后无法恢复，确定删除这条待办吗？'"
    danger
    confirm-text="删除"
    @confirm="requestDelete"
  />

  <Teleport to="body">
    <div v-if="lightbox" class="lightbox" role="dialog" aria-modal="true" aria-label="查看图片" @click.self="lightbox = null">
      <button type="button" class="lightbox__close" aria-label="关闭" @click="lightbox = null">
        <AppIcon name="close" :size="22" />
      </button>
      <img :src="mediaURL(lightbox)" :alt="lightbox.originalName ?? '图片附件'" />
    </div>
  </Teleport>
</template>

<style scoped>
.detail {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-5);
}

.detail__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.detail__meta {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: var(--at-space-4);
  padding: var(--at-space-4);
  background: var(--at-bg-canvas-soft);
  border-radius: var(--at-radius-lg);
}

.detail__meta dt {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
  margin-bottom: 2px;
}

.detail__meta dd {
  font-size: var(--at-text-caption);
  color: var(--at-text-primary);
}

.detail__meta dd.is-overdue {
  color: var(--at-color-danger);
}

.detail__people {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.detail__person {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.detail__desc {
  white-space: pre-wrap;
  line-height: 1.75;
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body);
}

.detail__section {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  padding-top: var(--at-space-4);
  border-top: 1px solid var(--at-color-hairline);
}

.detail__section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.detail__section-head h3 {
  font-size: var(--at-text-caption);
  font-weight: 600;
}

.detail__completion p {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.subtask {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  padding: 6px 0;
  font-size: var(--at-text-body-sm);
  cursor: pointer;
}

.subtask__input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.subtask__box {
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  border: 1.5px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-xs);
  color: transparent;
  transition: background var(--at-duration-fast) var(--at-ease);
}

.subtask__input:checked + .subtask__box {
  background: var(--at-color-primary);
  border-color: var(--at-color-primary);
  color: var(--at-text-on-primary);
}

.subtask .is-done {
  color: var(--at-color-muted-soft);
  text-decoration: line-through;
}

.media {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: var(--at-space-3);
}

.media__item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.media__thumb {
  display: block;
  aspect-ratio: 1;
  overflow: hidden;
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface-strong);
}

.media__thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.media__audio {
  width: 100%;
  height: 36px;
}

.media__video {
  width: 100%;
  border-radius: var(--at-radius-md);
  background: var(--at-bg-dark);
}

.media__item figcaption {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.media__name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.media__remove {
  color: var(--at-color-muted-soft);
}

.media__remove:hover {
  color: var(--at-color-danger);
}

.upload {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  align-items: flex-start;
}

.upload__input {
  display: none;
}

.upload__progress {
  width: 100%;
  height: 4px;
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface-strong);
  overflow: hidden;
}

.upload__progress span {
  display: block;
  height: 100%;
  background: var(--at-color-primary);
  transition: width 0.2s ease;
}

.upload__error {
  font-size: var(--at-text-micro);
  color: var(--at-color-danger);
}

.complete-box {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  padding: var(--at-space-4);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-lg);
  background: var(--at-bg-canvas-soft);
}

.complete-box__title {
  font-size: var(--at-text-caption);
  font-weight: 600;
}

.complete-box__hint {
  font-size: 12px;
  color: var(--at-color-muted);
  line-height: 1.6;
}

.complete-box__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--at-space-2);
}

.detail__footer {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
  justify-content: flex-end;
}

.link-btn {
  font-size: var(--at-text-micro);
  color: var(--at-color-danger);
}

.lightbox {
  position: fixed;
  inset: 0;
  z-index: var(--at-z-overlay);
  display: grid;
  place-items: center;
  padding: var(--at-space-6);
  background: rgba(12, 10, 9, 0.88);
}

.lightbox img {
  max-width: 100%;
  max-height: 88vh;
  border-radius: var(--at-radius-md);
}

.lightbox__close {
  position: absolute;
  top: var(--at-space-4);
  right: var(--at-space-4);
  width: 44px;
  height: 44px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
}

.muted {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}
</style>
