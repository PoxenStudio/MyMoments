<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue';
import AppAvatar from '@/components/base/AppAvatar.vue';
import { avatarUrl } from '@/utils/avatar';
import type { User } from '@/types';

const props = withDefaults(defineProps<{
  modelValue: string;
  mentionIds: number[];
  members: User[];
  id?: string;
  placeholder?: string;
  rows?: number;
  maxlength?: number;
  disabled?: boolean;
}>(), {
  id: undefined,
  placeholder: undefined,
  rows: 3,
  maxlength: undefined,
  disabled: false,
});

const emit = defineEmits<{
  'update:modelValue': [value: string];
  'update:mentionIds': [value: number[]];
}>();

const root = ref<HTMLElement | null>(null);
const area = ref<HTMLTextAreaElement | null>(null);
const triggerPos = ref<number | null>(null);
const query = ref<string | null>(null);
const activeIndex = ref(0);

const candidates = computed(() => {
  if (query.value === null) return [];
  const q = query.value.trim().toLowerCase();
  return props.members
    .filter((m) => m.status === 'active')
    .filter((m) => q === '' || m.displayName.toLowerCase().includes(q))
    .slice(0, 8);
});

function onInput(event: Event): void {
  const el = event.target as HTMLTextAreaElement;
  emit('update:modelValue', el.value);
  const kept = props.mentionIds.filter((id) => {
    const member = props.members.find((m) => m.id === id);
    return member ? el.value.includes(`@${member.displayName}`) : false;
  });
  if (kept.length !== props.mentionIds.length) emit('update:mentionIds', kept);
  refreshQuery(el);
}

function refreshQuery(el: HTMLTextAreaElement): void {
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

function choose(member: User): void {
  const el = area.value;
  if (!el || triggerPos.value === null) return;
  const caret = el.selectionStart ?? props.modelValue.length;
  const before = props.modelValue.slice(0, triggerPos.value);
  const after = props.modelValue.slice(caret);
  const inserted = `@${member.displayName} `;
  emit('update:modelValue', before + inserted + after);
  if (!props.mentionIds.includes(member.id)) {
    emit('update:mentionIds', [...props.mentionIds, member.id]);
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
    <textarea
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
    <div v-if="query !== null && candidates.length" class="mention__menu" role="listbox" aria-label="选择要提及的成员">
      <button
        v-for="(member, index) in candidates"
        :key="member.id"
        type="button"
        role="option"
        class="mention__option"
        :class="{ 'is-active': index === activeIndex }"
        :aria-selected="index === activeIndex"
        @mousedown.prevent="choose(member)"
      >
        <AppAvatar :name="member.displayName" :size="24" :src="avatarUrl(member.avatarKey)" />
        <span>{{ member.displayName }}</span>
        <span v-if="mentionIds.includes(member.id)" class="mention__picked">已选</span>
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

.mention__picked {
  margin-left: auto;
  font-size: 11px;
  color: var(--at-color-muted);
}

@media (max-width: 639px) {
  .mention__input {
    font-size: 16px;
  }
}
</style>
