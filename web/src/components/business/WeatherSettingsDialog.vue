<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { api, ApiError } from '@/api/client';
import { useAuthStore } from '@/stores/auth';
import { useToastStore } from '@/stores/toast';
import type { Household, WeatherSource } from '@/types';
import { nearestCity, provinces } from '@/data/cities';
import AppButton from '@/components/base/AppButton.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import AppSwitch from '@/components/base/AppSwitch.vue';
import AppIcon from '@/components/base/AppIcon.vue';

const props = defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [] }>();

const auth = useAuthStore();
const toast = useToastStore();

const mode = ref<'auto' | 'manual'>('manual');
const provinceName = ref(provinces[0]?.name ?? '');
const cityName = ref('');
const currentCity = ref('');
const loading = ref(false);
const saving = ref(false);
const locating = ref(false);
const publishHousehold = ref(false);
const error = ref('');
const sources = ref<WeatherSource[]>([]);
const provider = ref('');

const sourceOptions = computed(() =>
  sources.value.map((item) => ({
    value: item.id,
    label: item.available ? `${item.name}${item.default ? '（默认）' : ''}` : `${item.name}（未配置）`,
  })),
);

const activeSource = computed(() => sources.value.find((item) => item.id === provider.value) ?? null);

const provinceOptions = provinces.map((p) => ({ value: p.name, label: p.name }));
const cityOptions = computed(() => {
  const province = provinces.find((p) => p.name === provinceName.value);
  return (province?.cities ?? []).map((c) => ({ value: c.name, label: c.name }));
});

const selectedCity = computed(() => {
  const province = provinces.find((p) => p.name === provinceName.value);
  return province?.cities.find((c) => c.name === cityName.value) ?? null;
});

watch(() => props.modelValue, (open) => {
  if (open) void load();
});

watch(provinceName, () => {
  if (!cityOptions.value.some((c) => c.value === cityName.value)) {
    cityName.value = cityOptions.value[0]?.value ?? '';
  }
});

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [prefs, sourceRes] = await Promise.all([
      auth.loadPreferences(),
      api.get<{ items: WeatherSource[] }>('/weather/sources').catch(() => ({ items: [] as WeatherSource[] })),
    ]);
    sources.value = sourceRes.items;
    provider.value =
      auth.household?.defaultWeatherProvider ||
      sourceRes.items.find((item) => item.default)?.id ||
      sourceRes.items[0]?.id ||
      '';
    mode.value = prefs.weatherAuto ? 'auto' : 'manual';
    currentCity.value = prefs.weatherCity ?? '';
    if (prefs.weatherCity) {
      const province = provinces.find((p) => p.cities.some((c) => c.name === prefs.weatherCity));
      if (province) {
        provinceName.value = province.name;
        cityName.value = prefs.weatherCity;
      }
    } else if (!cityName.value) {
      cityName.value = cityOptions.value[0]?.value ?? '';
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '读取天气设置失败';
  } finally {
    loading.value = false;
  }
}

function locate(): Promise<void> {
  return new Promise((resolve) => {
    if (!navigator.geolocation) {
      error.value = '当前浏览器不支持定位，请手动选择城市';
      resolve();
      return;
    }
    locating.value = true;
    error.value = '';
    navigator.geolocation.getCurrentPosition(
      (position) => {
        const city = nearestCity(position.coords.latitude, position.coords.longitude);
        if (city) {
          const province = provinces.find((p) => p.cities.some((c) => c.name === city.name));
          if (province) {
            provinceName.value = province.name;
            cityName.value = city.name;
          }
          currentCity.value = city.name;
        }
        locating.value = false;
        resolve();
      },
      () => {
        error.value = '定位失败，请允许位置权限或手动选择城市';
        locating.value = false;
        resolve();
      },
      { timeout: 8000, maximumAge: 600000 },
    );
  });
}

