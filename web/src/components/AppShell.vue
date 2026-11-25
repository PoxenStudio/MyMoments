<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { RouterLink, useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { useFamilyStore, useNotificationStore } from '@/stores/family';
import { useThemeStore } from '@/stores/theme';
import { useToastStore } from '@/stores/toast';
import AppAvatar from './base/AppAvatar.vue';
import AppDialog from './base/AppDialog.vue';
import AppIcon from './base/AppIcon.vue';
import AppIconButton from './base/AppIconButton.vue';
import StyleQuickMenu from './business/StyleQuickMenu.vue';
import SearchDialog from './SearchDialog.vue';
import NotificationPanel from './NotificationPanel.vue';
import type { IconName } from './base/icons';
import { todayISO } from '@/utils/chrono';
import { avatarUrl } from '@/utils/avatar';

interface NavItem {
  name: string;
  label: string;
  icon: IconName;
}

const navItems: NavItem[] = [
  { name: 'home', label: '首页', icon: 'home' },
  { name: 'calendar', label: '行事历', icon: 'calendar' },
  { name: 'tasks', label: '待办', icon: 'tasks' },
  { name: 'timeline', label: '时光轴', icon: 'timeline' },
  { name: 'family', label: '家庭', icon: 'family' },
  { name: 'messages', label: '留言板', icon: 'messages' },
  { name: 'notes', label: '便签', icon: 'notes' },
];

const utilityItems: NavItem[] = [
  { name: 'settings', label: '设置', icon: 'settings' },
];

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const family = useFamilyStore();
const notifications = useNotificationStore();
const toast = useToastStore();
const theme = useThemeStore();

const searchOpen = ref(false);
const notifOpen = ref(false);
const createOpen = ref(false);
const menuOpen = ref(false);
const drawerOpen = ref(false);
const styleOpen = ref(false);
const online = ref(typeof navigator === 'undefined' ? true : navigator.onLine);
const avatarWrap = ref<HTMLElement | null>(null);
const styleWrap = ref<HTMLElement | null>(null);
const menuButton = ref<HTMLButtonElement | null>(null);

const STORAGE_KEY = 'mymoments:sidebar-collapsed';
const manualCollapsed = ref<boolean | null>(readPreference());
const isMobile = ref(false);
const isWide = ref(true);

const currentName = computed(() => route.name?.toString() ?? '');
const title = computed(() => (route.meta.title ? String(route.meta.title) : '家时光'));
const showCreateFab = computed(() => currentName.value !== 'settings');
const initials = computed(() => auth.user?.displayName ?? '家');
const roleLabel = computed(() => (auth.user?.role === 'admin' ? '管理员' : '家庭成员'));

const collapsed = computed(() => {
  if (isMobile.value) return false;
  if (manualCollapsed.value !== null) return manualCollapsed.value;
  return !isWide.value;
});

const createActions: { label: string; icon: IconName; to: { name: string; query?: Record<string, string> } }[] = [
  { label: '新建待办', icon: 'tasks', to: { name: 'tasks', query: { new: '1' } } },
  { label: '新建日程', icon: 'calendar', to: { name: 'calendar', query: { new: '1', date: todayISO() } } },
  { label: '写一条留言', icon: 'messages', to: { name: 'messages', query: { new: '1' } } },
  { label: '新建便签', icon: 'notes', to: { name: 'notes', query: { new: '1' } } },
];

let pollTimer: number | undefined;
let mobileQuery: MediaQueryList | undefined;
let wideQuery: MediaQueryList | undefined;

function readPreference(): boolean | null {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    if (value === '1') return true;
    if (value === '0') return false;
  } catch {
    /* ignore */
  }
  return null;
}

function writePreference(value: boolean): void {
  try {
    localStorage.setItem(STORAGE_KEY, value ? '1' : '0');
  } catch {
    /* ignore */
  }
}

async function bootstrapData(): Promise<void> {
  if (!auth.user) return;
  await Promise.allSettled([family.load(), notifications.load(), theme.load()]);
}

