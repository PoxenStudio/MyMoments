<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import type { Task, TaskTemplate } from '@/types';
import { useFamilyStore } from '@/stores/family';
import { useTasksStore, type SubtaskPayload, type TaskPayload } from '@/stores/tasks';
import { ApiError } from '@/api/client';
import { parseRecurrence, taskColors, visibilityLabels } from '@/utils/task';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppButton from '@/components/base/AppButton.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppField from '@/components/base/AppField.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import AppTextarea from '@/components/base/AppTextarea.vue';

const props = withDefaults(defineProps<{
  modelValue: boolean;
  task?: Task | null;
  initialTemplate?: TaskTemplate | null;
  templates?: TaskTemplate[];
  defaultDate?: string | null;
}>(), {
  task: null,
  initialTemplate: null,
  templates: () => [],
  defaultDate: null,
});

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  saved: [task: Task];
}>();

const tasks = useTasksStore();
const family = useFamilyStore();

interface SubtaskRow {
  id?: number;
  title: string;
  done: boolean;
}

interface FormState {
  title: string;
  description: string;
  priority: string;
  dueDate: string;
  dueTime: string;
  visibility: string;
  color: string;
  assigneeIds: number[];
  subtasks: SubtaskRow[];
  reminder: string;
  freq: string;
  interval: string;
  weekdays: number[];
  until: string;
  anchor: string;
  templateId: number | null;
}

const form = reactive<FormState>(blankForm());
const moreOpen = ref(false);
const saving = ref(false);
const error = ref('');

const isEdit = computed(() => Boolean(props.task));
const title = computed(() => (isEdit.value ? '编辑待办' : '新建待办'));
const activeMembers = computed(() => family.members.filter((m) => m.status === 'active'));

const templateOptions = computed(() => [
  { value: '', label: '不使用模板' },
  ...props.templates.filter((t) => t.enabled).map((t) => ({ value: String(t.id), label: t.name })),
]);

const priorityOptions = [
  { value: 'low', label: '低' },
  { value: 'normal', label: '普通' },
  { value: 'high', label: '高' },
  { value: 'urgent', label: '紧急' },
];

const visibilityOptions = [
  { value: 'household', label: visibilityLabels.household },
  { value: 'selected', label: visibilityLabels.selected },
  { value: 'private', label: visibilityLabels.private },
];

const freqOptions = [
  { value: '', label: '不重复' },
  { value: 'daily', label: '每天' },
  { value: 'weekly', label: '每周' },
  { value: 'monthly', label: '每月' },
  { value: 'yearly', label: '每年' },
];

const anchorOptions = [
  { value: 'due', label: '按截止日期' },
  { value: 'completion', label: '按完成日期' },
];

const reminderOptions = [
  { value: '', label: '不提醒' },
  { value: '0', label: '准时提醒' },
  { value: '15', label: '提前 15 分钟' },
  { value: '30', label: '提前 30 分钟' },
  { value: '60', label: '提前 1 小时' },
  { value: '120', label: '提前 2 小时' },
  { value: '1440', label: '提前 1 天' },
];

const weekOptions = [
  { value: 1, label: '一' },
  { value: 2, label: '二' },
  { value: 3, label: '三' },
  { value: 4, label: '四' },
  { value: 5, label: '五' },
  { value: 6, label: '六' },
  { value: 0, label: '日' },
];

function blankForm(): FormState {
  return {
    title: '',
    description: '',
    priority: 'normal',
    dueDate: '',
    dueTime: '',
    visibility: 'household',
    color: '',
    assigneeIds: [],
    subtasks: [],
    reminder: '',
    freq: '',
    interval: '1',
    weekdays: [],
    until: '',
    anchor: 'due',
    templateId: null,
  };
}

function resetFromTask(task: Task | null): void {
  const next = blankForm();
  if (task) {
    next.title = task.title;
    next.description = task.description ?? '';
    next.priority = task.priority;
    next.dueDate = task.dueDate ?? '';
    next.dueTime = task.dueTime ?? '';
    next.visibility = task.visibility;
    next.color = task.color ?? '';
    next.assigneeIds = task.assignees.map((a) => a.id);
    next.subtasks = task.subtasks.map((s) => ({ id: s.id, title: s.title, done: s.done }));
    next.templateId = task.templateId ?? null;
    const rule = parseRecurrence(task.recurrenceRule);
    if (rule) {
      next.freq = rule.freq;
      next.interval = String(rule.interval || 1);
      next.weekdays = rule.weekdays ?? [];
      next.until = rule.until ?? '';
      next.anchor = rule.anchor ?? 'due';
    }
    const offsets = task.reminderOffsets ?? [];
    next.reminder = offsets.length ? String(offsets[0]) : '';
  }
  Object.assign(form, next);
  moreOpen.value = Boolean(task && (task.recurrenceRule || task.visibility !== 'household' || (task.reminderOffsets?.length ?? 0) > 0));
}

