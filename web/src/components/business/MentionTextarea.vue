<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue';
import AppAvatar from '@/components/base/AppAvatar.vue';
import { avatarUrl } from '@/utils/avatar';
import type { Contact, MentionTarget, User } from '@/types';

interface Candidate {
  type: 'user' | 'contact';
  id: number;
  name: string;
  avatarKey?: string | null;
  subtitle: string;
}

const props = withDefaults(defineProps<{
  modelValue: string;
  mentions: MentionTarget[];
  members: User[];
  contacts?: Contact[];
  id?: string;
  placeholder?: string;
  rows?: number;
  maxlength?: number;
  disabled?: boolean;
  /** 单行输入（用于标题等场景） */
  single?: boolean;
}>(), {
  contacts: () => [],
  id: undefined,
  placeholder: undefined,
  rows: 3,
  maxlength: undefined,
  disabled: false,
  single: false,
});

const emit = defineEmits<{
  'update:modelValue': [value: string];
  'update:mentions': [value: MentionTarget[]];
}>();

const root = ref<HTMLElement | null>(null);
const area = ref<HTMLInputElement | HTMLTextAreaElement | null>(null);
const triggerPos = ref<number | null>(null);
const query = ref<string | null>(null);
const activeIndex = ref(0);

const candidates = computed<Candidate[]>(() => {
  if (query.value === null) return [];
  const q = query.value.trim().toLowerCase();
  const out: Candidate[] = [];
  for (const m of props.members) {
    if (m.status !== 'active') continue;
    if (q && !m.displayName.toLowerCase().includes(q)) continue;
    out.push({ type: 'user', id: m.id, name: m.displayName, avatarKey: m.avatarKey, subtitle: '家庭成员' });
  }
  for (const c of props.contacts) {
    if (q && !c.name.toLowerCase().includes(q)) continue;
    out.push({ type: 'contact', id: c.id, name: c.name, avatarKey: c.avatarKey, subtitle: '联系人' });
  }
  return out.slice(0, 8);
});

function isPicked(candidate: Candidate): boolean {
  return props.mentions.some((t) => t.type === candidate.type && t.id === candidate.id);
}

function resolveName(t: MentionTarget): string | undefined {
  if (t.name) return t.name;
  if (t.type === 'user') return props.members.find((m) => m.id === t.id)?.displayName;
  return props.contacts.find((c) => c.id === t.id)?.name;
}

function onInput(event: Event): void {
  const el = event.target as HTMLInputElement | HTMLTextAreaElement;
  emit('update:modelValue', el.value);
  const kept = props.mentions.filter((t) => {
    const name = resolveName(t);
    return name ? el.value.includes(`@${name}`) : false;
  });
  if (kept.length !== props.mentions.length) emit('update:mentions', kept);
  refreshQuery(el);
}

