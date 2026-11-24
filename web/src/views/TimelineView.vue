<script setup lang="ts">
import AppEmptyState from '@/components/base/AppEmptyState.vue';
</script>

<template>
  <div class="at-page">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">回顾</p>
        <h1 class="at-page__title">时光轴</h1>
        <p class="at-page__subtitle">正在按里程碑接入。</p>
      </div>
    </header>
    <AppEmptyState icon="timeline" title="时光轴即将上线" description="M5 将聚合待办、日程、留言与便签。" />
  </div>
</template>
