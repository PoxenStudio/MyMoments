<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch, type CSSProperties } from 'vue';
import { ApiError, api } from '@/api/client';
import { useToastStore } from '@/stores/toast';
import { asData, type PhotoAlbumData, type PhotoAlbumItem } from '@/cards/types';
import AppButton from '@/components/base/AppButton.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';

const props = defineProps<{ data: unknown }>();
const emit = defineEmits<{ refresh: [] }>();

const toast = useToastStore();

const MAX_COUNT = 20;
/** 左右各最多露出 3 张，连同中间共 7 张参与排布。 */
const CARD_SCALES = [1, 0.88, 0.76, 0.66];
const CARD_OPACITY = [1, 0.95, 0.8, 0.6];
const CARD_BRIGHTNESS = [1, 0.92, 0.82, 0.72];

const index = ref(0);
const uploading = ref(false);
const deleting = ref<number | null>(null);
const hovering = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);
const stageEl = ref<HTMLElement | null>(null);
const stageWidth = ref(0);
let observer: ResizeObserver | undefined;
let timer: number | undefined;

const items = computed<PhotoAlbumItem[]>(() => asData<PhotoAlbumData>(props.data)?.items ?? []);
const count = computed(() => items.value.length);
const hasMultiple = computed(() => count.value > 1);
const current = computed<PhotoAlbumItem | null>(() => {
  if (!items.value.length) return null;
  const i = Math.min(index.value, items.value.length - 1);
  return items.value[i] ?? items.value[0] ?? null;
});
const currentIndex = computed(() => {
  const id = current.value?.id;
  const i = items.value.findIndex((p) => p.id === id);
  return i < 0 ? 0 : i;
});

/** 以当前照片为中心的相对位置，负数在左、正数在右，超出半圈则折返。 */
function relIndex(i: number): number {
  const n = count.value;
  if (!n) return 0;
  let d = ((i - currentIndex.value) % n + n) % n;
  if (d > n / 2) d -= n;
  return d;
}

/** 卡片沿横向依次排开，越靠外越小、越暗，形成走马灯景深。 */
function cardStyle(i: number): CSSProperties {
  const d = relIndex(i);
  const distance = Math.min(Math.abs(d), 3);
  // 步长收窄后相邻照片彼此叠压，一屏能露出 7 张
  const step = Math.max(stageWidth.value * 0.17, 42);
  return {
    transform: `translate(-50%, -50%) translateX(${d * step}px) scale(${CARD_SCALES[distance]})`,
    zIndex: 10 - distance,
    opacity: Math.abs(d) > 3 ? 0 : CARD_OPACITY[distance],
    filter: `brightness(${CARD_BRIGHTNESS[distance]})`,
    pointerEvents: Math.abs(d) > 3 ? 'none' : 'auto',
  };
}

function imageURL(id: number): string {
  return `/api/v1/photo-album/${id}/image`;
}

