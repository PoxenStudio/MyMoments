<script setup lang="ts">
import { useRouter } from 'vue-router';
import AppButton from '@/components/base/AppButton.vue';

const router = useRouter();
</script>

<template>
  <div class="at-page notfound">
    <p class="at-caption-upper">404</p>
    <h1 class="at-display notfound__title">这一页不在家里。</h1>
    <p class="at-muted">地址可能已失效，或内容已被移动。</p>
    <AppButton @click="router.push('/')">回到首页</AppButton>
  </div>
</template>

<style scoped>
.notfound {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--at-space-3);
  padding-top: 96px;
}

.notfound__title {
  font-size: var(--at-text-display-lg);
}
</style>
