<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import type { TaskTemplate } from '@/types';
import { useFamilyStore } from '@/stores/family';
import { useTasksStore } from '@/stores/tasks';
import { useToastStore } from '@/stores/toast';
import { ApiError } from '@/api/client';
import { parseRecurrence, priorityLabels } from '@/utils/task';
import AppAvatar from '@/components/base/AppAvatar.vue';
import { avatarUrl } from '@/utils/avatar';
import AppButton from '@/components/base/AppButton.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppField from '@/components/base/AppField.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import AppSwitch from '@/components/base/AppSwitch.vue';
import AppTextarea from '@/components/base/AppTextarea.vue';

const props = defineProps<{
  modelValue: boolean;
  templates: TaskTemplate[];
}>();

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  use: [template: TaskTemplate];
  changed: [];
}>();

const tasks = useTasksStore();
const family = useFamilyStore();
const toast = useToastStore();

const editing = ref<TaskTemplate | null>(null);
const creating = ref(false);
const saving = ref(false);
const error = ref('');
const pendingDelete = ref<TaskTemplate | null>(null);

const form = reactive({
  name: '',
  title: '',
  description: '',
  priority: 'normal',
  dueTime: '',
  assignees: [] as number[],
  subtasks: '',
  reminder: '',
  enabled: true,
  freq: '',
  interval: '1',
});

const activeMembers = computed(() => family.members.filter((m) => m.status === 'active'));
const showForm = computed(() => creating.value || editing.value !== null);
const dialogTitle = computed(() => (showForm.value ? (editing.value ? '编辑模板' : '新建模板') : '待办模板'));

const priorityOptions = [
  { value: 'low', label: '低' },
  { value: 'normal', label: '普通' },
  { value: 'high', label: '高' },
  { value: 'urgent', label: '紧急' },
];

const reminderOptions = [
  { value: '', label: '不提醒' },
  { value: '0', label: '准时提醒' },
  { value: '30', label: '提前 30 分钟' },
  { value: '60', label: '提前 1 小时' },
  { value: '1440', label: '提前 1 天' },
];

const freqOptions = [
  { value: '', label: '不重复' },
  { value: 'daily', label: '每天' },
  { value: 'weekly', label: '每周' },
  { value: 'monthly', label: '每月' },
  { value: 'yearly', label: '每年' },
];

function resetForm(): void {
  Object.assign(form, {
    name: '',
    title: '',
    description: '',
    priority: 'normal',
    dueTime: '',
    assignees: [],
    subtasks: '',
    reminder: '',
    enabled: true,
    freq: '',
    interval: '1',
  });
  error.value = '';
}

function openCreate(): void {
  editing.value = null;
  creating.value = true;
  resetForm();
}

function openEdit(template: TaskTemplate): void {
  editing.value = template;
  creating.value = false;
  error.value = '';
  Object.assign(form, {
    name: template.name,
    title: template.title,
    description: template.description ?? '',
    priority: template.priority ?? 'normal',
    dueTime: template.defaultDueTime ?? '',
    assignees: [...template.defaultAssigneeIds],
    subtasks: template.subtaskTemplate.join('\n'),
    reminder: template.reminderOffset !== null && template.reminderOffset !== undefined ? String(template.reminderOffset) : '',
    enabled: template.enabled,
    freq: parseRecurrence(template.recurrenceRule)?.freq ?? '',
    interval: String(parseRecurrence(template.recurrenceRule)?.interval ?? 1),
  });
}

function backToList(): void {
  creating.value = false;
  editing.value = null;
  error.value = '';
}

function toggleAssignee(id: number): void {
  form.assignees = form.assignees.includes(id) ? form.assignees.filter((v) => v !== id) : [...form.assignees, id];
}

