import { api } from '@/api/client';

export function avatarUrl(key: string | null | undefined): string | undefined {
  return key ? `/api/v1/avatars/${key}` : undefined;
}

export async function uploadAvatar(file: File, memberId?: number): Promise<string> {
  const form = new FormData();
  form.append('file', file);
  const path = memberId ? `/family/members/${memberId}/avatar` : '/me/avatar';
  const result = await api.upload<{ avatarKey: string }>(path, form);
  return result.avatarKey;
}
