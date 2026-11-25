import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import {
  SYSTEM_THEME,
  backgroundByKey,
  cornerByKey,
  radiusPx,
} from '@/data/theme';
import type { ThemeConfig, ThemePatch, ThemeScheme } from '@/types';

type StoredTheme = Partial<ThemeConfig> & {
  pageBackground?: Partial<ThemeConfig['pageBackground']>;
  cardCorner?: Partial<ThemeConfig['cardCorner']>;
};

function cloneSystem(): ThemeConfig {
  return {
    scheme: SYSTEM_THEME.scheme,
    cardTone: SYSTEM_THEME.cardTone,
    tone: SYSTEM_THEME.tone,
    cardRadius: SYSTEM_THEME.cardRadius,
    pageBackground: { ...SYSTEM_THEME.pageBackground },
    cardCorner: { ...SYSTEM_THEME.cardCorner },
  };
}

function mergeTheme(base: ThemeConfig, patch: StoredTheme | null | undefined): ThemeConfig {
  if (!patch) return { ...base, pageBackground: { ...base.pageBackground }, cardCorner: { ...base.cardCorner } };
  return {
    useDefault: patch.useDefault,
    scheme: patch.scheme ?? base.scheme,
    cardTone: patch.cardTone ?? base.cardTone,
    tone: patch.tone ?? base.tone,
    cardRadius: patch.cardRadius ?? base.cardRadius,
    pageBackground: { ...base.pageBackground, ...(patch.pageBackground ?? {}) },
    cardCorner: { ...base.cardCorner, ...(patch.cardCorner ?? {}) },
  };
}

function parseStored(raw: string | null | undefined): StoredTheme | null {
  if (!raw) return null;
  try {
    const value = JSON.parse(raw) as unknown;
    if (!value || typeof value !== 'object' || Array.isArray(value)) return null;
    return value as StoredTheme;
  } catch {
    return null;
  }
}

export const useThemeStore = defineStore('theme', () => {
  const personal = ref<StoredTheme | null>(null);
  const household = ref<StoredTheme | null>(null);
  const loaded = ref(false);
  const saving = ref(false);

  const householdTheme = computed<ThemeConfig>(() => mergeTheme(cloneSystem(), household.value));

  const effective = computed<ThemeConfig>(() => {
    if (personal.value && personal.value.useDefault === false) {
      return mergeTheme(cloneSystem(), personal.value);
    }
    return householdTheme.value;
  });

  const useDefault = computed(() => personal.value?.useDefault !== false);
  const scheme = computed<ThemeScheme>(() => (personal.value?.scheme === 'dark' ? 'dark' : 'light'));
  const pageImage = computed(() => {
    const bg = effective.value.pageBackground;
    if (bg.kind !== 'image' || !bg.image) return null;
    return backgroundByKey[bg.image] ?? null;
  });
  const pageBlur = computed(() => effective.value.pageBackground.blur ?? 0);
  const cornerImage = computed(() => {
    const key = effective.value.cardCorner.image;
    return key ? cornerByKey[key]?.image ?? null : null;
  });

  function apply(): void {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;
    const theme = effective.value;
    root.dataset.theme = theme.tone;
    root.dataset.scheme = scheme.value;
    if (theme.cardTone && theme.cardTone !== 'default') {
      root.dataset.cardTone = theme.cardTone;
    } else {
      delete root.dataset.cardTone;
    }
    root.style.setProperty('--at-radius-card', `${radiusPx[theme.cardRadius]}px`);
    const bg = theme.pageBackground;
    if (bg.kind === 'color' && bg.color && scheme.value === 'light') {
      root.style.setProperty('--at-bg-page', bg.color);
    } else {
      root.style.removeProperty('--at-bg-page');
    }
    const corner = cornerImage.value;
    root.style.setProperty('--at-card-corner-image', corner ? `url("${corner}")` : 'none');
    root.style.setProperty('--at-card-corner-opacity', String(theme.cardCorner.opacity));
  }

  async function persistPersonal(next: StoredTheme | null): Promise<void> {
    const auth = useAuthStore();
    saving.value = true;
    try {
      await auth.savePreferences({ style: next ? JSON.stringify(next) : '' });
    } finally {
      saving.value = false;
    }
  }

  async function load(force = false): Promise<void> {
    if (loaded.value && !force) return;
    const auth = useAuthStore();
    try {
      const prefs = await auth.loadPreferences();
      personal.value = parseStored(prefs.style);
      household.value = parseStored(auth.household?.style);
    } catch {
      personal.value = null;
      household.value = null;
    } finally {
      loaded.value = true;
      apply();
    }
  }

  async function setPersonal(patch: ThemePatch): Promise<void> {
    const base = personal.value && personal.value.useDefault === false
      ? mergeTheme(cloneSystem(), personal.value)
      : effective.value;
    const next = mergeTheme(base, { ...patch, useDefault: false, scheme: patch.scheme ?? scheme.value } as StoredTheme);
    personal.value = next;
    apply();
    await persistPersonal(next);
  }

  async function setScheme(nextScheme: ThemeScheme): Promise<void> {
    const next: StoredTheme = { ...(personal.value ?? {}), scheme: nextScheme };
    personal.value = next;
    apply();
    await persistPersonal(next);
  }

  async function setUseDefault(flag: boolean): Promise<void> {
    if (flag) {
      const next: StoredTheme = { ...(personal.value ?? {}), useDefault: true };
      personal.value = next;
      apply();
      await persistPersonal(next);
      return;
    }
    const base = mergeTheme(householdTheme.value, personal.value ?? { useDefault: false });
    const next: StoredTheme = { ...base, useDefault: false };
    personal.value = next;
    apply();
    await persistPersonal(next);
  }

  async function resetPersonal(): Promise<void> {
    personal.value = null;
    apply();
    await persistPersonal(null);
  }

  async function setHousehold(patch: ThemePatch): Promise<void> {
    const auth = useAuthStore();
    const base = mergeTheme(cloneSystem(), household.value);
    const next = mergeTheme(base, patch as StoredTheme);
    household.value = next;
    apply();
    saving.value = true;
    try {
      await auth.updateHousehold({ style: JSON.stringify(next) });
    } finally {
      saving.value = false;
    }
  }

  async function resetHousehold(): Promise<void> {
    const auth = useAuthStore();
    household.value = null;
    apply();
    saving.value = true;
    try {
      await auth.updateHousehold({ style: '' });
    } finally {
      saving.value = false;
    }
  }

  function reset(): void {
    personal.value = null;
    household.value = null;
    loaded.value = false;
    apply();
  }

  return {
    personal,
    household,
    loaded,
    saving,
    effective,
    householdTheme,
    useDefault,
    scheme,
    pageImage,
    pageBlur,
    cornerImage,
    load,
    apply,
    setPersonal,
    setScheme,
    setUseDefault,
    resetPersonal,
    setHousehold,
    resetHousehold,
    reset,
  };
});