async function save(): Promise<void> {
  error.value = '';
  if (!form.name.trim()) {
    error.value = '请填写模板名称';
    return;
  }
  saving.value = true;
  try {
    const rule = form.freq
      ? JSON.stringify({ freq: form.freq, interval: Math.max(1, Number(form.interval) || 1), until: '', anchor: 'due' })
      : null;
    await tasks.saveTemplate({
      name: form.name.trim(),
      title: form.title.trim() || form.name.trim(),
      description: form.description.trim() || null,
      priority: form.priority,
      defaultAssigneeIds: form.assignees,
      defaultDueTime: form.dueTime || null,
      subtaskTemplate: form.subtasks.split('\n').map((s) => s.trim()).filter(Boolean),
      reminderOffset: form.reminder === '' ? null : Number(form.reminder),
      recurrenceRule: rule,
      enabled: form.enabled,
    }, editing.value?.id);
    toast.success(editing.value ? '模板已更新' : '模板已创建');
    backToList();
    emit('changed');
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '保存失败，请稍后重试';
  } finally {
    saving.value = false;
  }
}

async function toggleEnabled(template: TaskTemplate, value: boolean): Promise<void> {
  try {
    await tasks.saveTemplate({
      name: template.name,
      title: template.title,
      description: template.description,
      priority: template.priority,
      defaultAssigneeIds: template.defaultAssigneeIds,
      defaultDueTime: template.defaultDueTime,
      subtaskTemplate: template.subtaskTemplate,
      reminderOffset: template.reminderOffset,
      recurrenceRule: template.recurrenceRule,
      enabled: value,
    }, template.id);
    emit('changed');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '更新模板失败');
  }
}

async function confirmDelete(): Promise<void> {
  if (!pendingDelete.value) return;
  try {
    await tasks.deleteTemplate(pendingDelete.value.id);
    toast.success('模板已删除');
    pendingDelete.value = null;
    emit('changed');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除模板失败');
  }
}
</script>

<template>
  <AppDialog
    :model-value="modelValue"
    :title="dialogTitle"
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-if="showForm" class="tpl-form">
      <div class="tpl-form__grid">
        <AppField label="模板名称" for-id="tpl-name" required>
          <AppInput id="tpl-name" v-model="form.name" :maxlength="60" placeholder="如：周末采购" />
        </AppField>
        <AppField label="默认标题" for-id="tpl-title">
          <AppInput id="tpl-title" v-model="form.title" :maxlength="120" placeholder="留空则使用模板名称" />
        </AppField>
      </div>
      <AppField label="默认说明" for-id="tpl-desc">
        <AppTextarea id="tpl-desc" v-model="form.description" :rows="2" :maxlength="2000" />
      </AppField>
      <div class="tpl-form__grid">
        <AppField label="优先级" for-id="tpl-priority">
          <AppSelect id="tpl-priority" v-model="form.priority" :options="priorityOptions" />
        </AppField>
        <AppField label="默认时间" for-id="tpl-time">
          <AppInput id="tpl-time" v-model="form.dueTime" type="time" />
        </AppField>
      </div>
      <AppField label="指派给">
        <div class="chips">
          <button
            v-for="member in activeMembers"
            :key="member.id"
            type="button"
            class="chip"
            :class="{ 'is-active': form.assignees.includes(member.id) }"
            @click="toggleAssignee(member.id)"
          >
            <AppAvatar :name="member.displayName" :size="20" :src="avatarUrl(member.avatarKey)" />{{ member.displayName }}
          </button>
        </div>
      </AppField>
      <div class="tpl-form__grid">
        <AppField label="提醒" for-id="tpl-reminder">
          <AppSelect id="tpl-reminder" v-model="form.reminder" :options="reminderOptions" />
        </AppField>
        <AppField label="重复" for-id="tpl-freq">
          <AppSelect id="tpl-freq" v-model="form.freq" :options="freqOptions" />
        </AppField>
      </div>
      <AppField label="子任务" hint="每行一个子任务">
        <AppTextarea v-model="form.subtasks" :rows="3" placeholder="列清单&#10;买水果" />
      </AppField>
      <AppSwitch v-model="form.enabled" label="启用模板" description="停用后不在新建待办时显示" />
      <p v-if="error" class="form-error" role="alert">{{ error }}</p>
    </div>

    <div v-else class="tpl-list">
      <div class="tpl-list__head">
        <p class="muted">模板可预设标题、优先级、提醒与子任务，减少重复填写。</p>
        <AppButton size="sm" @click="openCreate"><AppIcon name="plus" :size="15" />新建模板</AppButton>
      </div>
      <AppEmptyState v-if="!props.templates.length" title="还没有模板" description="把常用的家务或采购做成模板" icon="tasks">
        <AppButton size="sm" @click="openCreate">新建模板</AppButton>
      </AppEmptyState>
      <ul v-else class="tpl-items">
        <li v-for="template in props.templates" :key="template.id" class="tpl-item">
          <div class="tpl-item__main">
            <div class="tpl-item__title">
              <AppIcon name="tasks" :size="16" />
              <span>{{ template.name }}</span>
              <AppSwitch
                :model-value="template.enabled"
                :label="`启用 ${template.name}`"
                @update:model-value="toggleEnabled(template, $event)"
              />
            </div>
            <p class="tpl-item__meta">
              {{ template.title }}
              <template v-if="template.priority"> · {{ priorityLabels[template.priority] }}</template>
              <template v-if="template.subtaskTemplate.length"> · {{ template.subtaskTemplate.length }} 个子任务</template>
              <template v-if="template.defaultDueTime"> · {{ template.defaultDueTime }}</template>
            </p>
          </div>
          <div class="tpl-item__actions">
            <AppButton size="sm" variant="outline" :disabled="!template.enabled" @click="emit('use', template)">使用</AppButton>
            <button type="button" class="icon-btn" aria-label="编辑模板" @click="openEdit(template)">
              <AppIcon name="edit" :size="16" />
            </button>
            <button type="button" class="icon-btn" aria-label="删除模板" @click="pendingDelete = template">
              <AppIcon name="trash" :size="16" />
            </button>
          </div>
        </li>
      </ul>
    </div>

    <template #footer>
      <template v-if="showForm">
        <AppButton variant="ghost" @click="backToList">返回</AppButton>
        <AppButton :loading="saving" @click="save">{{ editing ? '保存模板' : '创建模板' }}</AppButton>
      </template>
      <AppButton v-else variant="ghost" @click="emit('update:modelValue', false)">关闭</AppButton>
    </template>
  </AppDialog>

  <AppConfirmDialog
    :model-value="pendingDelete !== null"
    title="删除模板"
    :message="`确定删除「${pendingDelete?.name ?? ''}」吗？已创建的待办不受影响。`"
    danger
    confirm-text="删除"
    @update:model-value="pendingDelete = null"
    @confirm="confirmDelete"
  />
