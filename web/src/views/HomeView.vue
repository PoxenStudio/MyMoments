<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { useTasksStore } from '@/stores/tasks';
import { useToastStore } from '@/stores/toast';
import { ApiError } from '@/api/client';
import { todayISO } from '@/utils/chrono';
import { greetingFor } from '@/utils/task';
import AppButton from '@/components/base/AppButton.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import TaskFormDialog from '@/components/business/TaskFormDialog.vue';
import TaskItem from '@/components/business/TaskItem.vue';
import type { Task } from '@/types';

const auth = useAuthStore();
const tasks = useTasksStore();
const toast = useToastStore();

const formOpen = ref(false);
const today = computed(() => tasks.today || todayISO());

const greeting = computed(() => greetingFor(auth.user?.displayName ?? '家人'));
const dateText = computed(() => new Intl.DateTimeFormat('zh-CN', {
  month: 'long',
  day: 'numeric',
  weekday: 'long',
}).format(new Date()));

const recent = computed(() => tasks.tasks.slice(0, 6));
const overdue = computed(() => tasks.tasks.filter((t) => t.dueDate && t.dueDate < today.value).length);
const todayCount = computed(() => tasks.tasks.filter((t) => t.dueDate === today.value).length);
const summary = computed(() => {
  if (!tasks.counts.open) return '今天没有待办，享受一点空闲。';
  if (overdue.value) return `有 ${overdue.value} 件已经逾期，先从最小的一件开始。`;
  if (todayCount.value) return `今天有 ${todayCount.value} 件待办，慢慢来。`;
  return `还有 ${tasks.counts.open} 件待办在后面等着。`;
});

onMounted(() => {
  void tasks.load({ scope: 'open' }).catch(() => undefined);
});

async function completeTask(task: Task): Promise<void> {
  try {
    const result = await tasks.completeTask(task.id);
    await tasks.load({ scope: 'open' });
    const next = result.nextTask?.dueDate ? `，下次 ${result.nextTask.dueDate.slice(5)}` : '';
    toast.undo(`已完成「${task.title}」${next}`, () => {
      void undoTask(task);
    });
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '完成待办失败');
  }
}

async function undoTask(task: Task): Promise<void> {
  try {
    const result = await tasks.undoTask(task.id);
    await tasks.load({ scope: 'open' });
    toast.success(result.nextTaskRemoved ? '已撤销完成并撤回下一次待办' : '已撤销完成');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '撤销失败');
  }
}

async function onSaved(): Promise<void> {
  toast.success('待办已创建');
  await tasks.load({ scope: 'open' });
}
</script>

<template>
  <div class="home">
    <header class="hero">
      <p class="at-caption-upper hero__date">{{ dateText }}</p>
      <h1 class="at-display hero__title">{{ greeting }}</h1>
      <p class="hero__sub">{{ summary }}</p>
      <div class="hero__actions">
        <AppButton @click="formOpen = true"><AppIcon name="plus" :size="16" />新建待办</AppButton>
        <RouterLink to="/tasks" class="hero__link">全部待办<AppIcon name="arrow-right" :size="15" /></RouterLink>
      </div>
    </header>

    <div class="home__grid">
      <section class="panel">
        <div class="panel__head">
          <div>
            <p class="at-caption-upper">Today</p>
            <h2 class="panel__title">近期待办</h2>
          </div>
          <RouterLink to="/tasks" class="panel__more">查看全部</RouterLink>
        </div>
        <div v-if="tasks.loading && !tasks.tasks.length" class="panel__loading">
          <span v-for="n in 3" :key="n" class="panel__loading-row" />
        </div>
        <div v-else-if="recent.length" class="panel__list">
          <TaskItem
            v-for="task in recent"
            :key="task.id"
            :task="task"
            :today="today"
            compact
            @complete="completeTask"
            @undo="undoTask"
            @open="$router.push({ name: 'tasks', query: { open: String(task.id) } })"
          />
        </div>
        <AppEmptyState v-else title="今天没有待办" description="把想做的事记下来，家里人会看到" icon="tasks">
          <AppButton size="sm" variant="outline" @click="formOpen = true">新建一条</AppButton>
        </AppEmptyState>
      </section>

      <aside class="side">
        <section class="side__card">
          <p class="at-caption-upper">Overview</p>
          <dl class="stats">
            <div class="stats__item">
              <dt>待办</dt>
              <dd>{{ tasks.counts.open }}</dd>
            </div>
            <div class="stats__item">
              <dt>今天</dt>
              <dd>{{ tasks.counts.today }}</dd>
            </div>
            <div class="stats__item">
              <dt>逾期</dt>
              <dd :class="{ 'is-danger': tasks.counts.overdue > 0 }">{{ tasks.counts.overdue }}</dd>
            </div>
          </dl>
        </section>

        <section class="side__note">
          <p class="side__quote at-display">把每天的小事，慢慢收好。</p>
          <p class="side__caption">家时光 · MyMoments</p>
        </section>
      </aside>
    </div>

    <TaskFormDialog v-model="formOpen" :templates="tasks.templates" :default-date="today" @saved="onSaved" />
  </div>
