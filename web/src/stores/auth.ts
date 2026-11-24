import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { api } from '@/api/client';
import type { Household, Preferences, User } from '@/types';

interface BootstrapResponse {
  needsSetup: boolean;
  household?: Household;
  user?: User | null;
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null);
  const household = ref<Household | null>(null);
  const needsSetup = ref(false);
  const loaded = ref(false);
  const loading = ref(false);

  const isAuthenticated = computed(() => user.value !== null);
  const isAdmin = computed(() => user.value?.role === 'admin');
  const mustChangePassword = computed(() => user.value?.mustChangePassword === true);

  async function ensure(): Promise<void> {
    if (loaded.value) return;
    await refresh();
  }

  async function refresh(): Promise<void> {
    loading.value = true;
    try {
      const res = await api.get<BootstrapResponse>('/bootstrap');
      needsSetup.value = res.needsSetup;
      household.value = res.household ?? null;
      user.value = res.user ?? null;
    } catch (err) {
      needsSetup.value = false;
      user.value = null;
      throw err;
    } finally {
      loaded.value = true;
      loading.value = false;
    }
  }

  async function login(username: string, password: string): Promise<User> {
    const res = await api.post<{ user: User }>('/auth/login', { username, password });
    user.value = res.user;
    needsSetup.value = false;
    loaded.value = true;
    return res.user;
  }

  async function setup(payload: { householdName: string; username: string; displayName: string; password: string }): Promise<User> {
    const res = await api.post<{ user: User; household?: Household }>('/setup', payload);
    user.value = res.user;
    if (res.household) household.value = res.household;
    needsSetup.value = false;
    loaded.value = true;
    return res.user;
  }

  async function logout(): Promise<void> {
    try {
      await api.post('/auth/logout');
    } finally {
      user.value = null;
    }
  }

  async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
    await api.post('/auth/password', { currentPassword, newPassword });
    if (user.value) user.value = { ...user.value, mustChangePassword: false };
  }

  async function updateProfile(payload: { displayName?: string; relation?: string | null; avatarKey?: string | null }): Promise<User> {
    const res = await api.patch<{ user: User }>('/me', payload);
    user.value = res.user;
    return res.user;
  }

  async function loadPreferences(): Promise<Preferences> {
    return api.get<Preferences>('/me/preferences');
  }

  async function savePreferences(payload: Partial<Preferences>): Promise<Preferences> {
    return api.patch<Preferences>('/me/preferences', payload);
  }

  async function updateHousehold(payload: Partial<Household>): Promise<Household> {
    const res = await api.patch<{ household: Household }>('/family/household', payload);
    household.value = res.household;
    return res.household;
  }

  function clear(): void {
    user.value = null;
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('mymoments:unauthorized', () => clear());
  }

  return {
    user,
    household,
    needsSetup,
    loaded,
    loading,
    isAuthenticated,
    isAdmin,
    mustChangePassword,
    ensure,
    refresh,
    login,
    setup,
    logout,
    changePassword,
    updateProfile,
    loadPreferences,
    savePreferences,
    updateHousehold,
    clear,
  };
});
