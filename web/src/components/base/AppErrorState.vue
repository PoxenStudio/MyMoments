<script setup lang="ts">
import { computed } from 'vue';
import AppButton from './AppButton.vue';
import AppIcon from './AppIcon.vue';
import type { IconName } from './icons';

const props = withDefaults(defineProps<{
  kind?: 'network' | 'server' | 'permission' | 'session' | 'notfound';
  title?: string;
  description?: string;
  compact?: boolean;
  retryLabel?: string;
}>(), {
  kind: 'server',
  title: undefined,
  description: undefined,
  compact: false,
  retryLabel: '重试',
});

defineEmits<{ retry: [] }>();

const copy = computed<{ icon: IconName; title: string; description: string }>(() => {
  switch (props.kind) {
    case 'network':
      return { icon: 'wifi-off', title: '网络连接失败', description: '请检查网络连接后重试。' };
    case 'permission':
      return { icon: 'lock', title: '没有访问权限', description: '当前账号无权查看此内容。' };
    case 'session':
      return { icon: 'lock', title: '登录已过期', description: '请重新登录后继续。' };
    case 'notfound':
      return { icon: 'search', title: '内容不存在', description: '内容可能已被删除或移动。' };
    default:
      return { icon: 'alert', title: '服务暂时不可用', description: '请稍后再试。' };
  }
});
</script>

<template>
  <div class="at-error" :class="{ 'at-error--compact': compact }">
    <span class="at-error__icon"><AppIcon :name="copy.icon" :size="compact ? 20 : 26" /></span>
    <div class="at-error__text">
      <h3 class="at-error__title">{{ title ?? copy.title }}</h3>
      <p class="at-error__desc">{{ description ?? copy.description }}</p>
    </div>
    <div class="at-error__actions">
      <AppButton v-if="kind !== 'permission' && kind !== 'notfound'" size="sm" variant="outline" @click="$emit('retry')">
        {{ retryLabel }}
      </AppButton>
      <slot />
    </div>
  </div>
</template>

<style scoped>
.at-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: var(--at-space-2);
  padding: var(--at-space-8) var(--at-space-4);
}

.at-error--compact {
  flex-direction: row;
  text-align: left;
  padding: var(--at-space-4);
  gap: var(--at-space-3);
}

.at-error__icon {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  flex: none;
  border-radius: 50%;
  background: var(--at-color-danger-soft);
  color: var(--at-color-danger);
}

.at-error--compact .at-error__icon {
  width: 36px;
  height: 36px;
}

.at-error__text {
  flex: 1;
}

.at-error__title {
  font-size: var(--at-text-body-sm);
  font-weight: 500;
}

.at-error__desc {
  margin-top: 2px;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.at-error__actions {
  display: flex;
  gap: var(--at-space-2);
  margin-top: var(--at-space-2);
}

.at-error--compact .at-error__actions {
  margin-top: 0;
}
</style>
