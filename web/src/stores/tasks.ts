import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { api } from '@/api/client';
import type { Task, TaskTemplate } from '@/types';

export interface TaskCounts {
  open: number;
  completed: number;
  archived: number;
  today: number;
  overdue: number;
}

export interface TaskQuery {
  scope?: 'open' | 'completed' | 'archived' | 'all';
  assignee?: number | 'me';
  priority?: string;
  q?: string;
  templateId?: number;
  from?: string;
  to?: string;
}

export interface SubtaskPayload {
  id?: number;
  title: string;
  done: boolean;
}

export interface TaskPayload {
  title?: string;
  description?: string | null;
  status?: string;
  priority?: string;
  startDate?: string | null;
  dueDate?: string | null;
  dueTime?: string | null;
  templateId?: number | null;
  recurrenceRule?: string | null;
  color?: string | null;
  visibility?: string;
  assigneeIds?: number[];
  subtasks?: SubtaskPayload[];
  reminderOffsets?: number[];
  archived?: boolean;
}

export interface CompletionResult {
  task: Task;
  nextTask: Task | null;
  completion: Task['completion'];
}

export interface TemplatePayload {
  name: string;
  title: string;
  description?: string | null;
  priority?: string | null;
  color?: string | null;
  defaultAssigneeIds?: number[];
  defaultDueTime?: string | null;
  recurrenceRule?: string | null;
  reminderOffset?: number | null;
  subtaskTemplate?: string[];
  enabled?: boolean;
  sortOrder?: number;
}

const emptyCounts: TaskCounts = { open: 0, completed: 0, archived: 0, today: 0, overdue: 0 };

export const useTasksStore = defineStore('tasks', () => {
  const tasks = ref<Task[]>([]);
  const counts = ref<TaskCounts>({ ...emptyCounts });
  const today = ref('');
  const loading = ref(false);
  const error = ref<unknown>(null);
  const templates = ref<TaskTemplate[]>([]);
  const templatesLoaded = ref(false);

  const openTasks = computed(() => tasks.value.filter((t) => !t.archivedAt && t.status !== 'completed'));

  async function load(query: TaskQuery = { scope: 'open' }): Promise<Task[]> {
    loading.value = true;
    error.value = null;
    try {
      const res = await api.get<{ items: Task[]; counts: TaskCounts; today: string }>('/tasks', {
        scope: query.scope ?? 'open',
        assignee: query.assignee,
        priority: query.priority,
        q: query.q,
        templateId: query.templateId,
        from: query.from,
        to: query.to,
        limit: 200,
      });
      tasks.value = res.items;
      counts.value = res.counts;
      today.value = res.today;
      return res.items;
    } catch (err) {
      error.value = err;
      throw err;
    } finally {
      loading.value = false;
    }
  }

  async function getTask(id: number): Promise<Task> {
    const res = await api.get<{ task: Task }>(`/tasks/${id}`);
    return res.task;
  }

  async function createTask(payload: TaskPayload): Promise<Task> {
    const res = await api.post<{ task: Task }>('/tasks', payload);
    return res.task;
  }

  async function updateTask(id: number, payload: TaskPayload): Promise<Task> {
    const res = await api.patch<{ task: Task }>(`/tasks/${id}`, payload);
    return res.task;
  }

  async function deleteTask(id: number): Promise<void> {
    await api.del(`/tasks/${id}`);
  }

  async function completeTask(id: number, actualUserId?: number): Promise<CompletionResult> {
    return api.post<CompletionResult>(`/tasks/${id}/completion`, {
      actualUserId: actualUserId ?? null,
    });
  }

  async function undoTask(id: number): Promise<{ task: Task; nextTaskRemoved: boolean }> {
    return api.del<{ task: Task; nextTaskRemoved: boolean }>(`/tasks/${id}/completion`);
  }

  async function toggleSubtask(taskId: number, subtaskId: number, done: boolean): Promise<Task> {
    const res = await api.patch<{ task: Task }>(`/tasks/${taskId}/subtasks/${subtaskId}`, { done });
    return res.task;
  }

  async function loadTemplates(force = false): Promise<TaskTemplate[]> {
    if (templatesLoaded.value && !force) return templates.value;
    const res = await api.get<{ items: TaskTemplate[] }>('/templates');
    templates.value = res.items;
    templatesLoaded.value = true;
    return templates.value;
  }

  async function saveTemplate(payload: TemplatePayload, id?: number): Promise<TaskTemplate> {
    const res = id
      ? await api.patch<{ template: TaskTemplate }>(`/templates/${id}`, payload)
      : await api.post<{ template: TaskTemplate }>('/templates', payload);
    if (id) {
      templates.value = templates.value.map((t) => (t.id === id ? res.template : t));
    } else {
      templates.value = [...templates.value, res.template];
    }
    return res.template;
  }

  async function deleteTemplate(id: number): Promise<void> {
    await api.del(`/templates/${id}`);
    templates.value = templates.value.filter((t) => t.id !== id);
  }

  function reset(): void {
    tasks.value = [];
    counts.value = { ...emptyCounts };
    templates.value = [];
    templatesLoaded.value = false;
    error.value = null;
  }

  return {
    tasks,
    openTasks,
    counts,
    today,
    loading,
    error,
    templates,
    templatesLoaded,
    load,
    getTask,
    createTask,
    updateTask,
    deleteTask,
    completeTask,
    undoTask,
    toggleSubtask,
    loadTemplates,
    saveTemplate,
    deleteTemplate,
    reset,
  };
});