function syncViewport(): void {
  isMobile.value = mobileQuery?.matches ?? false;
  isWide.value = wideQuery?.matches ?? true;
  if (!isMobile.value) closeDrawer(false);
}

function toggleSidebar(): void {
  if (isMobile.value) {
    drawerOpen.value = !drawerOpen.value;
    return;
  }
  manualCollapsed.value = !collapsed.value;
  writePreference(manualCollapsed.value);
}

function closeDrawer(restoreFocus = true): void {
  if (!drawerOpen.value) return;
  drawerOpen.value = false;
  if (restoreFocus) menuButton.value?.focus();
}

function onOnline(): void {
  online.value = navigator.onLine;
}
function onOffline(): void {
  online.value = navigator.onLine;
}
function onDocumentClick(event: MouseEvent): void {
  if (avatarWrap.value && !avatarWrap.value.contains(event.target as Node)) menuOpen.value = false;
  if (styleWrap.value && !styleWrap.value.contains(event.target as Node)) styleOpen.value = false;
}
function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    menuOpen.value = false;
    styleOpen.value = false;
    closeDrawer();
  }
}
function onVisibility(): void {
  if (document.visibilityState === 'visible') void notifications.load().catch(() => undefined);
}

onMounted(() => {
  mobileQuery = window.matchMedia('(max-width: 1023px)');
  wideQuery = window.matchMedia('(min-width: 1280px)');
  syncViewport();
  mobileQuery.addEventListener('change', syncViewport);
  wideQuery.addEventListener('change', syncViewport);

  void bootstrapData();
  pollTimer = window.setInterval(() => {
    if (document.visibilityState === 'visible') void notifications.load().catch(() => undefined);
  }, 60000);
  window.addEventListener('online', onOnline);
  window.addEventListener('offline', onOffline);
  document.addEventListener('click', onDocumentClick);
  document.addEventListener('keydown', onKeydown);
  document.addEventListener('visibilitychange', onVisibility);
});

onBeforeUnmount(() => {
  if (pollTimer) window.clearInterval(pollTimer);
  mobileQuery?.removeEventListener('change', syncViewport);
  wideQuery?.removeEventListener('change', syncViewport);
  window.removeEventListener('online', onOnline);
  window.removeEventListener('offline', onOffline);
  document.removeEventListener('click', onDocumentClick);
  document.removeEventListener('keydown', onKeydown);
  document.removeEventListener('visibilitychange', onVisibility);
  document.body.style.overflow = '';
});

watch(() => auth.user?.id, () => {
  if (auth.user) void bootstrapData();
  else theme.reset();
});

watch(() => route.fullPath, () => {
  drawerOpen.value = false;
  menuOpen.value = false;
  styleOpen.value = false;
});

watch(drawerOpen, (open) => {
  document.body.style.overflow = open && isMobile.value ? 'hidden' : '';
});

function go(item: { name: string; query?: Record<string, string> }): void {
  createOpen.value = false;
  closeDrawer(false);
  void router.push(item);
}

function handleNavClick(): void {
  closeDrawer(false);
}

async function logout(): Promise<void> {
  menuOpen.value = false;
  await auth.logout();
  family.reset();
  notifications.reset();
  toast.push('已退出登录');
  await router.replace({ name: 'login' });
}
</script>

