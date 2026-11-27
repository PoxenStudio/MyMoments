<script setup lang="ts">
import { computed } from 'vue';
import AppSlider from '@/components/base/AppSlider.vue';
import {
  DEFAULT_BACKGROUND_OPACITY,
  backgroundPresets,
  cardTonePresets,
  cornerPresets,
  radiusOptions,
  radiusPx,
  tonePresets,
} from '@/data/theme';
import type { CardTone, ThemeConfig, ThemePatch } from '@/types';

const props = withDefaults(defineProps<{
  model: ThemeConfig;
  disabled?: boolean;
  showPreview?: boolean;
}>(), {
  disabled: false,
  showPreview: true,
});

const emit = defineEmits<{
  change: [patch: ThemePatch];
}>();

const cornerOpacity = computed(() => Math.round(props.model.cardCorner.opacity * 100));
const selectedCorner = computed(() => cornerPresets.find((item) => item.key === props.model.cardCorner.image) ?? null);
const selectedBg = computed(() => props.model.pageBackground);

function pickTone(key: ThemeConfig['tone']): void {
  emit('change', { tone: key });
}

function pickCardTone(key: CardTone): void {
  emit('change', { cardTone: key });
}

function pickRadius(key: ThemeConfig['cardRadius']): void {
  emit('change', { cardRadius: key });
}

function setKind(kind: 'color' | 'image'): void {
  emit('change', { pageBackground: { kind } });
}

function setColor(event: Event): void {
  emit('change', { pageBackground: { kind: 'color', color: (event.target as HTMLInputElement).value } });
}

function setBlur(value: number): void {
  emit('change', { pageBackground: { blur: value } });
}

function setOpacity(value: number): void {
  emit('change', { pageBackground: { opacity: value / 100 } });
}

function pickBackground(key: string): void {
  emit('change', { pageBackground: { kind: 'image', image: key } });
}

function pickCorner(key: string | null): void {
  emit('change', { cardCorner: { image: key } });
}

function setCornerOpacity(value: number): void {
  emit('change', { cardCorner: { opacity: value / 100 } });
}
</script>