async function save(): Promise<void> {
  if (mode.value === 'auto') {
    await locate();
    if (error.value) return;
  }
  const city = selectedCity.value;
  if (!city) {
    error.value = '请选择一个城市';
    return;
  }
  saving.value = true;
  error.value = '';
  try {
    if (auth.isAdmin && provider.value) {
      const chosen = sources.value.find((item) => item.id === provider.value);
      if (chosen && !chosen.available) {
        error.value = '该天气信息源尚未配置，请联系部署者';
        return;
      }
    }
    await auth.savePreferences({
      weatherAuto: mode.value === 'auto',
      weatherCity: city.name,
      weatherLat: city.lat,
      weatherLon: city.lon,
    });
    if (auth.isAdmin) {
      const patch: Partial<Household> = {};
      if (provider.value) patch.defaultWeatherProvider = provider.value;
      if (publishHousehold.value) {
        patch.defaultWeatherCity = city.name;
        patch.defaultWeatherLat = city.lat;
        patch.defaultWeatherLon = city.lon;
      }
      if (Object.keys(patch).length) await auth.updateHousehold(patch);
    }
    toast.success('天气位置已更新');
    emit('saved');
    emit('update:modelValue', false);
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '保存天气设置失败';
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <AppDialog :model-value="modelValue" title="天气设置" size="sm" @update:model-value="emit('update:modelValue', $event)">
    <div v-if="loading" class="weather-settings__loading">正在读取设置…</div>
    <div v-else class="weather-settings">
      <p class="weather-settings__current">
        <AppIcon name="map-pin" :size="15" />
        <span>{{ currentCity || '还没有设置位置' }}</span>
      </p>

      <div class="weather-settings__modes" role="tablist" aria-label="天气位置方式">
        <button
          type="button"
          role="tab"
          class="weather-settings__mode"
          :class="{ 'is-active': mode === 'auto' }"
          :aria-selected="mode === 'auto'"
          @click="mode = 'auto'"
        >
          使用当前位置
        </button>
        <button
          type="button"
          role="tab"
          class="weather-settings__mode"
          :class="{ 'is-active': mode === 'manual' }"
          :aria-selected="mode === 'manual'"
          @click="mode = 'manual'"
        >
          手动选择城市
        </button>
      </div>

      <div v-if="mode === 'auto'" class="weather-settings__auto">
        <p>天气卡片会请求浏览器定位，只用于获取所在城市。若无法定位，可以改用手动选择。</p>
        <AppButton variant="outline" size="sm" :loading="locating" @click="locate">
          <AppIcon name="map-pin" :size="15" />重新定位
        </AppButton>
      </div>

      <div v-else class="weather-settings__fields">
        <label class="weather-settings__field">
          <span>省份</span>
          <AppSelect v-model="provinceName" :options="provinceOptions" aria-label="选择省份" />
        </label>
        <label class="weather-settings__field">
          <span>城市</span>
          <AppSelect v-model="cityName" :options="cityOptions" aria-label="选择城市" />
        </label>
      </div>

      <div class="weather-settings__source">
        <div class="weather-settings__source-head">
          <span>天气信息源</span>
          <span class="weather-settings__source-hint">{{ auth.isAdmin ? '全家统一' : '由管理员设置' }}</span>
        </div>
        <AppSelect
          v-if="auth.isAdmin && sourceOptions.length"
          v-model="provider"
          :options="sourceOptions"
          aria-label="天气信息源"
        />
        <p v-else-if="!auth.isAdmin" class="weather-settings__source-readonly">{{ activeSource?.name || '默认信息源' }}</p>
        <p v-if="activeSource?.note" class="weather-settings__source-note">{{ activeSource.note }}</p>
      </div>

      <AppSwitch
        v-if="auth.isAdmin"
        v-model="publishHousehold"
        label="同时设为家庭默认"
        description="其他还没有设置位置的家人会使用这个城市"
      />

      <p v-if="error" class="weather-settings__error">{{ error }}</p>
    </div>

    <template #footer>
      <AppButton variant="ghost" @click="emit('update:modelValue', false)">取消</AppButton>
      <AppButton :loading="saving || locating" :disabled="loading" @click="save">保存</AppButton>
    </template>
  </AppDialog>
</template>

<style scoped>
.weather-settings {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.weather-settings__loading {
  padding: var(--at-space-6) 0;
  text-align: center;
  color: var(--at-color-muted);
  font-size: var(--at-text-caption);
}

.weather-settings__current {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: var(--at-space-3);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface-strong);
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
}

.weather-settings__modes {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
  padding: 3px;
  border-radius: var(--at-radius-pill);
  background: var(--at-bg-surface-strong);
}

.weather-settings__mode {
  min-height: 34px;
  border: 0;
  border-radius: var(--at-radius-pill);
  background: transparent;
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  transition: background var(--at-duration-fast) var(--at-ease), color var(--at-duration-fast) var(--at-ease);
}

.weather-settings__mode.is-active {
  background: var(--at-bg-surface);
  color: var(--at-color-ink);
  box-shadow: var(--at-shadow-xs);
}

.weather-settings__auto p {
  margin-bottom: var(--at-space-3);
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  line-height: 1.6;
}

.weather-settings__fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--at-space-3);
}

.weather-settings__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.weather-settings__error {
  font-size: var(--at-text-caption);
  color: var(--at-color-danger);
}

.weather-settings__source {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-top: var(--at-space-1);
}

.weather-settings__source-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.weather-settings__source-hint {
  color: var(--at-color-muted-soft);
}

.weather-settings__source-readonly {
  padding: var(--at-space-2) var(--at-space-3);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface-strong);
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
}

.weather-settings__source-note {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted-soft);
}

@media (max-width: 479px) {
  .weather-settings__fields {
    grid-template-columns: 1fr;
  }
}
</style>
