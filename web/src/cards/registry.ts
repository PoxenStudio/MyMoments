import type { Component } from 'vue';
import type { IconName } from '@/components/base/icons';
import OverviewCard from './OverviewCard.vue';
import TasksCard from './TasksCard.vue';
import CalendarCard from './CalendarCard.vue';
import TimelineCard from './TimelineCard.vue';
import BirthdaysCard from './BirthdaysCard.vue';
import RemindersCard from './RemindersCard.vue';
import MessagesCard from './MessagesCard.vue';
import NotesCard from './NotesCard.vue';
import WeatherCard from './WeatherCard.vue';
import MembersCard from './MembersCard.vue';
import ContactsCard from './ContactsCard.vue';

export interface CardDef {
  id: string;
  title: string;
  component: Component;
  icon: IconName;
  description: string;
  defaultVisible: boolean;
  supportsResize: boolean;
  path?: string;
}

export const cardDefs: CardDef[] = [
  { id: 'overview', title: '今日概览', component: OverviewCard, icon: 'sun', description: '待办、日程与生日的一眼概览', defaultVisible: true, supportsResize: true, path: '/tasks' },
  { id: 'tasks', title: '待办', component: TasksCard, icon: 'tasks', description: '最近需要完成的事', defaultVisible: true, supportsResize: true, path: '/tasks' },
  { id: 'calendar', title: '行事历', component: CalendarCard, icon: 'calendar', description: '未来七天的日程安排', defaultVisible: true, supportsResize: true, path: '/calendar' },
  { id: 'birthdays', title: '生日', component: BirthdaysCard, icon: 'gift', description: '家人与朋友的生日提醒', defaultVisible: true, supportsResize: true, path: '/calendar' },
  { id: 'reminders', title: '我的提醒', component: RemindersCard, icon: 'bell', description: '待处理的提醒', defaultVisible: true, supportsResize: true, path: '/tasks' },
  { id: 'timeline', title: '家庭时光轴', component: TimelineCard, icon: 'timeline', description: '最近的家庭动态', defaultVisible: false, supportsResize: true, path: '/timeline' },
  { id: 'messages', title: '留言', component: MessagesCard, icon: 'messages', description: '留言板上的最新消息', defaultVisible: false, supportsResize: true, path: '/messages' },
  { id: 'notes', title: '便签', component: NotesCard, icon: 'notes', description: '随手记下的小事', defaultVisible: false, supportsResize: true, path: '/notes' },
  { id: 'weather', title: '天气', component: WeatherCard, icon: 'cloud', description: '当前位置的天气与预报', defaultVisible: false, supportsResize: true },
  { id: 'members', title: '家庭成员', component: MembersCard, icon: 'family', description: '家庭里的每个人', defaultVisible: false, supportsResize: true, path: '/family' },
  { id: 'contacts', title: '通讯录', component: ContactsCard, icon: 'user', description: '常联系的亲友', defaultVisible: false, supportsResize: true, path: '/family' },
];

export const cardRegistry: Record<string, CardDef> = Object.fromEntries(
  cardDefs.map((def) => [def.id, def]),
);
