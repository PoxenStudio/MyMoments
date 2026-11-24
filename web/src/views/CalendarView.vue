<script setup lang="ts">
import AppEmptyState from '@/components/base/AppEmptyState.vue';
</script>

<template>
  <div class="at-page">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">时间</p>
        <h1 class="at-page__title">行事历</h1>
        <p class="at-page__subtitle">正在按里程碑接入。</p>
      </div>
    </header>
    <AppEmptyState icon="calendar" title="行事历即将上线" description="M3 将接入四视图、生日图层与重复例外。" />
  </div>
</template>