</template>

<style scoped>
.home {
  max-width: var(--at-content-max);
  margin: 0 auto;
  padding: var(--at-space-8) var(--at-space-6) var(--at-space-12);
}

.hero {
  padding: var(--at-space-6) 0 var(--at-space-8);
  border-bottom: 1px solid var(--at-color-hairline);
}

.hero__date {
  margin-bottom: var(--at-space-3);
}

.hero__title {
  font-size: var(--at-text-display-xl);
  line-height: 1.15;
}

.hero__sub {
  margin-top: var(--at-space-3);
  font-size: var(--at-text-body-sm);
  color: var(--at-color-muted);
}

.hero__actions {
  display: flex;
  align-items: center;
  gap: var(--at-space-4);
  margin-top: var(--at-space-5);
}

.hero__link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: var(--at-text-caption);
  color: var(--at-color-body);
}

.hero__link:hover {
  color: var(--at-color-ink);
}

.home__grid {
  display: grid;
  grid-template-columns: minmax(0, 1.7fr) minmax(240px, 1fr);
  gap: var(--at-space-6);
  padding-top: var(--at-space-8);
  align-items: start;
}

.panel {
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-xl);
  box-shadow: var(--at-shadow-panel);
  padding: var(--at-space-5) var(--at-space-6);
}

.panel__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--at-space-3);
  padding-bottom: var(--at-space-3);
  border-bottom: 1px solid var(--at-color-hairline);
}

.panel__title {
  margin-top: 2px;
  font-family: var(--at-font-display);
  font-size: 24px;
  font-weight: 300;
  letter-spacing: 0.2px;
}

.panel__more {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.panel__more:hover {
  color: var(--at-color-ink);
}

.panel__list {
  display: flex;
  flex-direction: column;
}

.panel__loading {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-2);
  padding-top: var(--at-space-3);
}

.panel__loading-row {
  height: 44px;
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface-strong);
  animation: pulse 1.4s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}

.side {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.side__card {
  padding: var(--at-space-5);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-xl);
  box-shadow: var(--at-shadow-panel);
}

.stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--at-space-3);
  margin-top: var(--at-space-3);
}

.stats__item dt {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.stats__item dd {
  margin-top: 2px;
  font-family: var(--at-font-display);
  font-size: 32px;
  font-weight: 300;
  color: var(--at-color-ink);
}

.stats__item dd.is-danger {
  color: var(--at-color-danger);
}

.side__note {
  padding: var(--at-space-6);
  background: var(--at-bg-dark);
  border-radius: var(--at-radius-xl);
  color: var(--at-text-on-dark);
}

.side__quote {
  font-size: 24px;
  line-height: 1.5;
  font-weight: 300;
}

.side__caption {
  margin-top: var(--at-space-4);
  font-size: var(--at-text-micro);
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--at-text-on-dark-soft);
}

@media (max-width: 1023px) {
  .home {
    padding: var(--at-space-6) var(--at-space-4) var(--at-space-12);
  }

  .home__grid {
    grid-template-columns: 1fr;
  }

  .hero__title {
    font-size: var(--at-text-display-lg);
  }
}

@media (max-width: 639px) {
  .home {
    padding: var(--at-space-5) var(--at-space-4) 96px;
  }

  .hero {
    padding-top: var(--at-space-2);
  }

  .hero__title {
    font-size: 34px;
  }

  .panel {
    padding: var(--at-space-4);
  }
}
</style>