function applyTemplate(template: TaskTemplate): void {
  form.templateId = template.id;
  if (!form.title.trim()) form.title = template.title || template.name;
  if (!form.description && template.description) form.description = template.description;
  if (template.priority) form.priority = template.priority;
  if (template.color) form.color = template.color;
  if (!form.dueTime && template.defaultDueTime) form.dueTime = template.defaultDueTime;
  if (!form.assigneeIds.length && template.defaultAssigneeIds.length) form.assigneeIds = [...template.defaultAssigneeIds];
  if (!form.subtasks.length && template.subtaskTemplate.length) {
    form.subtasks = template.subtaskTemplate.map((title) => ({ title, done: false }));
  }
  if (template.reminderOffset !== null && template.reminderOffset !== undefined) {
    form.reminder = String(template.reminderOffset);
  }
  const rule = parseRecurrence(template.recurrenceRule);
  if (rule && !form.freq) {
    form.freq = rule.freq;
    form.interval = String(rule.interval || 1);
    form.weekdays = rule.weekdays ?? [];
    form.until = rule.until ?? '';
    form.anchor = rule.anchor ?? 'due';
    moreOpen.value = true;
  }
}

function onTemplateChange(value: string): void {
  const id = Number(value);
  const template = props.templates.find((t) => t.id === id);
  if (template) applyTemplate(template);
  else form.templateId = null;
}

watch(() => props.modelValue, (open) => {
  if (!open) return;
  error.value = '';
  saving.value = false;
  resetFromTask(props.task);
  if (!props.task && props.initialTemplate) applyTemplate(props.initialTemplate);
  if (!props.task && props.defaultDate && !form.dueDate) form.dueDate = props.defaultDate;
});

watch(() => props.task, (task) => {
  if (props.modelValue) resetFromTask(task);
});

function toggleAssignee(id: number): void {
  form.assigneeIds = form.assigneeIds.includes(id)
    ? form.assigneeIds.filter((v) => v !== id)
    : [...form.assigneeIds, id];
}

function toggleWeekday(day: number): void {
  form.weekdays = form.weekdays.includes(day)
    ? form.weekdays.filter((d) => d !== day)
    : [...form.weekdays, day];
}

function addSubtask(): void {
  form.subtasks = [...form.subtasks, { title: '', done: false }];
}

function removeSubtask(index: number): void {
  form.subtasks = form.subtasks.filter((_, i) => i !== index);
}

function buildRule(): string | null {
  if (!form.freq) return null;
  return JSON.stringify({
    freq: form.freq,
    interval: Math.max(1, Number(form.interval) || 1),
    weekdays: form.freq === 'weekly' ? form.weekdays : [],
    until: form.until || '',
    anchor: form.anchor,
  });
}

function buildPayload(): TaskPayload {
  const subtasks: SubtaskPayload[] = form.subtasks
    .filter((s) => s.title.trim())
    .map((s) => ({ id: s.id, title: s.title.trim(), done: s.done }));
  const payload: TaskPayload = {
    title: form.title.trim(),
    description: form.description.trim() || null,
    priority: form.priority,
    dueDate: form.dueDate || null,
    dueTime: form.dueTime || null,
    visibility: form.visibility,
    color: form.color || null,
    assigneeIds: form.assigneeIds,
    subtasks,
    reminderOffsets: form.reminder === '' ? [] : [Number(form.reminder)],
    recurrenceRule: buildRule(),
  };
  if (form.templateId) payload.templateId = form.templateId;
  return payload;
}

