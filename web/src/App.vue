<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import AppShell from '@/components/AppShell.vue';
import AppToasts from '@/components/base/AppToasts.vue';

const route = useRoute();
const isAuthLayout = computed(() => route.meta.layout === 'auth');
</script>

<template>
  <AppShell v-if="!isAuthLayout">
    <RouterView />
  </AppShell>
  <RouterView v-else />
  <AppToasts />
</template>