<template>
  <div class="shell" :class="{ 'is-collapsed': collapsed, 'is-drawer-open': drawerOpen }">
    <div
      class="shell__canvas"
      aria-hidden="true"
      :style="{
        backgroundImage: theme.pageImage ? `url('${theme.pageImage}')` : undefined,
        filter: theme.pageImage ? `blur(${theme.pageBlur}px)` : undefined,
      }"
    />
    <a class="shell__skip" href="#main">跳到主要内容</a>

    <aside id="app-sidebar" class="sidebar" aria-label="主导航">
      <div class="sidebar__head">
        <RouterLink to="/" class="brand" aria-label="家时光首页" :title="collapsed ? '家时光' : undefined">
          <img src="/icons/icon-192.png" alt="" width="30" height="30" class="brand__mark" />
          <span class="brand__name at-display">家时光</span>
        </RouterLink>
        <button
          type="button"
          class="sidebar__toggle"
          :title="collapsed ? '展开侧边栏' : '收起侧边栏'"
          :aria-label="collapsed ? '展开侧边栏' : '收起侧边栏'"
          @click="toggleSidebar"
        >
          <AppIcon :name="collapsed ? 'sidebar-expand' : 'sidebar-collapse'" :size="18" />
        </button>
        <button type="button" class="sidebar__close" aria-label="关闭导航" @click="closeDrawer()">
          <AppIcon name="close" :size="20" />
        </button>
      </div>

      <nav class="sidebar__nav">
        <RouterLink
          v-for="item in navItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="sidebar__link"
          :class="{ 'is-active': currentName === item.name }"
          :title="collapsed ? item.label : undefined"
          @click="handleNavClick"
        >
          <AppIcon :name="item.icon" :size="20" />
          <span class="sidebar__label">{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar__foot">
        <RouterLink
          v-for="item in utilityItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="sidebar__link"
          :class="{ 'is-active': currentName === item.name }"
          :title="collapsed ? item.label : undefined"
          @click="handleNavClick"
        >
          <AppIcon :name="item.icon" :size="20" />
          <span class="sidebar__label">{{ item.label }}</span>
        </RouterLink>

        <div class="sidebar__user">
          <AppAvatar :name="initials" :size="32" :src="avatarUrl(auth.user?.avatarKey)" />
          <span class="sidebar__user-meta">
            <span class="sidebar__user-name">{{ auth.user?.displayName }}</span>
            <span class="sidebar__user-role">{{ roleLabel }}</span>
          </span>
        </div>
      </div>
    </aside>

    <Transition name="scrim">
      <div v-if="drawerOpen" class="shell__scrim" @click="closeDrawer()" />
    </Transition>

    <div class="shell__content">
      <header class="topbar">
        <div class="topbar__inner">
          <button
            ref="menuButton"
            type="button"
            class="topbar__menu"
            aria-label="打开导航"
            aria-controls="app-sidebar"
            :aria-expanded="drawerOpen"
            @click="drawerOpen = true"
          >
            <AppIcon name="menu" :size="22" />
          </button>

          <p class="topbar__title at-display">{{ title }}</p>

          <div class="topbar__actions">
            <div ref="styleWrap" class="style-menu">
              <AppIconButton
                icon="palette"
                label="样式"
                :aria-expanded="styleOpen"
                @click.stop="styleOpen = !styleOpen"
              />
              <Transition name="menu">
                <StyleQuickMenu v-if="styleOpen" />
              </Transition>
            </div>
            <AppIconButton icon="search" label="搜索" @click="searchOpen = true" />
            <div class="bell">
              <AppIconButton icon="bell" label="消息通知" @click="notifOpen = true" />
              <span v-if="notifications.unread" class="bell__badge">{{ notifications.unread > 99 ? '99+' : notifications.unread }}</span>
            </div>
            <div ref="avatarWrap" class="avatar-menu">
              <button
                type="button"
                class="avatar-menu__trigger"
                :aria-label="`账号菜单，${auth.user?.displayName ?? ''}`"
                :aria-expanded="menuOpen"
                @click.stop="menuOpen = !menuOpen"
              >
                <AppAvatar :name="initials" :size="32" :src="avatarUrl(auth.user?.avatarKey)" />
              </button>
              <Transition name="menu">
                <div v-if="menuOpen" class="avatar-menu__panel" role="menu">
                  <div class="avatar-menu__head">
                    <span class="avatar-menu__name">{{ auth.user?.displayName }}</span>
                    <span class="avatar-menu__role">{{ roleLabel }}</span>
                  </div>
                  <RouterLink to="/settings" class="avatar-menu__item" role="menuitem" @click="menuOpen = false">
                    <AppIcon name="settings" :size="16" />设置
                  </RouterLink>
                  <RouterLink to="/change-password" class="avatar-menu__item" role="menuitem" @click="menuOpen = false">
                    <AppIcon name="lock" :size="16" />修改密码
                  </RouterLink>
                  <button type="button" class="avatar-menu__item avatar-menu__item--danger" role="menuitem" @click="logout">
                    <AppIcon name="logout" :size="16" />退出登录
                  </button>
                </div>
              </Transition>
            </div>
          </div>
        </div>
      </header>

      <div v-if="!online" class="offline" role="status">
        <AppIcon name="wifi-off" :size="16" />
        <span>当前处于离线状态，部分操作可能无法保存</span>
      </div>

      <main id="main" class="shell__main" :aria-label="title">
        <slot />
      </main>

      <button v-if="showCreateFab" type="button" class="fab" aria-label="快速新建" @click="createOpen = true">
        <AppIcon name="plus" :size="22" />
      </button>
    </div>

    <AppDialog v-model="createOpen" title="快速新建" size="sm">
      <div class="quick">
        <button v-for="action in createActions" :key="action.label" type="button" class="quick__item" @click="go(action.to)">
          <span class="quick__icon"><AppIcon :name="action.icon" :size="20" /></span>
          <span>{{ action.label }}</span>
          <AppIcon name="chevron-right" :size="16" class="quick__chevron" />
        </button>
      </div>
    </AppDialog>

    <SearchDialog v-model="searchOpen" />
    <NotificationPanel v-model="notifOpen" />
  </div>
</template>

<style scoped>
.shell {
  --sidebar-w: var(--at-sidebar-width);
  position: relative;
  min-height: 100vh;
  display: flex;
  isolation: isolate;
}

.shell__canvas {
  position: fixed;
  inset: -40px;
  z-index: -1;
  background-size: cover;
  background-position: center;
  pointer-events: none;
  opacity: 0.72;
}

.shell.is-collapsed {
  --sidebar-w: var(--at-sidebar-width-collapsed);
}

.shell__skip {
  position: absolute;
  left: var(--at-space-4);
  top: -60px;
  z-index: var(--at-z-dialog);
  padding: var(--at-space-2) var(--at-space-4);
  background: var(--at-bg-dark);
  color: #fff;
  border-radius: var(--at-radius-md);
  transition: top var(--at-duration-fast) var(--at-ease);
}

.shell__skip:focus {
  top: var(--at-space-2);
}

.sidebar {
  position: sticky;
  top: 0;
  flex: none;
  display: flex;
  flex-direction: column;
  width: var(--sidebar-w);
  height: 100vh;
  padding: var(--at-space-3);
  background: var(--at-bg-surface);
  background: var(--at-bg-chrome);
  border-right: 1px solid var(--at-color-hairline);
  transition: width var(--at-duration-base) var(--at-ease), transform var(--at-duration-base) var(--at-ease);
  z-index: var(--at-z-nav);
}

.sidebar__head {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  padding: var(--at-space-2) var(--at-space-2) var(--at-space-4);
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 2px;
  border-radius: var(--at-radius-md);
}

.brand__mark {
  border-radius: 8px;
  flex: none;
}

.brand__name {
  font-size: 22px;
  letter-spacing: 0.6px;
  color: var(--at-color-ink);
  white-space: nowrap;
}

.sidebar__toggle {
  display: grid;
  place-items: center;
  flex: none;
  margin-left: auto;
  width: 32px;
  height: 32px;
  border-radius: var(--at-radius-md);
  color: var(--at-color-muted);
}

.sidebar__toggle:hover {
  background: var(--at-bg-canvas-soft);
  color: var(--at-color-ink);
}

.sidebar__close {
  display: none;
  margin-left: auto;
  width: var(--at-touch-min);
  height: var(--at-touch-min);
  place-items: center;
  border-radius: 50%;
  color: var(--at-color-muted);
}

.sidebar__close:hover {
  background: var(--at-bg-canvas-soft);
  color: var(--at-color-ink);
}

.sidebar__nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
  overflow-x: hidden;
}