async function submit(): Promise<void> {
  error.value = '';
  if (!form.title.trim()) {
    error.value = '请填写待办标题';
    return;
  }
  if (form.dueTime && !form.dueDate) {
    error.value = '设置截止时间前请先选择截止日期';
    return;
  }
  if (form.freq && !form.dueDate) {
    error.value = '重复待办需要设置截止日期';
    return;
  }
  saving.value = true;
  try {
    const payload = buildPayload();
    const saved = props.task
      ? await tasks.updateTask(props.task.id, payload)
      : await tasks.createTask(payload);
    emit('saved', saved);
    emit('update:modelValue', false);
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '保存失败，请稍后重试';
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <AppDialog
    :model-value="modelValue"
    :title="title"
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <form class="task-form" @submit.prevent="submit">
      <AppField label="标题" for-id="task-title" required>
        <AppInput
          id="task-title"
          v-model="form.title"
          placeholder="要做什么？"
          :maxlength="120"
          :invalid="Boolean(error) && !form.title.trim()"
        />
      </AppField>

      <AppField label="说明" for-id="task-desc">
        <AppTextarea id="task-desc" v-model="form.description" :rows="3" placeholder="补充细节（可选）" :maxlength="5000" />
      </AppField>

      <div class="task-form__grid">
        <AppField label="截止日期" for-id="task-date">
          <AppInput id="task-date" v-model="form.dueDate" type="date" />
        </AppField>
        <AppField label="时间" for-id="task-time">
          <AppInput id="task-time" v-model="form.dueTime" type="time" />
        </AppField>
      </div>

      <div class="task-form__grid">
        <AppField label="优先级" for-id="task-priority">
          <AppSelect id="task-priority" v-model="form.priority" :options="priorityOptions" />
        </AppField>
        <AppField label="可见范围" for-id="task-visibility">
          <AppSelect id="task-visibility" v-model="form.visibility" :options="visibilityOptions" />
        </AppField>
      </div>

      <AppField label="指派给" hint="可多选；不选则由自己负责">
        <div class="chips">
          <button
            v-for="member in activeMembers"
            :key="member.id"
            type="button"
            class="chip"
            :class="{ 'is-active': form.assigneeIds.includes(member.id) }"
            @click="toggleAssignee(member.id)"
          >
            <AppAvatar :name="member.displayName" :size="20" />
            {{ member.displayName }}
          </button>
          <span v-if="!activeMembers.length" class="muted">暂无家庭成员</span>
        </div>
      </AppField>

      <AppField label="子任务">
        <div class="subtasks">
          <div v-for="(subtask, index) in form.subtasks" :key="index" class="subtasks__row">
            <span class="subtasks__dot" :class="{ 'is-done': subtask.done }" />
            <AppInput v-model="subtask.title" placeholder="子任务内容" :maxlength="120" />
            <button type="button" class="icon-btn" aria-label="删除子任务" @click="removeSubtask(index)">
              <AppIcon name="close" :size="15" />
            </button>
          </div>
          <button type="button" class="link-btn" @click="addSubtask">
            <AppIcon name="plus" :size="15" />添加子任务
          </button>
        </div>
      </AppField>

      <button type="button" class="more-toggle" :aria-expanded="moreOpen" @click="moreOpen = !moreOpen">
        <AppIcon name="settings" :size="15" />
        更多设置
        <AppIcon :name="moreOpen ? 'chevron-up' : 'chevron-down'" :size="15" class="more-toggle__chevron" />
      </button>

      <div v-if="moreOpen" class="more">
        <AppField v-if="!isEdit && templates.length" label="从模板填充" for-id="task-template">
          <AppSelect
            id="task-template"
            :model-value="form.templateId ? String(form.templateId) : ''"
            :options="templateOptions"
            @update:model-value="onTemplateChange"
          />
        </AppField>

        <div class="task-form__grid">
          <AppField label="提醒" for-id="task-reminder">
            <AppSelect id="task-reminder" v-model="form.reminder" :options="reminderOptions" />
          </AppField>
          <AppField label="标识色">
            <div class="swatches">
              <button
                type="button"
                class="swatch swatch--none"
                :class="{ 'is-active': !form.color }"
                aria-label="无标识色"
                @click="form.color = ''"
              />
              <button
                v-for="color in taskColors"
                :key="color"
                type="button"
                class="swatch"
                :class="{ 'is-active': form.color === color }"
                :style="{ background: color }"
                :aria-label="`标识色 ${color}`"
                @click="form.color = color"
              />
            </div>
          </AppField>
        </div>

        <div class="recur">
          <p class="recur__title">重复</p>
          <div class="task-form__grid">
            <AppField label="频率" for-id="task-freq">
              <AppSelect id="task-freq" v-model="form.freq" :options="freqOptions" />
            </AppField>
            <AppField v-if="form.freq" label="间隔" for-id="task-interval">
              <AppInput id="task-interval" v-model="form.interval" type="number" min="1" max="365" />
            </AppField>
          </div>
          <AppField v-if="form.freq === 'weekly'" label="星期">
            <div class="chips">
              <button
                v-for="day in weekOptions"
                :key="day.value"
                type="button"
                class="chip chip--small"
                :class="{ 'is-active': form.weekdays.includes(day.value) }"
                @click="toggleWeekday(day.value)"
              >
                {{ day.label }}
              </button>
            </div>
          </AppField>
          <div v-if="form.freq" class="task-form__grid">
            <AppField label="计算方式" for-id="task-anchor">
              <AppSelect id="task-anchor" v-model="form.anchor" :options="anchorOptions" />
            </AppField>
            <AppField label="结束于" for-id="task-until">
              <AppInput id="task-until" v-model="form.until" type="date" />
            </AppField>
          </div>
          <p v-if="form.freq" class="recur__hint">
            {{ form.anchor === 'completion' ? '完成后从当天开始计算下一次。' : '完成当前待办后自动生成下一次，不补造已错过的历史。' }}
          </p>
        </div>

        <p v-if="isEdit && task?.seriesId" class="more__notice">
          这是重复待办的一次发生项，修改会同时影响后续自动生成的待办。
        </p>
      </div>

      <p v-if="error" class="form-error" role="alert">{{ error }}</p>
    </form>

    <template #footer>
      <AppButton variant="ghost" @click="emit('update:modelValue', false)">取消</AppButton>
      <AppButton :loading="saving" @click="submit">{{ isEdit ? '保存' : '创建待办' }}</AppButton>
    </template>
  </AppDialog>
</template>

<style scoped>
.task-form {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.task-form__grid {
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
  background: var(--at-bg-surface);
  transition: border-color var(--at-duration-fast) var(--at-ease), background var(--at-duration-fast) var(--at-ease);
}

.chip--small {
  min-width: 38px;
  justify-content: center;
  padding: 4px 8px;
}

.chip.is-active {
  border-color: var(--at-color-primary);
  background: var(--at-color-primary);
  color: var(--at-text-on-primary);
}

.chip.is-active :deep(.at-avatar) {
  color: var(--at-color-ink);
}

.subtasks {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
}

.subtasks__row {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
}

.subtasks__dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--at-color-hairline-strong);
}

.subtasks__dot.is-done {
  background: var(--at-color-success);
}

.icon-btn {
  flex: none;
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border-radius: var(--at-radius-md);
  color: var(--at-color-muted);
}

.icon-btn:hover {
  background: var(--at-bg-surface-strong);
  color: var(--at-color-danger);
}

.link-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  align-self: flex-start;
  padding: 4px 0;
  font-size: var(--at-text-caption);
  color: var(--at-color-brand);
}