<template>
  <div class="style-editor" :class="{ 'is-disabled': disabled }">
    <section class="style-editor__section">
      <div class="style-editor__heading">
        <h3>页面色调</h3>
        <p>只调整页面氛围，品牌主色与文字保持墨色。</p>
      </div>
      <div class="tone-grid" role="group" aria-label="页面色调">
        <button
          v-for="tone in tonePresets"
          :key="tone.key"
          type="button"
          class="tone-swatch tone-swatch--page"
          :class="{ 'is-active': model.tone === tone.key }"
          :aria-pressed="model.tone === tone.key"
          :aria-label="tone.label"
          :title="`${tone.label} · ${tone.description}`"
          @click="pickTone(tone.key)"
        >
          <span class="tone-swatch__dot" :style="{ '--swatch': tone.swatch }" />
          <span class="tone-swatch__label">{{ tone.label }}</span>
        </button>
      </div>
    </section>

    <section class="style-editor__section">
      <div class="style-editor__heading">
        <h3>卡片色调</h3>
        <p>独立于页面色调，只改变卡片与面板的底色。</p>
      </div>
      <div class="tone-grid" role="group" aria-label="卡片色调">
        <button
          v-for="tone in cardTonePresets"
          :key="tone.key"
          type="button"
          class="tone-swatch"
          :class="{ 'is-active': (model.cardTone ?? 'default') === tone.key }"
          :aria-pressed="(model.cardTone ?? 'default') === tone.key"
          :aria-label="tone.label"
          :title="tone.label"
          @click="pickCardTone(tone.key)"
        >
          <span class="tone-swatch__dot" :style="{ '--swatch': tone.swatch }" />
          <span class="tone-swatch__label">{{ tone.label }}</span>
        </button>
      </div>
    </section>

    <section class="style-editor__section">
      <div class="style-editor__heading">
        <h3>卡片圆角</h3>
        <p>影响所有卡片与面板的圆角大小。</p>
      </div>
      <div class="radius-grid" role="group" aria-label="卡片圆角">
        <button
          v-for="option in radiusOptions"
          :key="option.key"
          type="button"
          class="radius-option"
          :class="{ 'is-active': model.cardRadius === option.key }"
          :aria-pressed="model.cardRadius === option.key"
          @click="pickRadius(option.key)"
        >
          <span class="radius-option__shape" :style="{ borderRadius: `${radiusPx[option.key]}px` }" />
          <span class="radius-option__label">{{ option.label }}</span>
        </button>
      </div>
    </section>

    <section class="style-editor__section">
      <div class="style-editor__heading">
        <h3>页面背景</h3>
        <p>选择纯色或预设背景图，并按需要虚化。</p>
      </div>
      <div class="segmented" role="group" aria-label="背景类型">
        <button
          type="button"
          class="segmented__item"
          :class="{ 'is-active': selectedBg.kind === 'color' }"
          :aria-pressed="selectedBg.kind === 'color'"
          @click="setKind('color')"
        >
          纯色
        </button>
        <button
          type="button"
          class="segmented__item"
          :class="{ 'is-active': selectedBg.kind === 'image' }"
          :aria-pressed="selectedBg.kind === 'image'"
          @click="setKind('image')"
        >
          背景图
        </button>
      </div>

      <div v-if="selectedBg.kind === 'color'" class="color-row">
        <input
          type="color"
          class="color-row__picker"
          :value="selectedBg.color ?? '#f5f5f5'"
          aria-label="背景颜色"
          @input="setColor"
        />
        <span class="color-row__value at-tabular">{{ selectedBg.color ?? '#f5f5f5' }}</span>
      </div>

      <template v-else>
        <div class="bg-grid" role="group" aria-label="背景图">
          <button
            v-for="bg in backgroundPresets"
            :key="bg.key"
            type="button"
            class="bg-option"
            :class="{ 'is-active': selectedBg.image === bg.key }"
            :aria-pressed="selectedBg.image === bg.key"
            :aria-label="bg.label"
            :title="bg.label"
            @click="pickBackground(bg.key)"
          >
            <span class="bg-option__thumb" :style="{ backgroundImage: `url('${bg.image}')` }" />
            <span class="bg-option__label">{{ bg.label }}</span>
          </button>
        </div>
        <AppSlider
          :model-value="selectedBg.blur ?? 0"
          :min="0"
          :max="20"
          :step="1"
          label="虚化程度"
          suffix=" px"
          :disabled="disabled"
          @commit="setBlur"
        />
        <AppSlider
          :model-value="Math.round((selectedBg.opacity ?? DEFAULT_BACKGROUND_OPACITY) * 100)"
          :min="20"
          :max="100"
          :step="5"
          label="图片浓度"
          suffix="%"
          :disabled="disabled"
          @commit="setOpacity"
        />
      </template>
    </section>

    <section class="style-editor__section">
      <div class="style-editor__heading">
        <h3>卡片角图</h3>
        <p>只出现在卡片右下角的装饰角图，默认不显示。</p>
      </div>
      <div class="corner-grid" role="group" aria-label="卡片角图">
        <button
          type="button"
          class="corner-option corner-option--none"
          :class="{ 'is-active': !model.cardCorner.image }"
          :aria-pressed="!model.cardCorner.image"
          aria-label="无角图"
          @click="pickCorner(null)"
        >
          <span class="corner-option__thumb">无</span>
        </button>
        <button
          v-for="item in cornerPresets"
          :key="item.key"
          type="button"
          class="corner-option"
          :class="{ 'is-active': model.cardCorner.image === item.key }"
          :aria-pressed="model.cardCorner.image === item.key"
          :aria-label="item.label"
          :title="`${item.group === 'pattern' ? '花纹' : '卡通'} · ${item.label}`"
          @click="pickCorner(item.key)"
        >
          <span class="corner-option__thumb" :style="{ backgroundImage: `url('${item.image}')` }" />
          <span class="corner-option__label">{{ item.label }}</span>
        </button>
      </div>
      <AppSlider
        :model-value="cornerOpacity"
        :min="5"
        :max="25"
        :step="1"
        label="角图浓度"
        suffix="%"
        :disabled="disabled || !selectedCorner"
        @commit="setCornerOpacity"
      />
    </section>

    <section v-if="showPreview" class="style-editor__preview" aria-label="样式预览">
      <div
        class="preview-canvas"
        :class="{ 'has-image': selectedBg.kind === 'image' && selectedBg.image }"
        :style="{
          background: selectedBg.kind === 'color' ? selectedBg.color : undefined,
          backgroundImage: selectedBg.kind === 'image' && selectedBg.image
            ? `url('${backgroundPresets.find((item) => item.key === selectedBg.image)?.image}')`
            : undefined,
          filter: selectedBg.kind === 'image' ? `blur(${selectedBg.blur ?? 0}px)` : undefined,
          opacity: selectedBg.kind === 'image' && selectedBg.image
            ? String(selectedBg.opacity ?? DEFAULT_BACKGROUND_OPACITY)
            : undefined,
        }"
      />
      <div class="preview-card" :style="{ borderRadius: `${radiusPx[model.cardRadius]}px` }">
        <span
          v-if="selectedCorner"
          class="preview-card__corner"
          :style="{ backgroundImage: `url('${selectedCorner.image}')`, opacity: model.cardCorner.opacity }"
        />
        <p class="preview-card__title">周末家庭晚餐</p>
        <p class="preview-card__text">周六 18:30 · 一起做顿热乎的饭</p>
      </div>
      <span class="preview-label at-caption-upper">实时预览</span>
    </section>
  </div>
