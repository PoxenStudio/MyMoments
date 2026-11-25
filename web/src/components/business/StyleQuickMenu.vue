<script setup lang="ts">
import { ref } from 'vue';
import { RouterLink } from 'vue-router';
import AppButton from '@/components/base/AppButton.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import { radiusPx, radiusOptions, tonePresets, cardTonePresets } from '@/data/theme';
import { useThemeStore } from '@/stores/theme';
import { useToastStore } from '@/stores/toast';
import type { CardRadius, CardTone, ThemeTone } from '@/types';

const theme = useThemeStore();
const toast = useToastStore();
const busy = ref(false);

async function pickTone(tone: ThemeTone): Promise<void> {
  if (busy.value) return;
  busy.value = true;
  try {
    await theme.setPersonal({ tone });
  } catch {
    toast.error('样式保存失败');
  } finally {
    busy.value = false;
  }
}

async function pickCardTone(cardTone: CardTone): Promise<void> {
  if (busy.value) return;
  busy.value = true;
  try {
    await theme.setPersonal({ cardTone });
  } catch {
    toast.error('样式保存失败');
  } finally {
    busy.value = false;
  }
}

async function pickRadius(radius: CardRadius): Promise<void> {
  if (busy.value) return;
  busy.value = true;
  try {
    await theme.setPersonal({ cardRadius: radius });
  } catch {
    toast.error('样式保存失败');
  } finally {
    busy.value = false;
  }
}

async function restore(): Promise<void> {
  busy.value = true;
  try {
    await theme.resetPersonal();
    toast.push('已恢复默认样式');
  } catch {
    toast.error('恢复默认失败');
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <div class="quick-menu at-card" role="dialog" aria-label="样式快速设置">
    <header class="quick-menu__head">
      <div>
        <p class="quick-menu__title">快速样式</p>
        <p class="quick-menu__hint">即时生效，并保存到你的偏好</p>
      </div>
      <span v-if="theme.useDefault" class="quick-menu__badge">跟随默认</span>
    </header>

    <section class="quick-menu__section">
      <p class="quick-menu__label">色调</p>
      <div class="quick-menu__tones" role="group" aria-label="快速切换色调">
        <button
          v-for="tone in tonePresets"
          :key="tone.key"
          type="button"
          class="quick-tone"
          :class="{ 'is-active': theme.effective.tone === tone.key }"
          :aria-pressed="theme.effective.tone === tone.key"
          :aria-label="tone.label"
          :title="`${tone.label} · ${tone.description}`"
          :style="{ background: tone.swatch }"
          @click="pickTone(tone.key)"
        >
          <span v-if="theme.effective.tone === tone.key" class="quick-tone__check">
            <AppIcon name="check" :size="14" />
          </span>
        </button>
      </div>
    </section>

    <section class="quick-menu__section">
      <p class="quick-menu__label">卡片色调</p>
      <div class="quick-menu__tones quick-menu__tones--cards" role="group" aria-label="快速切换卡片色调">
        <button
          v-for="tone in cardTonePresets"
          :key="tone.key"
          type="button"
          class="quick-tone"
          :class="{ 'is-active': (theme.effective.cardTone ?? 'default') === tone.key }"
          :aria-pressed="(theme.effective.cardTone ?? 'default') === tone.key"
          :aria-label="tone.label"
          :title="tone.label"
          :style="{ background: tone.swatch }"
          @click="pickCardTone(tone.key)"
        >
          <span v-if="(theme.effective.cardTone ?? 'default') === tone.key" class="quick-tone__check">
            <AppIcon name="check" :size="14" />
          </span>
        </button>
      </div>
    </section>

    <section class="quick-menu__section">
      <p class="quick-menu__label">圆角</p>
      <div class="quick-menu__radius" role="group" aria-label="快速切换卡片圆角">
        <button
          v-for="option in radiusOptions"
          :key="option.key"
          type="button"
          class="quick-radius"
          :class="{ 'is-active': theme.effective.cardRadius === option.key }"
          :aria-pressed="theme.effective.cardRadius === option.key"
          @click="pickRadius(option.key)"
        >
          <span class="quick-radius__shape" :style="{ borderRadius: `${Math.min(radiusPx[option.key], 14)}px` }" />
          {{ option.label }}
        </button>
      </div>
    </section>

    <footer class="quick-menu__foot">
      <AppButton variant="ghost" size="sm" :disabled="busy" @click="restore">恢复默认</AppButton>
      <RouterLink to="/settings" class="quick-menu__more">更多样式设置</RouterLink>
    </footer>
  </div>
</template>

<style scoped>
.quick-menu {
  position: absolute;
  top: calc(100% + 10px);
  right: 0;
  width: 320px;
  max-width: calc(100vw - 24px);
  padding: var(--at-space-4);
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
  box-shadow: var(--at-shadow-dialog);
  z-index: var(--at-z-overlay);
}

.quick-menu__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--at-space-3);
}

.quick-menu__title {
  margin: 0;
  font-size: var(--at-text-body-sm);
  font-weight: 600;
}

.quick-menu__hint {
  margin: 2px 0 0;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.quick-menu__badge {
  flex: none;
  padding: 2px 8px;
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface-strong);
  color: var(--at-color-muted);
  font-size: var(--at-text-micro);
}

.quick-menu__section {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
}

.quick-menu__label {
  margin: 0;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.quick-menu__tones {
  display: grid;
  grid-template-columns: repeat(11, 1fr);
  gap: 4px;
}

.quick-menu__tones--cards {
  grid-template-columns: repeat(8, 1fr);
}

.quick-tone {
  position: relative;
  aspect-ratio: 1;
  border-radius: 50%;
  border: 1px solid var(--at-color-hairline-strong);
  cursor: pointer;
}

.quick-tone.is-active {
  box-shadow: 0 0 0 2px var(--at-bg-surface), 0 0 0 3px var(--at-color-ink);
}

.quick-tone__check {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  color: var(--at-color-ink);
}

.quick-menu__radius {
  display: flex;
  gap: var(--at-space-2);
}

.quick-radius {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-height: 36px;
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface);
  color: var(--at-color-muted);
  font-size: var(--at-text-micro);
  cursor: pointer;
}

.quick-radius.is-active {
  border-color: var(--at-color-ink);
  color: var(--at-color-ink);
}

.quick-radius__shape {
  width: 14px;
  height: 14px;
  border: 1.5px solid currentColor;
}

.quick-menu__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-2);
}

.quick-menu__more {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.quick-menu__more:hover {
  color: var(--at-color-ink);
}

@media (max-width: 639px) {
  .quick-menu {
    position: fixed;
    top: 64px;
    left: 12px;
    right: 12px;
    width: auto;
    max-width: none;
  }

  .quick-menu__tones {
    grid-template-columns: repeat(6, 1fr);
    gap: 6px;
  }
}
</style>
