<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { asData, type MessagesData } from '@/cards/types';
import { formatShortDate } from '@/utils/chrono';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';

const props = defineProps<{ data: unknown }>();
const d = computed(() => asData<MessagesData>(props.data));
const items = computed(() => d.value?.items ?? []);
</script>

<template>
  <div v-if="!items.length">
    <AppEmptyState title="还没有留言" description="给家人留一句话吧" icon="messages" />
  </div>
  <ul v-else class="posts">
    <li v-for="post in items" :key="post.id" class="posts__row">
      <RouterLink :to="{ name: 'messages', query: { open: String(post.id) } }" class="posts__link">
        <AppAvatar :name="post.authorName ?? '家人'" :size="32" />
        <span class="posts__body">
          <span class="posts__head">
            <span class="posts__author">{{ post.authorName }}</span>
            <span class="posts__date">{{ formatShortDate((post.occurrenceDate ?? new Date(post.createdAt * 1000).toISOString()).slice(0, 10)) }}</span>
          </span>
          <span class="posts__text">{{ post.body }}</span>
        </span>
      </RouterLink>
    </li>
  </ul>
</template>

<style scoped>
.posts {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.posts__row {
  border-bottom: 1px solid var(--at-color-hairline);
}

.posts__row:last-child {
  border-bottom: 0;
}

.posts__link {
  display: flex;
  align-items: flex-start;
  gap: var(--at-space-3);
  padding: var(--at-space-3) 0;
  color: inherit;
}

.posts__body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  flex: 1;
}

.posts__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--at-space-2);
}

.posts__author {
  font-size: var(--at-text-caption);
  color: var(--at-color-body-strong);
}

.posts__date {
  flex: none;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.posts__text {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body);
  line-height: 1.55;
}
</style>
