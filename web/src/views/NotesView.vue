<script setup lang="ts">
import AppEmptyState from '@/components/base/AppEmptyState.vue';
</script>

<template>
  <div class="at-page">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">记录</p>
        <h1 class="at-page__title">便签</h1>
        <p class="at-page__subtitle">正在按里程碑接入。</p>
      </div>
    </header>
    <AppEmptyState icon="notes" title="便签即将上线" description="M4 将接入便签、检查清单与回忆日期。" />
  </div>
</template>