function isAllowedImage(file: File): boolean {
  if (['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) return true;
  const ext = file.name.split('.').pop()?.toLowerCase();
  return ext === 'png' || ext === 'jpg' || ext === 'jpeg' || ext === 'webp';
}

function pickFile(): void {
  fileInput.value?.click();
}

async function onFileChange(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const files = Array.from(input.files ?? []);
  input.value = '';
  if (!files.length) return;

  if (!files.every(isAllowedImage)) {
    toast.error('仅支持 png、webp、jpg 格式的图片');
    return;
  }
  const remaining = MAX_COUNT - count.value;
  if (files.length > remaining) {
    toast.error(`照片墙最多 ${MAX_COUNT} 张，还能上传 ${remaining} 张`);
    return;
  }

  uploading.value = true;
  try {
    for (const file of files) {
      const form = new FormData();
      form.append('file', file);
      await api.upload<{ photo: PhotoAlbumItem }>('/photo-album', form);
    }
    toast.success(`已上传 ${files.length} 张照片`);
    emit('refresh');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '上传失败，请稍后重试');
  } finally {
    uploading.value = false;
  }
}

async function removeCurrent(): Promise<void> {
  const item = current.value;
  if (!item) return;
  deleting.value = item.id;
  try {
    await api.del(`/photo-album/${item.id}`);
    toast.success('照片已删除');
    if (index.value >= count.value - 1 && index.value > 0) index.value -= 1;
    emit('refresh');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除照片失败');
  } finally {
    deleting.value = null;
  }
}

function prev(): void {
  if (!count.value) return;
  index.value = (index.value - 1 + count.value) % count.value;
  restartTimer();
}

function next(): void {
  if (!count.value) return;
  index.value = (index.value + 1) % count.value;
  restartTimer();
}

function goTo(i: number): void {
  index.value = i;
  restartTimer();
}

function startTimer(): void {
  stopTimer();
  if (!hasMultiple.value) return;
  timer = window.setInterval(() => {
    if (document.hidden || hovering.value) return;
    next();
  }, 4000);
}

function stopTimer(): void {
  if (timer) window.clearInterval(timer);
  timer = undefined;
}

function restartTimer(): void {
  startTimer();
}

watch(stageEl, (el) => {
  observer?.disconnect();
  observer = undefined;
  if (!el) {
    stageWidth.value = 0;
    return;
  }
  stageWidth.value = el.clientWidth;
  if (typeof ResizeObserver === 'undefined') return;
  observer = new ResizeObserver((entries) => {
    const rect = entries[0]?.contentRect;
    if (rect) stageWidth.value = rect.width;
  });
  observer.observe(el);
});

watch(hasMultiple, () => {
  index.value = 0;
  startTimer();
});

onMounted(startTimer);
onBeforeUnmount(() => {
  observer?.disconnect();
  stopTimer();
});
</script>

<template>
  <div class="photo" @mouseenter="hovering = true" @mouseleave="hovering = false">
    <div v-if="!count" class="photo__empty">
      <AppEmptyState
        title="还没有照片"
        description="上传家庭照片，多张照片会以走马灯自动展示"
        icon="image"
      />
    </div>

    <div v-else ref="stageEl" class="photo__stage">
      <button
        v-for="(p, i) in items"
        :key="p.id"
        type="button"
        class="photo__card"
        :class="{ 'is-active': i === currentIndex }"
        :style="cardStyle(i)"
        :aria-label="`第 ${i + 1} 张照片`"
        @click="goTo(i)"
      >
        <img :src="imageURL(p.id)" :alt="p.originalName ?? ''" class="photo__img" draggable="false" />
      </button>
      <AppIconButton
        v-if="hasMultiple"
        icon="chevron-left"
        label="上一张"
        size="sm"
        class="photo__nav photo__nav--prev"
        @click="prev"
      />
      <AppIconButton
        v-if="hasMultiple"
        icon="chevron-right"
        label="下一张"
        size="sm"
        class="photo__nav photo__nav--next"
        @click="next"
      />
      <AppIconButton
        icon="trash"
        label="删除照片"
        size="sm"
        class="photo__delete"
        :active="deleting === current!.id"
        @click="removeCurrent"
      />
      <span class="photo__count">{{ currentIndex + 1 }} / {{ count }}</span>
      <div v-if="hasMultiple" class="photo__dots" role="group" aria-label="照片切换">
        <button
          v-for="(p, i) in items"
          :key="p.id"
          type="button"
          class="photo__dot"
          :class="{ 'is-active': i === currentIndex }"
          :aria-label="`第 ${i + 1} 张`"
          @click="goTo(i)"
        />
      </div>
    </div>

    <div class="photo__toolbar" :class="{ 'photo__toolbar--overlay': count }">
      <input
        ref="fileInput"
        type="file"
        accept="image/png,image/jpeg,image/webp"
        multiple
        hidden
        @change="onFileChange"
      />
      <AppButton
        size="sm"
        variant="outline"
        :loading="uploading"
        :disabled="count >= MAX_COUNT"
        @click="pickFile"
      >
        <AppIcon name="upload" :size="16" />{{ count >= MAX_COUNT ? '已达上限' : '上传照片' }}
      </AppButton>
    </div>
  </div>
</template>

<style scoped>
.photo {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  min-width: 0;
  height: 100%;
  min-height: 0;
}

.photo__empty :deep(.at-empty) {
  padding: var(--at-space-4) var(--at-space-3);
}

.photo__stage {
  position: relative;
  flex: 1;
  min-height: 168px;
  overflow: hidden;
  border-radius: var(--at-radius-lg);
  background:
    radial-gradient(120% 90% at 50% 0%, color-mix(in srgb, var(--at-bg-dark-elevated) 90%, #ffffff 10%), var(--at-bg-dark) 72%);
  isolation: isolate;
}

.photo__stage::before {
  content: '';
  position: absolute;
  inset: 0;
  z-index: 5;
  pointer-events: none;
  background: linear-gradient(90deg, rgba(0, 0, 0, 0.4), transparent 16%, transparent 84%, rgba(0, 0, 0, 0.4));
}

.photo__card {
  position: absolute;
  top: 50%;
  left: 50%;
  display: block;
  height: calc(100% - 30px);
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
  transform-origin: center;
  transition:
    transform 560ms cubic-bezier(0.22, 0.61, 0.36, 1),
    opacity 420ms var(--at-ease),
    filter 420ms var(--at-ease);
  will-change: transform, opacity;
}

.photo__card.is-active {
  cursor: default;
}

.photo__card:focus-visible {
  outline: 2px solid var(--at-text-on-dark);
  outline-offset: 3px;
  border-radius: var(--at-radius-md);
}

.photo__img {
  display: block;
  height: 100%;
  width: auto;
  max-width: none;
  border-radius: var(--at-radius-md);
  background: var(--at-bg-dark-elevated);
  box-shadow:
    0 18px 36px -16px rgba(0, 0, 0, 0.6),
    0 2px 6px rgba(0, 0, 0, 0.24);
  user-select: none;
  -webkit-user-drag: none;
}

.photo__card.is-active .photo__img {
  box-shadow:
    0 24px 48px -18px rgba(0, 0, 0, 0.68),
    0 0 0 1px color-mix(in srgb, var(--at-bg-surface) 22%, transparent);
}

.photo__nav,
.photo__delete {
  position: absolute;
  z-index: 20;
  top: 50%;
  transform: translateY(-50%);
  background: color-mix(in srgb, var(--at-bg-surface) 88%, transparent);
  backdrop-filter: blur(4px);
  box-shadow: var(--at-shadow-panel);
}

.photo__nav--prev {
  left: var(--at-space-2);
}

.photo__nav--next {
  right: var(--at-space-2);
}

.photo__delete {
  z-index: 20;
  top: var(--at-space-2);
  right: var(--at-space-2);
  transform: none;
  opacity: 0;
  pointer-events: none;
  transition: opacity var(--at-duration-base) var(--at-ease);
}

.photo:hover .photo__delete,
.photo:focus-within .photo__delete {
  opacity: 1;
  pointer-events: auto;
}

.photo__count {
  position: absolute;
  z-index: 20;
  top: var(--at-space-2);
  left: var(--at-space-2);
  padding: 2px var(--at-space-2);
  font-size: var(--at-text-micro);
  color: var(--at-text-on-dark);
  background: color-mix(in srgb, var(--at-bg-dark) 58%, transparent);
  border-radius: var(--at-radius-pill);
}

.photo__dots {
  position: absolute;
  z-index: 20;
  left: 50%;
  bottom: var(--at-space-2);
  transform: translateX(-50%);
  display: flex;
  gap: 6px;
}

.photo__dot {
  width: 7px;
  height: 7px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: color-mix(in srgb, var(--at-text-on-dark) 42%, transparent);
  cursor: pointer;
  transition: background var(--at-duration-fast) var(--at-ease), transform var(--at-duration-fast) var(--at-ease);
}

.photo__dot.is-active {
  background: var(--at-text-on-dark);
  transform: scale(1.2);
}

.photo__toolbar {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  flex-wrap: wrap;
}

.photo__toolbar--overlay {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 16;
  padding: 36px var(--at-space-3) 26px;
  background: linear-gradient(to top, rgba(12, 10, 9, 0.82), rgba(12, 10, 9, 0));
  opacity: 0;
  transform: translateY(6px);
  pointer-events: none;
  transition: opacity var(--at-duration-base) var(--at-ease), transform var(--at-duration-base) var(--at-ease);
}

.photo:hover .photo__toolbar--overlay,
.photo:focus-within .photo__toolbar--overlay {
  opacity: 1;
  transform: none;
  pointer-events: auto;
}

.photo__toolbar--overlay :deep(.at-btn--outline) {
  color: var(--at-text-on-dark);
  border-color: color-mix(in srgb, var(--at-text-on-dark) 38%, transparent);
  background: color-mix(in srgb, var(--at-bg-dark) 40%, transparent);
  backdrop-filter: blur(6px);
}

.photo__toolbar--overlay :deep(.at-btn--outline):hover:not(:disabled) {
  background: color-mix(in srgb, var(--at-text-on-dark) 16%, transparent);
  border-color: color-mix(in srgb, var(--at-text-on-dark) 55%, transparent);
}

@media (hover: none) {
  .photo__toolbar--overlay,
  .photo__delete {
    opacity: 1;
    transform: none;
    pointer-events: auto;
  }
}
</style>
