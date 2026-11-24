<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { RouterLink, useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { useFamilyStore, useNotificationStore } from '@/stores/family';
import { useToastStore } from '@/stores/toast';
import AppAvatar from './base/AppAvatar.vue';
import AppButton from './base/AppButton.vue';
import AppDialog from './base/AppDialog.vue';
import AppIcon from './base/AppIcon.vue';
import AppIconButton from './base/AppIconButton.vue';
import SearchDialog from './SearchDialog.vue';
import NotificationPanel from './NotificationPanel.vue';
import type { IconName } from './base/icons';
import { todayISO } from '@/utils/chrono';

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

const moreItems: NavItem[] = [
  { name: 'family', label: '家庭成员', icon: 'family' },
  { name: 'messages', label: '留言板', icon: 'messages' },
  { name: 'notes', label: '便签', icon: 'notes' },
  { name: 'settings', label: '设置', icon: 'settings' },
];

const bottomItems: NavItem[] = [
  { name: 'home', label: '首页', icon: 'home' },
  { name: 'calendar', label: '行事历', icon: 'calendar' },
  { name: 'tasks', label: '待办', icon: 'tasks' },
  { name: 'timeline', label: '时光轴', icon: 'timeline' },
];

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const family = useFamilyStore();
const notifications = useNotificationStore();
const toast = useToastStore();

const searchOpen = ref(false);
const notifOpen = ref(false);
const moreOpen = ref(false);
const createOpen = ref(false);
const menuOpen = ref(false);
const online = ref(typeof navigator === 'undefined' ? true : navigator.onLine);
const avatarWrap = ref<HTMLElement | null>(null);

const currentName = computed(() => route.name?.toString() ?? '');
const title = computed(() => (route.meta.title ? String(route.meta.title) : '家时光'));
const initials = computed(() => auth.user?.displayName ?? '家');

const createActions: { label: string; icon: IconName; to: { name: string; query?: Record<string, string> } }[] = [
  { label: '新建待办', icon: 'tasks', to: { name: 'tasks', query: { new: '1' } } },
  { label: '新建日程', icon: 'calendar', to: { name: 'calendar', query: { new: '1', date: todayISO() } } },
  { label: '写一条留言', icon: 'messages', to: { name: 'messages', query: { new: '1' } } },
  { label: '新建便签', icon: 'notes', to: { name: 'notes', query: { new: '1' } } },
];

let pollTimer: number | undefined;

async function bootstrapData(): Promise<void> {
  if (!auth.user) return;
  await Promise.allSettled([family.load(), notifications.load()]);
}

function onOnline(): void {
  online.value = navigator.onLine;
}
function onOffline(): void {
  online.value = navigator.onLine;
}
function onDocumentClick(event: MouseEvent): void {
  if (avatarWrap.value && !avatarWrap.value.contains(event.target as Node)) menuOpen.value = false;
}
function onVisibility(): void {
  if (document.visibilityState === 'visible') void notifications.load().catch(() => undefined);
}

onMounted(() => {
  void bootstrapData();
  pollTimer = window.setInterval(() => {
    if (document.visibilityState === 'visible') void notifications.load().catch(() => undefined);
  }, 60000);
  window.addEventListener('online', onOnline);
  window.addEventListener('offline', onOffline);
  document.addEventListener('click', onDocumentClick);
  document.addEventListener('visibilitychange', onVisibility);
});

onBeforeUnmount(() => {
  if (pollTimer) window.clearInterval(pollTimer);
  window.removeEventListener('online', onOnline);
  window.removeEventListener('offline', onOffline);
  document.removeEventListener('click', onDocumentClick);
  document.removeEventListener('visibilitychange', onVisibility);
});

watch(() => auth.user?.id, () => {
  if (auth.user) void bootstrapData();
});

function go(item: { name: string; query?: Record<string, string> }): void {
  createOpen.value = false;
  moreOpen.value = false;
  void router.push(item);
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
  <div class="shell">
    <a class="shell__skip" href="#main">跳到主要内容</a>
    <header class="topbar">
      <div class="topbar__inner">
        <RouterLink to="/" class="brand" aria-label="家时光首页">
          <img src="/icons/icon-192.png" alt="" width="30" height="30" class="brand__mark" />
          <span class="brand__name at-display">家时光</span>
        </RouterLink>

        <nav class="topnav at-hide-mobile" aria-label="主导航">
          <RouterLink
            v-for="item in navItems"
            :key="item.name"
            :to="{ name: item.name }"
            class="topnav__link"
            :class="{ 'is-active': currentName === item.name }"
          >
            {{ item.label }}
          </RouterLink>
        </nav>

        <div class="topbar__actions">
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
              <AppAvatar :name="initials" :size="32" />
            </button>
            <Transition name="menu">
              <div v-if="menuOpen" class="avatar-menu__panel" role="menu">
                <div class="avatar-menu__head">
                  <span class="avatar-menu__name">{{ auth.user?.displayName }}</span>
                  <span class="avatar-menu__role">{{ auth.user?.role === 'admin' ? '管理员' : '家庭成员' }}</span>
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

    <button type="button" class="fab at-hide-mobile" aria-label="快速新建" @click="createOpen = true">
      <AppIcon name="plus" :size="22" />
    </button>

    <nav class="bottomnav" aria-label="移动端导航">
      <RouterLink
        v-for="item in bottomItems"
        :key="item.name"
        :to="{ name: item.name }"
        class="bottomnav__item"
        :class="{ 'is-active': currentName === item.name }"
      >
        <AppIcon :name="item.icon" :size="21" />
        <span>{{ item.label }}</span>
      </RouterLink>
      <button type="button" class="bottomnav__item" :class="{ 'is-active': moreOpen }" @click="moreOpen = true">
        <AppIcon name="menu" :size="21" />
        <span>更多</span>
      </button>
    </nav>

    <AppDialog v-model="createOpen" title="快速新建" size="sm">
      <div class="quick">
        <button v-for="action in createActions" :key="action.label" type="button" class="quick__item" @click="go(action.to)">
          <span class="quick__icon"><AppIcon :name="action.icon" :size="20" /></span>
          <span>{{ action.label }}</span>
          <AppIcon name="chevron-right" :size="16" class="quick__chevron" />
        </button>
      </div>
    </AppDialog>

    <AppDialog v-model="moreOpen" title="更多" size="sm">
      <div class="more">
        <section>
          <p class="at-caption-upper more__label">前往</p>
          <div class="more__grid">
            <RouterLink v-for="item in moreItems" :key="item.name" :to="{ name: item.name }" class="more__item" @click="moreOpen = false">
              <AppIcon :name="item.icon" :size="20" />
              <span>{{ item.label }}</span>
            </RouterLink>
          </div>
        </section>
        <section>
          <p class="at-caption-upper more__label">快速新建</p>
          <div class="more__actions">
            <AppButton v-for="action in createActions" :key="action.label" variant="outline" size="sm" @click="go(action.to)">
              {{ action.label }}
            </AppButton>
          </div>
        </section>
      </div>
    </AppDialog>

    <SearchDialog v-model="searchOpen" />
    <NotificationPanel v-model="notifOpen" />
  </div>
</template>

<style scoped>
.shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
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

.topbar {
  position: sticky;
  top: 0;
  z-index: var(--at-z-nav);
  border-bottom: 1px solid var(--at-color-hairline);
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: saturate(1.1) blur(10px);
}

.topbar__inner {
  max-width: var(--at-content-max);
  height: var(--at-topbar-height);
  margin: 0 auto;
  padding: 0 var(--at-space-6);
  display: flex;
  align-items: center;
  gap: var(--at-space-6);
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  flex: none;
}

.brand__mark {
  border-radius: 8px;
}

.brand__name {
  font-size: 22px;
  letter-spacing: 0.6px;
  color: var(--at-color-ink);
}

.topnav {
  display: flex;
  align-items: center;
  gap: 2px;
  flex: 1;
}

.topnav__link {
  padding: 8px 12px;
  border-radius: var(--at-radius-pill);
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  transition: color var(--at-duration-fast) var(--at-ease), background var(--at-duration-fast) var(--at-ease);
}

.topnav__link:hover {
  color: var(--at-color-ink);
  background: var(--at-bg-canvas-soft);
}

.topnav__link.is-active {
  color: var(--at-color-ink);
  background: var(--at-bg-surface-strong);
  font-weight: 500;
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
  right: max(24px, calc((100vw - var(--at-content-max)) / 2 + 24px));
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

.bottomnav {
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

.more {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-5);
}

.more__label {
  margin-bottom: var(--at-space-2);
}

.more__grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--at-space-2);
}

.more__item {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  padding: var(--at-space-3);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-md);
  font-size: var(--at-text-caption);
}

.more__item:hover {
  background: var(--at-bg-canvas-soft);
}

.more__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-2);
}

@media (max-width: 1023px) {
  .topbar__inner {
    padding: 0 var(--at-space-4);
    gap: var(--at-space-3);
  }

  .brand__name {
    font-size: 20px;
  }
}

@media (max-width: 639px) {
  .topbar__inner {
    height: 56px;
  }

  .shell__main {
    padding-bottom: 0;
  }

  .fab {
    display: none;
  }

  .bottomnav {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: var(--at-z-nav);
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    padding: 6px 4px calc(6px + env(safe-area-inset-bottom));
    background: rgba(255, 255, 255, 0.96);
    border-top: 1px solid var(--at-color-hairline);
    backdrop-filter: blur(10px);
  }

  .bottomnav__item {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 3px;
    min-height: 48px;
    border-radius: var(--at-radius-md);
    font-size: 11px;
    color: var(--at-color-muted);
  }

  .bottomnav__item.is-active {
    color: var(--at-color-ink);
    font-weight: 600;
  }
}
</style>
