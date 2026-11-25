<script setup lang="ts">
import { computed } from 'vue';
import AppButton from '@/components/base/AppButton.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppIconButton from '@/components/base/AppIconButton.vue';
import AppSwitch from '@/components/base/AppSwitch.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import { cardRegistry } from '@/cards/registry';
import type { DashboardLayoutItem } from '@/types';

const props = defineProps<{
  modelValue: boolean;
  layout: DashboardLayoutItem[];
  layoutSource: string;
  saving: boolean;
  isAdmin: boolean;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  toggle: [id: string];
  move: [id: string, dir: -1 | 1];
  resize: [id: string, size: DashboardLayoutItem['size']];
  background: [id: string, background: string];
  reset: [];
  publish: [];
}>();

const sizeOptions = [
  { value: 'standard', label: '标准' },
  { value: 'wide', label: '宽' },
  { value: 'tall', label: '高' },
];

const backgrounds = [
  { value: 'default', label: '米白' },
  { value: 'warm', label: '暖阳' },
  { value: 'sage', label: '青绿' },
  { value: 'sky', label: '天空' },
  { value: 'sand', label: '沙色' },
  { value: 'gradient', label: '渐变' },
];

const sourceLabel = computed(() => {
  if (props.layoutSource === 'personal') return '当前使用个人布局';
  if (props.layoutSource === 'household') return '当前使用家庭默认布局';
  return '当前使用系统默认布局';
});

function def(id: string) {
  return cardRegistry[id];
}

function canMove(index: number, dir: -1 | 1): boolean {
  if (props.layout[index]?.id === 'overview') return false;
  const target = index + dir;
  if (target < 1 || target >= props.layout.length) return false;
  return props.layout[target]?.id !== 'overview';
}
</script>

<template>
  <AppDialog :model-value="modelValue" title="整理首页" size="lg" @update:model-value="emit('update:modelValue', $event)">
    <p class="customizer__hint">{{ sourceLabel }}。可以调整卡片顺序、大小与底色，也可以先隐藏暂时不看的内容。</p>

    <ul class="customizer__list">
      <li v-for="(item, index) in layout" :key="item.id" class="customizer__row">
        <div class="customizer__head">
          <span class="customizer__icon"><AppIcon :name="def(item.id)?.icon ?? 'grid'" :size="17" /></span>
          <AppSwitch
            :model-value="item.visible"
            :label="def(item.id)?.title ?? item.id"
            :description="def(item.id)?.description"
            :disabled="item.id === 'overview'"
            @update:model-value="emit('toggle', item.id)"
          />
        </div>

        <div class="customizer__controls">
          <div class="customizer__moves">
            <AppIconButton
              icon="chevron-up"
              label="上移"
              size="sm"
              :disabled="!canMove(index, -1)"
              @click="emit('move', item.id, -1)"
            />
            <AppIconButton
              icon="chevron-down"
              label="下移"
              size="sm"
              :disabled="!canMove(index, 1)"
              @click="emit('move', item.id, 1)"
            />
          </div>
          <AppSelect
            :model-value="item.size"
            :options="sizeOptions"
            :aria-label="`${def(item.id)?.title ?? item.id}尺寸`"
            @update:model-value="emit('resize', item.id, $event as DashboardLayoutItem['size'])"
          />
          <div class="customizer__swatches" role="group" :aria-label="`${def(item.id)?.title ?? item.id}底色`">
            <button
              v-for="bg in backgrounds"
              :key="bg.value"
              type="button"
              class="customizer__swatch"
              :class="[`customizer__swatch--${bg.value}`, { 'is-active': item.background === bg.value }]"
              :title="bg.label"
              :aria-label="bg.label"
              :aria-pressed="item.background === bg.value"
              @click="emit('background', item.id, bg.value)"
            />
          </div>
        </div>
      </li>
    </ul>

    <template #footer>
      <div class="customizer__footer">
        <div class="customizer__footer-left">
          <AppButton variant="ghost" size="sm" :loading="saving" @click="emit('reset')">恢复家庭默认</AppButton>
          <AppButton v-if="isAdmin" variant="outline" size="sm" :loading="saving" @click="emit('publish')">
            发布为家庭默认
          </AppButton>
        </div>
        <AppButton size="sm" @click="emit('update:modelValue', false)">完成</AppButton>
      </div>
    </template>
  </AppDialog>
</template>

<style scoped>
.customizer__hint {
  margin-bottom: var(--at-space-4);
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  line-height: 1.6;
}

.customizer__list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.customizer__row {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  padding: var(--at-space-3) 0;
  border-top: 1px solid var(--at-color-hairline);
}

.customizer__row:first-child {
  border-top: 0;
}

.customizer__head {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
}

.customizer__icon {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  flex: none;
  border-radius: 50%;
  background: var(--at-bg-surface-strong);
  color: var(--at-color-body);
}

.customizer__head :deep(.at-switch-row) {
  flex: 1;
}

.customizer__controls {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  padding-left: calc(32px + var(--at-space-3));
}

.customizer__moves {
  display: flex;
  gap: 2px;
}

.customizer__controls :deep(.at-select) {
  width: 88px;
}

.customizer__swatches {
  display: flex;
  gap: 6px;
  margin-left: auto;
}

.customizer__swatch {
  position: relative;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1px solid var(--at-color-hairline-strong);
  transition: transform var(--at-duration-fast) var(--at-ease), box-shadow var(--at-duration-fast) var(--at-ease);
}

.customizer__swatch::after {
  content: '';
  position: absolute;
  inset: -12px;
  border-radius: 50%;
}

.customizer__swatch:hover {
  transform: scale(1.08);
}

.customizer__swatch.is-active {
  box-shadow: 0 0 0 2px var(--at-bg-surface), 0 0 0 3px var(--at-color-ink);
}

.customizer__swatch--default { background: var(--at-bg-surface); }
.customizer__swatch--warm { background: var(--at-card-peach); }
.customizer__swatch--sage { background: var(--at-card-mint); }
.customizer__swatch--sky { background: var(--at-card-sky); }
.customizer__swatch--sand { background: var(--at-card-sand); }
.customizer__swatch--gradient {
  background: linear-gradient(135deg, var(--at-card-lavender) 0%, var(--at-card-peach) 52%, var(--at-card-mint) 100%);
}

.customizer__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-2);
  width: 100%;
}

.customizer__footer-left {
  display: flex;
  gap: var(--at-space-2);
}

@media (max-width: 639px) {
  .customizer__controls {
    flex-wrap: wrap;
    padding-left: 0;
  }

  .customizer__swatches {
    margin-left: 0;
  }

  .customizer__footer {
    flex-direction: column-reverse;
    align-items: stretch;
  }

  .customizer__footer-left {
    justify-content: space-between;
  }
}
</style>
