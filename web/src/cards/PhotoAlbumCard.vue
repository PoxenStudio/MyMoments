<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { ApiError, api } from '@/api/client';
import { useToastStore } from '@/stores/toast';
import { asData, type PhotoAlbumData, type PhotoAlbumItem } from '@/cards/types';
import AppButton from '@/components/base/AppButton.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';
import AppSelect from '@/components/base/AppSelect.vue';

const props = defineProps<{ data: unknown }>();
const emit = defineEmits<{ refresh: [] }>();

const toast = useToastStore();

const MAX_COUNT = 20;
const fitOptions = [
  { value: 'cover', label: '自动缩放' },
  { value: 'contain', label: '等比例缩放' },
];

const fit = ref<string>('cover');
const index = ref(0);
const uploading = ref(false);
const deleting = ref<number | null>(null);
const hovering = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);
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
      form.append('fit', fit.value);
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

watch(hasMultiple, () => {
  index.value = 0;
  startTimer();
});

onMounted(startTimer);
onBeforeUnmount(stopTimer);
</script>

<template>
  <div class="photo">
    <div v-if="!count" class="photo__empty">
      <AppEmptyState
        title="还没有照片"
        description="上传家庭照片，多张照片会以走马灯自动展示"
        icon="image"
      />
    </div>

    <div v-else class="photo__stage" @mouseenter="hovering = true" @mouseleave="hovering = false">
      <img
        :src="imageURL(current!.id)"
        :alt="current!.originalName ?? ''"
        class="photo__img"
        :class="`photo__img--${current!.fit}`"
      />
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

    <div class="photo__toolbar">
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
      <label class="photo__mode">
        <span class="photo__mode-label">显示方式</span>
        <AppSelect v-model="fit" :options="fitOptions" aria-label="显示方式" />
      </label>
    </div>
  </div>
</template>

<style scoped>
.photo {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  min-width: 0;
}

.photo__empty :deep(.at-empty) {
  padding: var(--at-space-4) var(--at-space-3);
}

.photo__stage {
  position: relative;
  height: 200px;
  overflow: hidden;
  border-radius: var(--at-radius-lg);
  background: var(--at-bg-canvas-soft);
}

.photo__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.photo__img--cover {
  object-fit: cover;
}

.photo__img--contain {
  object-fit: contain;
}

.photo__nav,
.photo__delete {
  position: absolute;
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
  top: var(--at-space-2);
  right: var(--at-space-2);
  transform: none;
}

.photo__count {
  position: absolute;
  left: var(--at-space-2);
  bottom: var(--at-space-2);
  padding: 2px var(--at-space-2);
  font-size: var(--at-text-micro);
  color: var(--at-text-on-dark);
  background: color-mix(in srgb, var(--at-color-ink) 58%, transparent);
  border-radius: var(--at-radius-pill);
}

.photo__dots {
  position: absolute;
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
  background: color-mix(in srgb, var(--at-bg-surface) 70%, transparent);
  cursor: pointer;
  transition: background var(--at-duration-fast) var(--at-ease), transform var(--at-duration-fast) var(--at-ease);
}

.photo__dot.is-active {
  background: var(--at-color-ink);
  transform: scale(1.2);
}

.photo__toolbar {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  flex-wrap: wrap;
}

.photo__mode {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  margin-left: auto;
}

.photo__mode-label {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  white-space: nowrap;
}

.photo__mode :deep(.at-select) {
  width: 116px;
}

.photo__mode :deep(.at-select__control) {
  height: 32px;
  font-size: 14px;
  padding-top: 0;
  padding-bottom: 0;
}
</style>