.sidebar__foot {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: auto;
  padding-top: var(--at-space-3);
  border-top: 1px solid var(--at-color-hairline-soft);
}

.sidebar__link {
  position: relative;
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  width: 100%;
  min-height: var(--at-touch-min);
  padding: 0 var(--at-space-3);
  border-radius: var(--at-radius-md);
  color: var(--at-color-muted);
  font-size: var(--at-text-caption);
  font-weight: 500;
  white-space: nowrap;
  transition: background var(--at-duration-fast) var(--at-ease), color var(--at-duration-fast) var(--at-ease);
}

.sidebar__link:hover {
  background: var(--at-bg-canvas-soft);
  color: var(--at-color-ink);
}

.sidebar__link.is-active {
  background: var(--at-bg-surface-strong);
  color: var(--at-color-ink);
}

.sidebar__link.is-active::before {
  content: '';
  position: absolute;
  left: calc(var(--at-space-3) * -1);
  top: 10px;
  bottom: 10px;
  width: 3px;
  border-radius: 0 var(--at-radius-pill) var(--at-radius-pill) 0;
  background: var(--at-color-ink);
}

.sidebar__label {
  overflow: hidden;
  text-overflow: ellipsis;
}

.sidebar__user {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  padding: var(--at-space-2) var(--at-space-2);
  margin-top: var(--at-space-2);
}