</template>

<style scoped>
.style-editor {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-5);
}

.style-editor.is-disabled {
  opacity: 0.55;
  pointer-events: none;
}

.style-editor__section {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
}

.style-editor__heading h3 {
  margin: 0;
  font-size: var(--at-text-body-sm);
  font-weight: 600;
}

.style-editor__heading p {
  margin: 2px 0 0;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.tone-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(88px, 1fr));
  gap: var(--at-space-2);
}

.tone-swatch {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  min-height: var(--at-touch-min);
  padding: 0 var(--at-space-2);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface);
  color: var(--at-color-body);
  font-size: var(--at-text-micro);
  cursor: pointer;
}

.tone-swatch:hover {
  border-color: var(--at-color-hairline-strong);
}

.tone-swatch.is-active {
  border-color: var(--at-color-ink);
  box-shadow: 0 0 0 2px var(--at-color-ink) inset;
}

.tone-swatch__dot {
  width: 18px;
  height: 18px;
  flex: none;
  border-radius: 50%;
  border: 1px solid var(--at-color-hairline-strong);
  background: var(--swatch);
}

/* 深色模式下压低页面色调色块亮度，保留色相以维持区分度 */
html[data-scheme='dark'] .tone-swatch--page .tone-swatch__dot {
  background: color-mix(in srgb, var(--swatch) 72%, #0c0a09);
}

.tone-swatch__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.radius-grid {
  display: flex;
  gap: var(--at-space-2);
}

.radius-option {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  min-height: var(--at-touch-min);
  padding: var(--at-space-2);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface);
  color: var(--at-color-muted);
  font-size: var(--at-text-micro);
  cursor: pointer;
}

.radius-option.is-active {
  border-color: var(--at-color-ink);
  color: var(--at-color-ink);
}

.radius-option__shape {
  width: 26px;
  height: 26px;
  border: 2px solid var(--at-color-hairline-strong);
  background: var(--at-bg-surface-strong);
}

.radius-option.is-active .radius-option__shape {
  border-color: var(--at-color-ink);
}

.segmented {
  display: inline-flex;
  align-self: flex-start;
  padding: 3px;
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface-strong);
}

.segmented__item {
  min-height: 34px;
  padding: 0 var(--at-space-4);
  border: 0;
  border-radius: var(--at-radius-pill);
  background: transparent;
  color: var(--at-color-muted);
  font-size: var(--at-text-caption);
  cursor: pointer;
}

.segmented__item.is-active {
  background: var(--at-bg-surface);
  color: var(--at-color-ink);
  box-shadow: var(--at-shadow-xs);
}

.color-row {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
}

.color-row__picker {
  width: 44px;
  height: 44px;
  padding: 0;
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
  background: transparent;
  cursor: pointer;
}

.color-row__value {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.bg-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(84px, 1fr));
  gap: var(--at-space-2);
}

.bg-option,
.corner-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 6px;
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface);
  color: var(--at-color-muted);
  font-size: var(--at-text-micro);
  cursor: pointer;
}

.bg-option.is-active,
.corner-option.is-active {
  border-color: var(--at-color-ink);
  color: var(--at-color-ink);
}

.bg-option__thumb {
  width: 100%;
  aspect-ratio: 4 / 3;
  border-radius: var(--at-radius-sm);
  background-size: cover;
  background-position: center;
}

.bg-option__label,
.corner-option__label {
  line-height: 1.2;
}

.corner-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(68px, 1fr));
  gap: var(--at-space-2);
}

.corner-option__thumb {
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  border-radius: var(--at-radius-sm);
  background-color: var(--at-bg-surface-strong);
  background-size: 80%;
  background-repeat: no-repeat;
  background-position: center;
}

.corner-option--none .corner-option__thumb {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.style-editor__preview {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 150px;
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-lg);
  overflow: hidden;
}

.preview-canvas {
  position: absolute;
  inset: -24px;
  background: var(--at-bg-page);
}

.preview-canvas.has-image {
  transform: scale(1.1);
}

.preview-card {
  position: relative;
  width: min(240px, 78%);
  padding: var(--at-space-4);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline);
  box-shadow: var(--at-shadow-card);
  overflow: hidden;
}

.preview-card__corner {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 64px;
  height: 64px;
  background-size: contain;
  background-repeat: no-repeat;
  background-position: bottom right;
  pointer-events: none;
}

.preview-card__title {
  margin: 0;
  font-size: var(--at-text-body-sm);
  font-weight: 600;
}

.preview-card__text {
  margin: 4px 0 0;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.preview-label {
  position: absolute;
  top: var(--at-space-2);
  right: var(--at-space-3);
  color: var(--at-color-muted-soft);
}

@media (max-width: 639px) {
  .tone-grid {
    grid-template-columns: repeat(auto-fill, minmax(78px, 1fr));
  }
}
</style>
