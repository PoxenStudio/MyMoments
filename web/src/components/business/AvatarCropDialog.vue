<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, reactive, ref, shallowRef, watch } from 'vue';
import AppButton from '@/components/base/AppButton.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppSlider from '@/components/base/AppSlider.vue';

type LoadedImage = ImageBitmap | HTMLImageElement;

const props = withDefaults(defineProps<{
  modelValue: boolean;
  file: File | null;
  title?: string;
}>(), {
  title: '调整头像',
});

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  confirm: [file: File];
}>();

// 输出边长：足够覆盖高清屏上的圆形头像，同时避免原图整张上传。
const OUTPUT_SIZE = 512;
const MAX_ZOOM = 3;

const stage = ref<HTMLDivElement | null>(null);
const canvasEl = ref<HTMLCanvasElement | null>(null);
const source = shallowRef<LoadedImage | null>(null);
const sourceWidth = ref(0);
const sourceHeight = ref(0);
const stageSize = ref(280);
const ready = ref(false);
const saving = ref(false);
const error = ref('');

const zoom = ref(1);
const offset = reactive({ x: 0, y: 0 });

let releaseSource: (() => void) | null = null;
let resizeObserver: ResizeObserver | null = null;
let dragging = false;
let dragStartX = 0;
let dragStartY = 0;
let dragOriginX = 0;
let dragOriginY = 0;

const zoomPercent = computed({
  get: () => Math.round(zoom.value * 100),
  set: (value: number) => setZoom(value / 100),
});

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

// 图片按“覆盖”方式铺满正方形取景区，zoom 在此基础上放大。
function metrics(): { x: number; y: number; width: number; height: number } | null {
  if (!source.value) return null;
  const size = stageSize.value;
  const base = size / Math.min(sourceWidth.value, sourceHeight.value);
  const scale = base * zoom.value;
  const width = sourceWidth.value * scale;
  const height = sourceHeight.value * scale;
  const maxX = Math.max(0, (width - size) / 2);
  const maxY = Math.max(0, (height - size) / 2);
  offset.x = clamp(offset.x, -maxX, maxX);
  offset.y = clamp(offset.y, -maxY, maxY);
  return {
    x: (size - width) / 2 + offset.x,
    y: (size - height) / 2 + offset.y,
    width,
    height,
  };
}

function draw(): void {
  const canvas = canvasEl.value;
  const image = source.value;
  const box = metrics();
  if (!canvas || !image || !box) return;
  const ctx = canvas.getContext('2d');
  if (!ctx) return;
  const size = stageSize.value;
  const dpr = Math.min(window.devicePixelRatio || 1, 2);
  canvas.width = Math.round(size * dpr);
  canvas.height = Math.round(size * dpr);
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, size, size);
  ctx.drawImage(image, box.x, box.y, box.width, box.height);

  // 圆形之外的区域压暗，模拟头像实际显示效果。
  ctx.beginPath();
  ctx.rect(0, 0, size, size);
  ctx.arc(size / 2, size / 2, size / 2 - 1, 0, Math.PI * 2, true);
  ctx.fillStyle = 'rgba(28, 24, 20, 0.55)';
  ctx.fill('evenodd');

  ctx.beginPath();
  ctx.arc(size / 2, size / 2, size / 2 - 1, 0, Math.PI * 2);
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.9)';
  ctx.lineWidth = 2;
  ctx.stroke();
}

function setZoom(next: number): void {
  const target = clamp(next, 1, MAX_ZOOM);
  if (target === zoom.value) return;
  // 围绕取景区中心缩放，保持当前构图不跳动。
  const ratio = target / zoom.value;
  offset.x *= ratio;
  offset.y *= ratio;
  zoom.value = target;
  draw();
}

function release(): void {
  resizeObserver?.disconnect();
  resizeObserver = null;
  releaseSource?.();
  releaseSource = null;
  source.value = null;
  sourceWidth.value = 0;
  sourceHeight.value = 0;
  ready.value = false;
}

async function loadSource(file: File): Promise<void> {
  release();
  error.value = '';
  zoom.value = 1;
  offset.x = 0;
  offset.y = 0;

  let image: LoadedImage | null = null;
  if (typeof createImageBitmap === 'function') {
    try {
      const bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' });
      image = bitmap;
      releaseSource = () => bitmap.close();
    } catch {
      image = null;
    }
  }
  if (!image) {
    const url = URL.createObjectURL(file);
    try {
      const element = new Image();
      element.src = url;
      await element.decode();
      image = element;
      releaseSource = () => URL.revokeObjectURL(url);
    } catch {
      URL.revokeObjectURL(url);
      error.value = '图片读取失败，请更换一张图片';
      return;
    }
  }

  source.value = image;
  sourceWidth.value = image instanceof HTMLImageElement ? image.naturalWidth : image.width;
  sourceHeight.value = image instanceof HTMLImageElement ? image.naturalHeight : image.height;
  ready.value = true;
  // 等对话框挂载完成后再测量取景区尺寸并绘制。
  await nextTick();
  observeStage();
  draw();
}

function observeStage(): void {
  resizeObserver?.disconnect();
  const element = stage.value;
  if (!element) return;
  stageSize.value = element.clientWidth || stageSize.value;
  resizeObserver = new ResizeObserver(() => {
    const size = stage.value?.clientWidth ?? 0;
    if (size > 0 && size !== stageSize.value) {
      stageSize.value = size;
      draw();
    }
  });
  resizeObserver.observe(element);
}

