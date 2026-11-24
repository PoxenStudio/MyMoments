<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { asData, type NotesData } from '@/cards/types';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppIcon from '@/components/base/AppIcon.vue';

const props = defineProps<{ data: unknown }>();
const d = computed(() => asData<NotesData>(props.data));
const items = computed(() => d.value?.items ?? []);
</script>

<template>
  <div v-if="!items.length">
    <AppEmptyState title="没有便签" description="记下一句想说的话或一件小事" icon="notes" />
  </div>
  <ul v-else class="notes">
    <li v-for="note in items" :key="note.id" class="notes__row">
      <RouterLink :to="{ name: 'notes', query: { open: String(note.id) } }" class="notes__link">
        <span class="notes__head">
          <span class="notes__title">{{ note.title || '无标题便签' }}</span>
          <AppIcon v-if="note.pinned" name="pin" :size="14" class="notes__pin" />
        </span>
        <span v-if="note.body" class="notes__body">{{ note.body }}</span>
        <span class="notes__author">{{ note.authorName }}</span>
      </RouterLink>
    </li>
  </ul>
</template>

<style scoped>
.notes {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.notes__row {
  border-bottom: 1px solid var(--at-color-hairline);
}

.notes__row:last-child {
  border-bottom: 0;
}

.notes__link {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: var(--at-space-3) 0;
  color: inherit;
}

.notes__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-2);
}

.notes__title {
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notes__pin {
  flex: none;
  color: var(--at-color-muted);
}

.notes__body {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
  line-height: 1.55;
}

.notes__author {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted-soft);
}
</style>
