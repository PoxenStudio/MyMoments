<script setup lang="ts">
import { ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api/client';
import AppDialog from './base/AppDialog.vue';
import AppIcon from './base/AppIcon.vue';
import AppInput from './base/AppInput.vue';
import AppLoading from './base/AppLoading.vue';
import AppEmptyState from './base/AppEmptyState.vue';
import type { SearchResult } from '@/types';

const props = defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>();

const router = useRouter();
const query = ref('');
const results = ref<SearchResult[]>([]);
const loading = ref(false);
const error = ref('');
let timer: number | undefined;
let seq = 0;

const typeLabels: Record<SearchResult['type'], string> = {
  task: '待办',
  event: '日程',
  contact: '联系人',
  message: '留言',
  note: '便签',
};

watch(() => props.modelValue, (open) => {
  if (open) {
    query.value = '';
    results.value = [];
    error.value = '';
  }
});

function runSearch(): void {
  const q = query.value.trim();
  window.clearTimeout(timer);
  if (q.length < 1) {
    results.value = [];
    loading.value = false;
    return;
  }
  timer = window.setTimeout(async () => {
    const current = ++seq;
    loading.value = true;
    error.value = '';
    try {
      const res = await api.get<{ items: SearchResult[] }>('/search', { q });
      if (current === seq) results.value = res.items;
    } catch {
      if (current === seq) error.value = '搜索失败，请稍后重试';
    } finally {
      if (current === seq) loading.value = false;
    }
  }, 260);
}

function open(result: SearchResult): void {
  emit('update:modelValue', false);
  switch (result.type) {
    case 'task':
      router.push({ name: 'tasks', query: { open: String(result.id) } });
      break;
    case 'event':
      router.push({ name: 'calendar', query: { open: String(result.id), date: result.date ?? undefined } });
      break;
    case 'contact':
      router.push({ name: 'family', query: { tab: 'contacts', openContact: String(result.id) } });
      break;
    case 'message':
      router.push({ name: 'messages', query: { open: String(result.id) } });
      break;
    case 'note':
      router.push({ name: 'notes', query: { open: String(result.id) } });
      break;
  }
}
</script>

<template>
  <AppDialog
    :model-value="modelValue"
    title="搜索"
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="search">
      <div class="search__input">
        <AppIcon name="search" :size="18" class="search__icon" />
        <AppInput
          v-model="query"
          placeholder="搜索待办、日程、联系人、留言、便签"
          autocomplete="off"
          @update:model-value="runSearch"
          @keydown.enter="results[0] && open(results[0])"
        />
      </div>
      <div v-if="loading" class="search__state"><AppLoading :rows="2" /></div>
      <p v-else-if="error" class="search__error">{{ error }}</p>
      <div v-else-if="results.length" class="search__results">
        <button v-for="item in results" :key="`${item.type}-${item.id}`" type="button" class="search__item" @click="open(item)">
          <span class="search__type">{{ typeLabels[item.type] }}</span>
          <span class="search__text">
            <span class="search__title">{{ item.title }}</span>
            <span v-if="item.summary" class="search__summary">{{ item.summary }}</span>
          </span>
          <span v-if="item.date" class="search__date">{{ item.date }}</span>
        </button>
      </div>
      <AppEmptyState
        v-else-if="query.trim()"
        icon="search"
        title="没有找到相关内容"
        description="换一个关键词试试，支持标题、正文和姓名。"
      />
      <p v-else class="search__hint">输入关键词开始搜索。结果只包含你有权访问的内容。</p>
    </div>
  </AppDialog>
</template>

<style scoped>
.search {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
}

.search__input {
  position: relative;
}

.search__icon {
  position: absolute;
  left: 14px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--at-color-muted);
  z-index: 1;
  pointer-events: none;
}

.search__input :deep(input) {
  padding-left: 42px;
}

.search__results {
  display: flex;
  flex-direction: column;
  max-height: 52vh;
  overflow-y: auto;
}

.search__item {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  padding: var(--at-space-3);
  border-radius: var(--at-radius-md);
  text-align: left;
  transition: background var(--at-duration-fast) var(--at-ease);
}

.search__item:hover {
  background: var(--at-bg-canvas-soft);
}

.search__type {
  flex: none;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.5px;
  padding: 2px 8px;
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface-strong);
  color: var(--at-color-muted);
}

.search__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.search__title {
  font-size: var(--at-text-body-sm);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search__summary {
  font-size: 13px;
  color: var(--at-color-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search__date {
  flex: none;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
  font-variant-numeric: tabular-nums;
}

.search__hint,
.search__error {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  padding: var(--at-space-2) 0;
}

.search__error {
  color: var(--at-color-danger);
}
</style>
