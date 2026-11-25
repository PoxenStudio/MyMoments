import type { MentionTarget } from '@/types';

export interface MentionRef {
  type: MentionTarget['type'];
  id: number;
}

/**
 * 提交给后端前只保留 type/id：后端按 DisallowUnknownFields 解析请求体，
 * 且姓名/头像由服务端根据 type/id 解析，不需要前端回传。
 */
export function toMentionRefs(mentions?: MentionTarget[] | null): MentionRef[] {
  return (mentions ?? []).map((m) => ({ type: m.type, id: m.id }));
}
