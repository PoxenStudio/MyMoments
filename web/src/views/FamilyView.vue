<script setup lang="ts">
import AppEmptyState from '@/components/base/AppEmptyState.vue';
</script>

<template>
  <div class="at-page">
    <header class="at-page__header">
      <div>
        <p class="at-caption-upper">成员</p>
        <h1 class="at-page__title">家庭</h1>
        <p class="at-page__subtitle">正在按里程碑接入。</p>
      </div>
    </header>
    <AppEmptyState icon="family" title="家庭页即将上线" description="M3 将接入生日与通讯录，M1 已接入成员管理接口。" />
  </div>
</template>
