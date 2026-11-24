<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { asData, type ContactsData } from '@/cards/types';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';

const props = defineProps<{ data: unknown }>();
const d = computed(() => asData<ContactsData>(props.data));
const items = computed(() => d.value?.items ?? []);

function phoneOf(id: number): string {
  const contact = items.value.find((c) => c.id === id);
  const phone = contact?.items.find((i) => i.kind === 'phone');
  return phone?.value ?? '';
}
</script>

<template>
  <div v-if="!items.length">
    <AppEmptyState title="通讯录是空的" description="把常联系的人记下来" icon="family" />
  </div>
  <ul v-else class="contacts">
    <li v-for="contact in items" :key="contact.id" class="contacts__row">
      <RouterLink :to="{ name: 'family', query: { tab: 'contacts', open: String(contact.id) } }" class="contacts__link">
        <AppAvatar :name="contact.name" :size="34" />
        <span class="contacts__body">
          <span class="contacts__name">{{ contact.name }}</span>
          <span class="contacts__meta">{{ contact.organization || contact.category || '联系人' }}</span>
        </span>
        <span class="contacts__phone">{{ phoneOf(contact.id) }}</span>
      </RouterLink>
    </li>
  </ul>
</template>

<style scoped>
.contacts {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.contacts__row {
  border-bottom: 1px solid var(--at-color-hairline);
}

.contacts__row:last-child {
  border-bottom: 0;
}

.contacts__link {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  min-height: 48px;
  padding: var(--at-space-2) 0;
  color: inherit;
}

.contacts__body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.contacts__name {
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
}

.contacts__meta {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.contacts__phone {
  flex: none;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}
</style>