.sidebar__user-meta {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.3;
}

.sidebar__user-name {
  font-size: var(--at-text-caption);
  font-weight: 500;
  color: var(--at-color-ink);
  overflow: hidden;
  text-overflow: ellipsis;
}

.sidebar__user-role {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.shell.is-collapsed .sidebar {
  padding-left: var(--at-space-2);
  padding-right: var(--at-space-2);
}

.shell.is-collapsed .sidebar__head {
  justify-content: center;
  gap: 2px;
  padding-left: 0;
  padding-right: 0;
}

.shell.is-collapsed .brand__mark {
  width: 26px;
  height: 26px;
}

.shell.is-collapsed .sidebar__toggle {
  width: 28px;
  height: 28px;
  margin-left: 0;
}

.shell.is-collapsed .brand__name,
.shell.is-collapsed .sidebar__label,
.shell.is-collapsed .sidebar__user-meta {
  display: none;
}

.shell.is-collapsed .sidebar__link {
  justify-content: center;
  padding: 0;
}

.shell.is-collapsed .sidebar__link.is-active::before {
  left: calc(var(--at-space-2) * -1);
}

.shell.is-collapsed .sidebar__user {
  justify-content: center;
  padding-left: 0;
  padding-right: 0;
}

.shell__content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.topbar {
  position: sticky;
  top: 0;
  z-index: var(--at-z-nav);
  border-bottom: 1px solid var(--at-color-hairline);
  background: rgba(255, 255, 255, 0.92);
  background: color-mix(in srgb, var(--at-bg-chrome) 90%, transparent);
  backdrop-filter: saturate(1.1) blur(10px);
}

.topbar__inner {
  height: var(--at-topbar-height);
  padding: 0 var(--at-space-6);
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
}

.topbar__menu {
  display: none;
  place-items: center;
  width: var(--at-touch-min);
  height: var(--at-touch-min);
  margin-left: calc(var(--at-space-2) * -1);
  border-radius: var(--at-radius-pill);
  color: var(--at-color-body-strong);
}

.topbar__menu:hover {
  background: var(--at-bg-surface-strong);
  color: var(--at-color-ink);
}

.topbar__title {
  font-size: var(--at-text-title-sm);
  color: var(--at-color-ink);
}

.topbar__actions {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-left: auto;
}

.bell {
  position: relative;
}

.style-menu {
  position: relative;
}

.bell__badge {
  position: absolute;
  top: 5px;
  right: 4px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: var(--at-radius-pill);
  background: var(--at-color-brand);
  color: #fff;
  font-size: 10px;
  font-weight: 600;
  line-height: 16px;
  text-align: center;
  pointer-events: none;
}

.avatar-menu {
  position: relative;
}

.avatar-menu__trigger {
  display: grid;
  place-items: center;
  padding: 6px;
  border-radius: 50%;
}

.avatar-menu__trigger:hover {
  background: var(--at-bg-surface-strong);
}

.avatar-menu__panel {
  position: absolute;
  right: 0;
  top: calc(100% + 10px);
  width: 200px;
  padding: var(--at-space-2);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-lg);
  box-shadow: var(--at-shadow-dialog);
  z-index: var(--at-z-dialog);
}

