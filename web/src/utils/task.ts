import type { Priority, Task, TaskStatus, Visibility } from '@/types';

export interface RecurrenceRule {
  freq: 'daily' | 'weekly' | 'monthly' | 'yearly';
  interval: number;
  weekdays?: number[];
  until?: string;
  anchor?: 'due' | 'completion';
}

export const priorityLabels: Record<Priority, string> = {
  low: '低',
  normal: '普通',
  high: '高',
  urgent: '紧急',
};

export const statusLabels: Record<TaskStatus, string> = {
  todo: '待办',
  in_progress: '进行中',
  completed: '已完成',
};

export const visibilityLabels: Record<Visibility, string> = {
  household: '全家可见',
  selected: '指定成员',
  private: '仅自己',
};

export const taskColors = ['#b45309', '#0f766e', '#7c3aed', '#be123c', '#0369a1', '#57534e'];

const weekNames = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];

export function parseRecurrence(raw: string | null | undefined): RecurrenceRule | null {
  if (!raw) return null;
  try {
    const value = JSON.parse(raw) as RecurrenceRule;
    if (!value.freq) return null;
    return value;
  } catch {
    return null;
  }
}

export function recurrenceSummary(raw: string | null | undefined): string {
  const rule = parseRecurrence(raw);
  if (!rule) return '';
  const interval = rule.interval || 1;
  let text = '';
  if (rule.freq === 'weekly' && rule.weekdays?.length) {
    const days = [...rule.weekdays].sort((a, b) => a - b).map((d) => weekNames[d] ?? '').join('、');
    text = interval > 1 ? `每 ${interval} 周的${days}` : `每${days}`;
  } else {
    const unit = { daily: '天', weekly: '周', monthly: '月', yearly: '年' }[rule.freq];
    text = interval > 1 ? `每 ${interval} ${unit}` : { daily: '每天', weekly: '每周', monthly: '每月', yearly: '每年' }[rule.freq];
  }
  if (rule.anchor === 'completion') text += '，完成后计算';
  if (rule.until) text += `，至 ${rule.until}`;
  return text;
}

export function dueLabel(task: Task, today: string): { text: string; tone: 'muted' | 'today' | 'overdue' } {
  if (!task.dueDate) return { text: '无日期', tone: 'muted' };
  const time = task.dueTime ? ` ${task.dueTime}` : '';
  if (task.dueDate < today) return { text: `${task.dueDate.slice(5)}${time}`, tone: 'overdue' };
  if (task.dueDate === today) return { text: `今天${time}`, tone: 'today' };
  return { text: `${task.dueDate.slice(5)}${time}`, tone: 'muted' };
}

export function daysFromToday(date: string, today: string): number {
  const a = new Date(`${date}T00:00:00`);
  const b = new Date(`${today}T00:00:00`);
  return Math.round((a.getTime() - b.getTime()) / 86400000);
}

export function greetingFor(name: string): string {
  const hour = new Date().getHours();
  const prefix = hour < 6 ? '夜深了' : hour < 11 ? '早上好' : hour < 14 ? '中午好' : hour < 18 ? '下午好' : '晚上好';
  return `${prefix}，${name}`;
}
