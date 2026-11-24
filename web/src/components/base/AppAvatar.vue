<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(defineProps<{
  name: string;
  size?: number;
  src?: string | null;
}>(), {
  size: 36,
  src: null,
});

const palette = ['#eaf3f7', '#f7f4ef', '#eef7f3', '#f4f0f9', '#fbf2ec', '#f0efed'];
const color = computed(() => {
  let hash = 0;
  for (const ch of props.name) hash = (hash * 31 + ch.charCodeAt(0)) % 997;
  return palette[hash % palette.length];
});
const initial = computed(() => (props.name || '?').trim().charAt(0));
</script>

<template>
  <span
    class="at-avatar"
    :style="{ width: `${size}px`, height: `${size}px`, background: src ? undefined : color, fontSize: `${Math.max(12, size * 0.4)}px` }"
    :aria-label="name"
    role="img"
  >
    <img v-if="src" :src="src" :alt="name" class="at-avatar__img" />
    <template v-else>{{ initial }}</template>
  </span>
</template>

<style scoped>
.at-avatar {
  display: inline-grid;
  place-items: center;
  flex: none;
  border-radius: 50%;
  color: var(--at-color-body-strong);
  font-weight: 500;
  overflow: hidden;
  border: 1px solid var(--at-color-hairline);
}

.at-avatar__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>
