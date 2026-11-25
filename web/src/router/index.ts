import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean;
    layout?: 'auth' | 'app';
    title?: string;
  }
}

const routes: RouteRecordRaw[] = [
  { path: '/setup', name: 'setup', component: () => import('@/views/SetupView.vue'), meta: { public: true, layout: 'auth', title: '初始化' } },
  { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue'), meta: { public: true, layout: 'auth', title: '登录' } },
  { path: '/change-password', name: 'change-password', component: () => import('@/views/ChangePasswordView.vue'), meta: { title: '修改密码' } },
  { path: '/', name: 'home', component: () => import('@/views/HomeView.vue'), meta: { title: '首页' } },
  { path: '/calendar', name: 'calendar', component: () => import('@/views/CalendarView.vue'), meta: { title: '行事历' } },
  { path: '/tasks', name: 'tasks', component: () => import('@/views/TasksView.vue'), meta: { title: '待办' } },
  { path: '/timeline', name: 'timeline', component: () => import('@/views/TimelineView.vue'), meta: { title: '时光轴' } },
  { path: '/family', name: 'family', component: () => import('@/views/FamilyView.vue'), meta: { title: '家庭' } },
  { path: '/contacts', name: 'contacts', component: () => import('@/views/ContactsView.vue'), meta: { title: '联系人' } },
  { path: '/contacts/:id', name: 'contact-detail', component: () => import('@/views/ContactDetailView.vue'), meta: { title: '联系人详情' } },
  { path: '/messages', name: 'messages', component: () => import('@/views/MessagesView.vue'), meta: { title: '留言板' } },
  { path: '/notes', name: 'notes', component: () => import('@/views/NotesView.vue'), meta: { title: '便签' } },
  { path: '/settings', name: 'settings', component: () => import('@/views/SettingsView.vue'), meta: { title: '设置' } },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/views/NotFoundView.vue'), meta: { title: '未找到' } },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, saved) {
    if (saved) return saved;
    if (to.fullPath === from.fullPath) return false;
    return { top: 0 };
  },
});

router.beforeEach(async (to) => {
  const auth = useAuthStore();
  try {
    await auth.ensure();
  } catch {
    if (to.meta.public) return true;
    return { name: 'login', query: { redirect: to.fullPath } };
  }
  if (auth.needsSetup && to.name !== 'setup') return { name: 'setup' };
  if (!auth.needsSetup && to.name === 'setup') {
    return auth.isAuthenticated ? { name: 'home' } : { name: 'login' };
  }
  if (to.meta.public) {
    if (auth.isAuthenticated && to.name === 'login') return { name: 'home' };
    return true;
  }
  if (!auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } };
  }
  if (auth.mustChangePassword && to.name !== 'change-password') {
    return { name: 'change-password' };
  }
  return true;
});

router.afterEach((to) => {
  const base = '家时光';
  document.title = to.meta.title ? `${to.meta.title} · ${base}` : base;
});

export default router;
