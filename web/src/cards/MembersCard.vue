<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { asData, type MembersData } from '@/cards/types';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';

const props = defineProps<{ data: unknown }>();
const d = computed(() => asData<MembersData>(props.data));
const items = computed(() => d.value?.items ?? []);
</script>

<template>
  <div v-if="!items.length">
    <AppEmptyState title="还没有家庭成员" description="在家庭页邀请家人加入" icon="family" />
  </div>
  <ul v-else class="members">
    <li v-for="member in items" :key="member.id" class="members__row">
      <RouterLink :to="{ name: 'family' }" class="members__link">
        <AppAvatar :name="member.displayName" :size="34" />
        <span class="members__body">
          <span class="members__name">{{ member.displayName }}</span>
          <span class="members__relation">{{ member.relation || (member.role === 'admin' ? '管理员' : '家庭成员') }}</span>
        </span>
      </RouterLink>
    </li>
  </ul>
</template>

<style scoped>
.members {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.members__row {
  border-bottom: 1px solid var(--at-color-hairline);
}

.members__row:last-child {
  border-bottom: 0;
}

.members__link {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  min-height: 48px;
  padding: var(--at-space-2) 0;
  color: inherit;
}

.members__body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.members__name {
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
}

.members__relation {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}
</style>
