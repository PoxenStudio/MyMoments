<script setup lang="ts">
import AppEmptyState from '@/components/base/AppEmptyState.vue';
</script>

<template>
  <div class="at-page">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">交流</p>
        <h1 class="at-page__title">留言板</h1>
        <p class="at-page__subtitle">正在按里程碑接入。</p>
      </div>
    </header>
    <AppEmptyState icon="messages" title="留言板即将上线" description="M4 将接入留言、回复、@提及与个人消息。" />
  </div>
</template>