</template>

<style scoped>
.tpl-form {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.tpl-form__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--at-space-3);
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 34px;
  padding: 4px 12px;
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-pill);
  font-size: var(--at-text-caption);
  color: var(--at-color-body);
}

.chip.is-active {
  border-color: var(--at-color-primary);
  background: var(--at-color-primary);
  color: var(--at-text-on-primary);
}

.tpl-list {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.tpl-list__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
}

.muted {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.tpl-items {
  display: flex;
  flex-direction: column;
}

.tpl-item {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  padding: var(--at-space-3) 0;
  border-bottom: 1px solid var(--at-color-hairline);
}

.tpl-item:last-child {
  border-bottom: 0;
}

.tpl-item__main {
  flex: 1;
  min-width: 0;
}

.tpl-item__title {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  font-size: var(--at-text-body-sm);
  color: var(--at-color-ink);
}

.tpl-item__title :deep(.at-switch-row) {
  padding: 0;
  margin-left: var(--at-space-2);
}

.tpl-item__title :deep(.at-switch-row__text) {
  display: none;
}

.tpl-item__meta {
  margin-top: 3px;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tpl-item__actions {
  display: flex;
  align-items: center;
  gap: var(--at-space-1);
}

.icon-btn {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: var(--at-radius-md);
  color: var(--at-color-muted);
}

.icon-btn:hover {
  background: var(--at-bg-surface-strong);
  color: var(--at-color-ink);
}

.form-error {
  padding: 10px var(--at-space-3);
  border-radius: var(--at-radius-md);
  background: var(--at-color-danger-soft);
  color: var(--at-color-danger);
  font-size: var(--at-text-caption);
}

@media (max-width: 639px) {
  .tpl-form__grid {
    grid-template-columns: 1fr;
  }

  .tpl-list__head {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
