<script setup lang="ts">
import AppEmptyState from '@/components/base/AppEmptyState.vue';
</script>

<template>
  <div class="at-page">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">规划</p>
        <h1 class="at-page__title">待办</h1>
        <p class="at-page__subtitle">正在按里程碑接入。</p>
      </div>
    </header>
    <AppEmptyState title="待办即将上线" description="M2 将接入任务、模板、重复与完成记录。" />
  </div>
</template>