function refreshQuery(el: HTMLInputElement | HTMLTextAreaElement): void {
  const caret = el.selectionStart ?? el.value.length;
  const before = el.value.slice(0, caret);
  const match = /(?:^|[\s(（[【,，、:：])@([^\s@]{0,20})$/.exec(before);
  if (!match) {
    query.value = null;
    triggerPos.value = null;
    return;
  }
  query.value = match[1];
  triggerPos.value = caret - match[1].length - 1;
  activeIndex.value = 0;
}

function choose(candidate: Candidate): void {
  const el = area.value;
  if (!el || triggerPos.value === null) return;
  const caret = el.selectionStart ?? props.modelValue.length;
  const before = props.modelValue.slice(0, triggerPos.value);
  const after = props.modelValue.slice(caret);
  const inserted = `@${candidate.name} `;
  emit('update:modelValue', before + inserted + after);
  if (!isPicked(candidate)) {
    emit('update:mentions', [
      ...props.mentions,
      { type: candidate.type, id: candidate.id, name: candidate.name, avatarKey: candidate.avatarKey ?? null },
    ]);
  }
  query.value = null;
  triggerPos.value = null;
  void nextTick(() => {
    el.focus();
    const pos = before.length + inserted.length;
    el.setSelectionRange(pos, pos);
  });
}

function onKeydown(event: KeyboardEvent): void {
  if (query.value === null || candidates.value.length === 0) return;
  if (event.key === 'ArrowDown') {
    event.preventDefault();
    activeIndex.value = (activeIndex.value + 1) % candidates.value.length;
  } else if (event.key === 'ArrowUp') {
    event.preventDefault();
    activeIndex.value = (activeIndex.value - 1 + candidates.value.length) % candidates.value.length;
  } else if (event.key === 'Enter' || event.key === 'Tab') {
    event.preventDefault();
    choose(candidates.value[activeIndex.value]);
  } else if (event.key === 'Escape') {
    event.preventDefault();
    query.value = null;
    triggerPos.value = null;
  }
}

function onDocumentClick(event: MouseEvent): void {
  if (root.value && !root.value.contains(event.target as Node)) {
    query.value = null;
    triggerPos.value = null;
  }
}

onMounted(() => document.addEventListener('click', onDocumentClick));
onBeforeUnmount(() => document.removeEventListener('click', onDocumentClick));

defineExpose({
  focus: () => area.value?.focus(),
});
</script>

<template>
  <div ref="root" class="mention">
    <input
      v-if="single"
      :id="id"
      ref="area"
      class="mention__input mention__input--single"
      type="text"
      autocomplete="off"
      :value="modelValue"
      :placeholder="placeholder"
      :maxlength="maxlength"
      :disabled="disabled"
      @input="onInput"
      @keydown="onKeydown"
      @click="refreshQuery($event.target as HTMLInputElement)"
    />
    <textarea
      v-else
      :id="id"
      ref="area"
      class="mention__input"
      :value="modelValue"
      :rows="rows"
      :placeholder="placeholder"
      :maxlength="maxlength"
      :disabled="disabled"
      @input="onInput"
      @keydown="onKeydown"
      @click="refreshQuery($event.target as HTMLTextAreaElement)"
    />
    <div v-if="query !== null && candidates.length" class="mention__menu" role="listbox" aria-label="选择要提及的成员或联系人">
      <button
        v-for="(candidate, index) in candidates"
        :key="`${candidate.type}-${candidate.id}`"
        type="button"
        role="option"
        class="mention__option"
        :class="{ 'is-active': index === activeIndex }"
        :aria-selected="index === activeIndex"
        @mousedown.prevent="choose(candidate)"
      >
        <AppAvatar :name="candidate.name" :size="24" :src="avatarUrl(candidate.avatarKey)" />
        <span class="mention__name">{{ candidate.name }}</span>
        <span class="mention__subtitle">{{ candidate.subtitle }}</span>
        <span v-if="isPicked(candidate)" class="mention__picked">已选</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.mention {
  position: relative;
}

.mention__input {
  width: 100%;
  min-height: 78px;
  padding: var(--at-space-3) var(--at-space-4);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-md);
  color: var(--at-text-primary);
  font-size: var(--at-text-body-sm);
  line-height: 1.6;
  resize: vertical;
  transition: border-color var(--at-duration-fast) var(--at-ease), box-shadow var(--at-duration-fast) var(--at-ease);
}

.mention__input--single {
  height: 44px;
  min-height: 0;
  padding: 0 var(--at-space-4);
  resize: none;
}

.mention__input::placeholder {
  color: var(--at-color-muted-soft);
}

.mention__input:focus {
  outline: none;
  border-color: var(--at-color-ink);
  box-shadow: 0 0 0 1px var(--at-color-ink);
}

.mention__input:disabled {
  background: var(--at-bg-canvas-soft);
  color: var(--at-color-muted);
}

.mention__menu {
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 4px);
  z-index: var(--at-z-sticky);
  padding: var(--at-space-1);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-lg);
  box-shadow: var(--at-shadow-dialog);
  max-height: 240px;
  overflow-y: auto;
}

.mention__option {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  width: 100%;
  padding: var(--at-space-2);
  border-radius: var(--at-radius-md);
  font-size: var(--at-text-caption);
  text-align: left;
}

.mention__option.is-active,
.mention__option:hover {
  background: var(--at-bg-surface-strong);
}

.mention__name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mention__subtitle {
  font-size: 11px;
  color: var(--at-color-muted);
  padding: 1px 8px;
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-pill);
}

.mention__picked {
  font-size: 11px;
  color: var(--at-color-muted);
}

@media (max-width: 639px) {
  .mention__input {
    font-size: 16px;
  }
}
</style>
