<script setup lang="ts">
import { useAuthStore } from '@/stores/auth';
import AppButton from '@/components/base/AppButton.vue';

const auth = useAuthStore();
</script>

<template>
  <div class="at-page">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">家庭面板</p>
        <h1 class="at-page__title">欢迎回来，{{ auth.user?.displayName }}</h1>
        <p class="at-page__subtitle">首期功能正在按里程碑接入。</p>
      </div>
      <div class="at-page__actions">
        <AppButton variant="outline" @click="$router.push('/tasks')">看待办</AppButton>
      </div>
    </header>
  </div>
</template>