.more-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: var(--at-space-3) 0 0;
  border-top: 1px solid var(--at-color-hairline);
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  text-align: left;
}

.more-toggle__chevron {
  margin-left: auto;
}

.more {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
  padding: var(--at-space-4);
  background: var(--at-bg-canvas-soft);
  border-radius: var(--at-radius-lg);
}

.swatches {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
  min-height: 44px;
  align-items: center;
}

.swatch {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid transparent;
  box-shadow: 0 0 0 1px var(--at-color-hairline-strong);
}

.swatch--none {
  background: var(--at-bg-surface);
  position: relative;
}

.swatch--none::after {
  content: '';
  position: absolute;
  inset: 9px 3px;
  transform: rotate(-45deg);
  border-top: 1px solid var(--at-color-muted-soft);
}

.swatch.is-active {
  border-color: var(--at-bg-surface);
  box-shadow: 0 0 0 2px var(--at-color-primary);
}

.recur {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  padding-top: var(--at-space-3);
  border-top: 1px solid var(--at-color-hairline);
}

.recur__title {
  font-size: var(--at-text-caption);
  font-weight: 600;
}

.recur__hint {
  font-size: 12px;
  color: var(--at-color-muted);
  line-height: 1.6;
}

.more__notice {
  font-size: 12px;
  color: var(--at-color-muted);
  line-height: 1.6;
}

.form-error {
  padding: 10px var(--at-space-3);
  border-radius: var(--at-radius-md);
  background: var(--at-color-danger-soft);
  color: var(--at-color-danger);
  font-size: var(--at-text-caption);
}

.muted {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

@media (max-width: 639px) {
  .task-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>