.avatar-menu__head {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--at-space-2) var(--at-space-3) var(--at-space-3);
  border-bottom: 1px solid var(--at-color-hairline);
  margin-bottom: var(--at-space-1);
}

.avatar-menu__name {
  font-size: var(--at-text-body-sm);
  font-weight: 500;
}

.avatar-menu__role {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.avatar-menu__item {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  width: 100%;
  padding: 9px var(--at-space-3);
  border-radius: var(--at-radius-md);
  font-size: var(--at-text-caption);
  text-align: left;
}

.avatar-menu__item:hover {
  background: var(--at-bg-canvas-soft);
}

.avatar-menu__item--danger {
  color: var(--at-color-danger);
}

.menu-enter-active,
.menu-leave-active {
  transition: opacity var(--at-duration-fast) var(--at-ease), transform var(--at-duration-fast) var(--at-ease);
}

.menu-enter-from,
.menu-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.offline {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--at-space-2);
  padding: 6px var(--at-space-4);
  background: #fffbeb;
  color: #92400e;
  font-size: var(--at-text-micro);
}

.shell__main {
  flex: 1;
  min-width: 0;
}

.fab {
  position: fixed;
  right: max(24px, calc((100vw - var(--sidebar-w) - var(--at-content-max)) / 2 + 24px));
  bottom: 32px;
  z-index: var(--at-z-sticky);
  width: 52px;
  height: 52px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: var(--at-color-primary);
  color: var(--at-text-on-primary);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
  transition: transform var(--at-duration-fast) var(--at-ease), background var(--at-duration-fast) var(--at-ease);
}

.fab:hover {
  background: var(--at-color-primary-active);
  transform: translateY(-1px);
}

.shell__scrim {
  display: none;
}

.quick {
  display: flex;
  flex-direction: column;
}

.quick__item {
  display: flex;
  align-items: center;
  gap: var(--at-space-3);
  padding: var(--at-space-3);
  border-radius: var(--at-radius-md);
  font-size: var(--at-text-body-sm);
  text-align: left;
}

.quick__item:hover {
  background: var(--at-bg-canvas-soft);
}

.quick__icon {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: var(--at-bg-surface-strong);
}

.quick__chevron {
  margin-left: auto;
  color: var(--at-color-muted-soft);
}

@media (max-width: 1023px) {
  .shell {
    --sidebar-w: var(--at-sidebar-mobile-width);
    display: block;
  }

  .sidebar {
    position: fixed;
    top: 0;
    bottom: 0;
    left: 0;
    height: 100dvh;
    width: var(--at-sidebar-mobile-width);
    transform: translateX(-100%);
    box-shadow: var(--at-shadow-dialog);
    z-index: calc(var(--at-z-overlay) + 1);
  }

  .shell.is-drawer-open .sidebar {
    transform: none;
  }

  .shell__scrim {
    display: block;
    position: fixed;
    inset: 0;
    z-index: var(--at-z-overlay);
    background: var(--at-color-scrim);
    backdrop-filter: blur(2px);
  }

  .scrim-enter-active,
  .scrim-leave-active {
    transition: opacity var(--at-duration-base) var(--at-ease);
  }

  .scrim-enter-from,
  .scrim-leave-to {
    opacity: 0;
  }

  .sidebar__close {
    display: grid;
  }

  .sidebar__toggle {
    display: none;
  }

  .topbar__menu {
    display: grid;
  }

  .topbar__inner {
    padding: 0 var(--at-space-4);
  }

  .topbar__title {
    font-size: var(--at-text-body-sm);
  }

  .fab {
    right: 20px;
    bottom: calc(24px + env(safe-area-inset-bottom));
  }
}

@media (max-width: 639px) {
  .topbar__inner {
    height: 56px;
  }

  .topbar__title {
    font-size: var(--at-text-caption);
  }
}
</style>
