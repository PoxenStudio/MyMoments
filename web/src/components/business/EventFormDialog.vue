<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import type { CalendarEvent } from '@/types';
import { useCalendarStore, type EventPayload } from '@/stores/calendar';
import { useFamilyStore } from '@/stores/family';
import { ApiError } from '@/api/client';
import { parseRecurrence, taskColors, visibilityLabels } from '@/utils/task';
import { dateTimeToLocal } from '@/utils/calendar';
import AppButton from '@/components/base/AppButton.vue';
import AppCheckbox from '@/components/base/AppCheckbox.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppField from '@/components/base/AppField.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import AppSwitch from '@/components/base/AppSwitch.vue';
import AppTextarea from '@/components/base/AppTextarea.vue';

const props = withDefaults(defineProps<{
  modelValue: boolean;
  event?: CalendarEvent | null;
  defaultDate?: string;
  defaultHour?: string;
}>(), {
  event: null,
  defaultDate: '',
  defaultHour: '',
});

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  saved: [event: CalendarEvent];
}>();

const calendar = useCalendarStore();
const family = useFamilyStore();
const saving = ref(false);

interface FormState {
  title: string;
  description: string;
  allDay: boolean;
  startDate: string;
  startTime: string;
  endDate: string;
  endTime: string;
  location: string;
  color: string;
  visibility: string;
  assigneeIds: number[];
  freq: string;
  interval: string;
  weekdays: number[];
  until: string;
  scope: 'all' | 'this' | 'following';
  reminderOffsets: number[];
}

const form = reactive<FormState>(blankForm());
const error = ref('');

const isEdit = computed(() => Boolean(props.event?.id));
const isSeries = computed(() => Boolean(props.event?.recurrenceRule));
const title = computed(() => (isEdit.value ? '编辑事件' : '新建事件'));
const activeMembers = computed(() => family.members.filter((m) => m.status === 'active'));

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

