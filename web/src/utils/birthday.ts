import type { RouteLocationRaw } from 'vue-router';
import type { Birthday } from '@/types';

/**
 * 生日条目的跳转目标：联系人 → 联系人详情，家庭成员 → 家庭页，其余 → 生日编辑。
 */
export function birthdayRoute(birthday: Birthday): RouteLocationRaw {
  if (birthday.contactId) {
    return { name: 'contact-detail', params: { id: String(birthday.contactId) } };
  }
  if (birthday.userId) {
    return { path: '/family', query: { tab: 'members' } };
  }
  return { path: '/family', query: { tab: 'birthdays', open: String(birthday.id) } };
}