function onPointerDown(event: PointerEvent): void {
  if (!ready.value) return;
  event.preventDefault();
  dragging = true;
  dragStartX = event.clientX;
  dragStartY = event.clientY;
  dragOriginX = offset.x;
  dragOriginY = offset.y;
  (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
}

function onPointerMove(event: PointerEvent): void {
  if (!dragging) return;
  const ratio = stageSize.value / Math.max(1, (event.currentTarget as HTMLElement).clientWidth);
  offset.x = dragOriginX + (event.clientX - dragStartX) * ratio;
  offset.y = dragOriginY + (event.clientY - dragStartY) * ratio;
  draw();
}

function onPointerUp(event: PointerEvent): void {
  dragging = false;
  try {
    (event.currentTarget as HTMLElement).releasePointerCapture(event.pointerId);
  } catch {
    // 指针已释放时忽略
  }
}

function onWheel(event: WheelEvent): void {
  if (!ready.value) return;
  event.preventDefault();
  setZoom(zoom.value * (event.deltaY < 0 ? 1.1 : 0.9));
}

function onKeydown(event: KeyboardEvent): void {
  if (!ready.value) return;
  const step = event.shiftKey ? 16 : 4;
  const moves: Record<string, [number, number]> = {
    ArrowLeft: [-step, 0],
    ArrowRight: [step, 0],
    ArrowUp: [0, -step],
    ArrowDown: [0, step],
  };
  const move = moves[event.key];
  if (!move) return;
  event.preventDefault();
  offset.x += move[0];
  offset.y += move[1];
  draw();
}

function cropFile(): Promise<File | null> {
  const image = source.value;
  const box = metrics();
  if (!image || !box) return Promise.resolve(null);
  const ratio = OUTPUT_SIZE / stageSize.value;
  const out = document.createElement('canvas');
  out.width = OUTPUT_SIZE;
  out.height = OUTPUT_SIZE;
  const ctx = out.getContext('2d');
  if (!ctx) return Promise.resolve(null);

  const keepAlpha = props.file?.type === 'image/png';
  if (!keepAlpha) {
    ctx.fillStyle = '#fff';
    ctx.fillRect(0, 0, OUTPUT_SIZE, OUTPUT_SIZE);
  }
  ctx.drawImage(image, box.x * ratio, box.y * ratio, box.width * ratio, box.height * ratio);

  const type = keepAlpha ? 'image/png' : 'image/jpeg';
  return new Promise((resolve) => {
    out.toBlob((blob) => {
      if (!blob) {
        resolve(null);
        return;
      }
      const base = props.file?.name.replace(/\.[^.]+$/, '') || 'avatar';
      resolve(new File([blob], `${base}-crop.${keepAlpha ? 'png' : 'jpg'}`, { type }));
    }, type, 0.92);
  });
}

async function confirm(): Promise<void> {
  if (!ready.value) return;
  saving.value = true;
  error.value = '';
  try {
    const file = await cropFile();
    if (!file) {
      error.value = '生成头像失败，请重试';
      return;
    }
    emit('confirm', file);
    emit('update:modelValue', false);
  } finally {
    saving.value = false;
  }
}

watch(() => props.modelValue, (open) => {
  if (open && props.file) void loadSource(props.file);
  if (!open) release();
});

onBeforeUnmount(release);
</script>

<template>
  <AppDialog
    :model-value="modelValue"
    :title="title"
    size="sm"
    :close-on-backdrop="false"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="avatar-crop">
      <div ref="stage" class="avatar-crop__stage">
        <canvas
          ref="canvasEl"
          class="avatar-crop__canvas"
          tabindex="0"
          role="img"
          aria-label="头像裁剪预览"
          @pointerdown="onPointerDown"
          @pointermove="onPointerMove"
          @pointerup="onPointerUp"
          @pointercancel="onPointerUp"
          @wheel="onWheel"
          @keydown="onKeydown"
        />
      </div>
      <AppSlider
        v-model="zoomPercent"
        label="缩放"
        :min="100"
        :max="300"
        :step="5"
        suffix="%"
        :disabled="!ready"
      />
      <p class="avatar-crop__hint">拖动图片调整位置，滚轮或滑块缩放；圆形范围内就是头像最终效果。</p>
      <p v-if="error" class="avatar-crop__error" role="alert">{{ error }}</p>
    </div>

    <template #footer>
      <AppButton variant="ghost" @click="emit('update:modelValue', false)">取消</AppButton>
      <AppButton :loading="saving" :disabled="!ready" @click="confirm">使用此区域</AppButton>
    </template>
  </AppDialog>
</template>

<style scoped>
.avatar-crop {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.avatar-crop__stage {
  align-self: center;
  width: min(280px, 100%);
  aspect-ratio: 1;
  border-radius: var(--at-radius-lg);
  overflow: hidden;
  background: var(--at-bg-surface-strong);
  touch-action: none;
}

.avatar-crop__canvas {
  display: block;
  width: 100%;
  height: 100%;
  cursor: grab;
}

.avatar-crop__canvas:active {
  cursor: grabbing;
}

.avatar-crop__canvas:focus-visible {
  outline: 2px solid var(--at-color-primary);
  outline-offset: -2px;
}

.avatar-crop__hint {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
  line-height: 1.6;
  text-align: center;
}

.avatar-crop__error {
  font-size: var(--at-text-caption);
  color: var(--at-color-danger);
  text-align: center;
}
</style>