const scopeOptions = [
  { value: 'this', label: '仅本次' },
  { value: 'following', label: '本次及以后' },
  { value: 'all', label: '整个系列' },
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

const reminderOptions = [
  { value: 0, label: '准时' },
  { value: 15, label: '提前 15 分钟' },
  { value: 60, label: '提前 1 小时' },
  { value: 1440, label: '提前 1 天' },
  { value: 2880, label: '提前 2 天' },
  { value: 10080, label: '提前 1 周' },
];

function blankForm(): FormState {
  return {
    title: '',
    description: '',
    allDay: false,
    startDate: '',
    startTime: '09:00',
    endDate: '',
    endTime: '',
    location: '',
    color: '',
    visibility: 'household',
    assigneeIds: [],
    freq: '',
    interval: '1',
    weekdays: [],
    until: '',
    scope: 'all',
    reminderOffsets: [],
  };
}

function resetFromEvent(event: CalendarEvent | null): void {
  const next = blankForm();
  if (event) {
    next.title = event.title;
    next.description = event.description ?? '';
    next.allDay = event.allDay;
    next.startDate = event.startLocal.slice(0, 10);
    next.startTime = event.allDay ? '09:00' : event.startLocal.slice(11, 16);
    next.endDate = event.endLocal?.slice(0, 10) ?? '';
    next.endTime = event.endLocal && !event.allDay ? event.endLocal.slice(11, 16) : '';
    next.location = event.location ?? '';
    next.color = event.color ?? '';
    next.visibility = event.visibility;
    next.assigneeIds = event.assignees.map((a) => a.id);
    next.reminderOffsets = [...(event.reminderOffsets ?? [])];
    const rule = parseRecurrence(event.recurrenceRule);
    if (rule) {
      next.freq = rule.freq;
      next.interval = String(rule.interval || 1);
      next.weekdays = [...(rule.weekdays ?? [])];
      next.until = rule.until ?? '';
    }
    next.scope = 'this';
  } else {
    next.startDate = props.defaultDate || new Date().toISOString().slice(0, 10);
    next.startTime = props.defaultHour || '09:00';
    next.endDate = next.startDate;
    next.endTime = '';
  }
  Object.assign(form, next);
}

watch(() => props.modelValue, (open) => {
  if (open) {
    error.value = '';
    resetFromEvent(props.event ?? null);
    void family.load().catch(() => undefined);
  }
});

function toggleWeekday(day: number): void {
  form.weekdays = form.weekdays.includes(day)
    ? form.weekdays.filter((d) => d !== day)
    : [...form.weekdays, day];
}

function buildRule(): string | null {
  if (!form.freq) return null;
  const rule: Record<string, unknown> = {
    freq: form.freq,
    interval: Math.max(1, Number(form.interval) || 1),
    anchor: 'due',
  };
  if (form.freq === 'weekly' && form.weekdays.length) rule.weekdays = form.weekdays;
  if (form.until) rule.until = form.until;
  return JSON.stringify(rule);
}

function buildPayload(): EventPayload | null {
  error.value = '';
  if (!form.title.trim()) {
    error.value = '请填写事件标题';
    return null;
  }
  if (!form.startDate) {
    error.value = '请选择开始日期';
    return null;
  }
  const startLocal = form.allDay
    ? form.startDate
    : dateTimeToLocal(`${form.startDate}T${form.startTime || '09:00'}`);
  let endLocal: string | null = null;
  if (form.endDate) {
    endLocal = form.allDay
      ? form.endDate
      : dateTimeToLocal(`${form.endDate}T${form.endTime || form.startTime || '09:00'}`);
  } else if (!form.allDay && form.endTime) {
    endLocal = dateTimeToLocal(`${form.startDate}T${form.endTime}`);
  }
  if (endLocal && endLocal < startLocal) {
    error.value = '结束时间不能早于开始时间';
    return null;
  }
  const payload: EventPayload = {
    title: form.title.trim(),
    description: form.description.trim() || null,
    startLocal,
    endLocal,
    allDay: form.allDay,
    location: form.location.trim() || null,
    color: form.color || null,
    visibility: form.visibility,
    assigneeIds: form.assigneeIds,
    reminderOffsets: form.reminderOffsets,
    recurrenceRule: buildRule(),
  };
  if (isEdit.value && isSeries.value) {
    payload.scope = form.scope;
    if (form.scope !== 'all' && props.event?.occurrenceDate) {
      payload.occurrenceDate = props.event.occurrenceDate;
    }
  }
  return payload;
}

async function submit(): Promise<void> {
  const payload = buildPayload();
  if (!payload || saving.value) return;
  error.value = '';
  saving.value = true;
  try {
    const saved = props.event
      ? await calendar.updateEvent(props.event.id, payload)
      : await calendar.createEvent(payload);
    emit('saved', saved);
    emit('update:modelValue', false);
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '保存事件失败';
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <AppDialog :model-value="modelValue" :title="title" size="lg" @update:model-value="emit('update:modelValue', $event)">
    <form class="event-form at-stack" @submit.prevent="submit">
      <AppField label="标题" for-id="event-title" required>
        <AppInput id="event-title" v-model="form.title" placeholder="例如：外婆生日聚餐" :maxlength="120" />
      </AppField>

      <AppSwitch v-model="form.allDay" label="全天事件" description="全天事件只记录日期，不设置具体时间" />

      <div class="event-form__grid">
        <AppField label="开始日期" for-id="event-start-date" required>
          <AppInput id="event-start-date" v-model="form.startDate" type="date" />
        </AppField>
        <AppField v-if="!form.allDay" label="开始时间" for-id="event-start-time">
          <AppInput id="event-start-time" v-model="form.startTime" type="time" />
        </AppField>
      </div>

      <div class="event-form__grid">
        <AppField label="结束日期" for-id="event-end-date" hint="可留空">
          <AppInput id="event-end-date" v-model="form.endDate" type="date" />
        </AppField>
        <AppField v-if="!form.allDay" label="结束时间" for-id="event-end-time" hint="可留空">
          <AppInput id="event-end-time" v-model="form.endTime" type="time" />
        </AppField>
      </div>

      <div class="event-form__grid">
        <AppField label="地点" for-id="event-location">
          <AppInput id="event-location" v-model="form.location" placeholder="例如：外婆家" />
        </AppField>
        <AppField label="可见范围" for-id="event-visibility">
          <AppSelect id="event-visibility" v-model="form.visibility" :options="visibilityOptions" aria-label="可见范围" />
        </AppField>
      </div>

      <AppField v-if="form.visibility === 'selected'" label="参与成员">
        <div class="event-form__chips">
          <button
            v-for="member in activeMembers"
            :key="member.id"
            type="button"
            class="event-form__chip"
            :class="{ 'is-on': form.assigneeIds.includes(member.id) }"
            :aria-pressed="form.assigneeIds.includes(member.id)"
            @click="form.assigneeIds = form.assigneeIds.includes(member.id)
              ? form.assigneeIds.filter((id) => id !== member.id)
              : [...form.assigneeIds, member.id]"
          >
            {{ member.displayName }}
          </button>
        </div>
      </AppField>

      <div class="event-form__grid">
        <AppField label="重复" for-id="event-freq">
          <AppSelect id="event-freq" v-model="form.freq" :options="freqOptions" aria-label="重复频率" />
        </AppField>
        <AppField v-if="form.freq" label="间隔" for-id="event-interval">
          <AppInput id="event-interval" v-model="form.interval" type="number" min="1" max="30" />
        </AppField>
      </div>

      <AppField v-if="form.freq === 'weekly'" label="重复日">
        <div class="event-form__week">
          <button
            v-for="day in weekOptions"
            :key="day.value"
            type="button"
            class="event-form__weekday"
            :class="{ 'is-on': form.weekdays.includes(day.value) }"
            :aria-pressed="form.weekdays.includes(day.value)"
            @click="toggleWeekday(day.value)"
          >
            {{ day.label }}
          </button>
        </div>
      </AppField>

      <AppField v-if="form.freq" label="结束日期" for-id="event-until" hint="留空表示一直重复">
        <AppInput id="event-until" v-model="form.until" type="date" />
      </AppField>

      <AppField label="提醒">
        <div class="event-form__reminders">
          <AppCheckbox
            v-for="option in reminderOptions"
            :key="option.value"
            :model-value="form.reminderOffsets.includes(option.value)"
            :label="option.label"
            @update:model-value="form.reminderOffsets = $event
              ? [...form.reminderOffsets, option.value]
              : form.reminderOffsets.filter((v) => v !== option.value)"
          />
        </div>
      </AppField>

      <AppField label="颜色">
        <div class="event-form__colors">
          <button
            type="button"
            class="event-form__color"
            :class="{ 'is-on': !form.color }"
            aria-label="默认颜色"
            @click="form.color = ''"
          />
          <button
            v-for="color in taskColors"
            :key="color"
            type="button"
            class="event-form__color"
            :class="{ 'is-on': form.color === color }"
            :style="{ background: color }"
            :aria-label="`颜色 ${color}`"
            @click="form.color = color"
          />
        </div>
      </AppField>

      <AppField label="说明" for-id="event-description">
        <AppTextarea id="event-description" v-model="form.description" :rows="3" placeholder="补充一些细节…" :maxlength="5000" />
      </AppField>

      <AppField v-if="isEdit && isSeries" label="修改范围" for-id="event-scope" hint="重复事件可只调整这一次或整个系列">
        <AppSelect id="event-scope" v-model="form.scope" :options="scopeOptions" aria-label="修改范围" />
      </AppField>

      <p v-if="error" class="event-form__error" role="alert"><AppIcon name="alert" :size="16" />{{ error }}</p>
    </form>
    <template #footer>
      <AppButton variant="ghost" @click="emit('update:modelValue', false)">取消</AppButton>
      <AppButton @click="submit">保存</AppButton>
    </template>
  </AppDialog>
</template>

<style scoped>
.event-form__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--at-space-3);
}

.event-form__chips,
.event-form__week,
.event-form__colors,
.event-form__reminders {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

.event-form__chip {
  min-height: 36px;
  padding: 0 var(--at-space-3);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface);
  font-size: var(--at-text-caption);
  color: var(--at-text-secondary);
}

.event-form__chip.is-on {
  border-color: var(--at-color-primary);
  background: var(--at-color-primary);
  color: var(--at-text-on-primary);
}

.event-form__weekday {
  width: 38px;
  height: 38px;
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: 50%;
  background: var(--at-bg-surface);
  font-size: var(--at-text-caption);
  color: var(--at-text-secondary);
}

.event-form__weekday.is-on {
  border-color: var(--at-color-primary);
  background: var(--at-color-primary);
  color: var(--at-text-on-primary);
}

.event-form__color {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  border: 1px solid var(--at-color-hairline-strong);
  background: var(--at-bg-surface-strong);
}

.event-form__color.is-on {
  outline: 2px solid var(--at-color-ink);
  outline-offset: 2px;
}

.event-form__reminders :deep(.at-checkbox) {
  min-height: 40px;
  padding: 0 var(--at-space-3);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-pill);
}

.event-form__error {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--at-color-danger);
  font-size: 13px;
}

@media (max-width: 639px) {
  .event-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>
