<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import AppAvatar from '@/components/base/AppAvatar.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import { avatarUrl } from '@/utils/avatar';
import type { Contact, MentionTarget, User } from '@/types';

const props = withDefaults(defineProps<{
  text: string;
  mentions?: MentionTarget[];
  contacts?: Contact[];
  members?: User[];
}>(), {
  mentions: () => [],
  contacts: () => [],
  members: () => [],
});

const router = useRouter();
const openKey = ref<string | null>(null);

interface Segment {
  type: 'text' | 'mention';
  text: string;
  target?: MentionTarget;
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

const segments = computed<Segment[]>(() => {
  const named = props.mentions.filter((m) => m.name);
  if (!named.length) return [{ type: 'text', text: props.text }];
  const sorted = [...named].sort((a, b) => (b.name?.length ?? 0) - (a.name?.length ?? 0));
  const names = sorted.map((m) => escapeRegExp(m.name!));
  const regex = new RegExp(`@(${names.join('|')})`, 'g');
  const out: Segment[] = [];
  let last = 0;
  let match: RegExpExecArray | null;
  while ((match = regex.exec(props.text)) !== null) {
    if (match.index > last) out.push({ type: 'text', text: props.text.slice(last, match.index) });
    const name = match[1];
    const target = sorted.find((m) => m.name === name);
    out.push({ type: 'mention', text: `@${name}`, target });
    last = match.index + match[0].length;
  }
  if (last < props.text.length) out.push({ type: 'text', text: props.text.slice(last) });
  return out;
});

function contactFor(target: MentionTarget): Contact | undefined {
  if (target.type !== 'contact') return undefined;
  return props.contacts.find((c) => c.id === target.id);
}

function memberFor(target: MentionTarget): User | undefined {
  if (target.type !== 'user') return undefined;
  return props.members.find((m) => m.id === target.id);
}

function birthdayLabel(c: Contact): string {
  if (c.birthdayMonth == null || c.birthdayDay == null) return '';
  const base = `${c.birthdayMonth}月${c.birthdayDay}日`;
  return c.birthYear ? `${c.birthYear}年${base}` : base;
}

function pendingLabel(c: Contact): string {
  const count = c.reminders.filter((r) => r.enabled).length;
  return count ? `${count} 项问候/待办` : '';
}

function keyOf(target: MentionTarget): string {
  return `${target.type}-${target.id}`;
}

function toggle(target: MentionTarget): void {
  openKey.value = openKey.value === keyOf(target) ? null : keyOf(target);
}

function goDetail(target: MentionTarget): void {
  if (target.type === 'contact') {
    void router.push({ name: 'contact-detail', params: { id: String(target.id) } });
  }
}

function onDocumentClick(event: MouseEvent): void {
  const target = event.target as HTMLElement;
  if (!target.closest('.mention-chip')) openKey.value = null;
}

onMounted(() => document.addEventListener('click', onDocumentClick));
onBeforeUnmount(() => document.removeEventListener('click', onDocumentClick));
</script>

<template>
  <span class="mention-renderer">
    <template v-for="(seg, index) in segments" :key="index">
      <template v-if="seg.type === 'text'">{{ seg.text }}</template>
      <span v-else-if="seg.target" class="mention-chip-wrap">
        <button type="button" class="mention-chip" @click.stop="toggle(seg.target)">
          <AppAvatar :name="seg.target.name ?? ''" :size="16" :src="avatarUrl(seg.target.avatarKey)" />
          <span class="mention-chip__name">{{ seg.text }}</span>
        </button>
        <div
          v-if="openKey === keyOf(seg.target)"
          class="mention-card"
          @click.stop
        >
          <div class="mention-card__head">
            <AppAvatar
              :name="seg.target.name ?? ''"
              :size="40"
              :src="avatarUrl(contactFor(seg.target)?.avatarKey ?? memberFor(seg.target)?.avatarKey ?? seg.target.avatarKey)"
            />
            <div class="mention-card__who">
              <strong>{{ seg.target.name }}</strong>
              <span class="mention-card__type">{{ seg.target.type === 'contact' ? '联系人' : '家庭成员' }}</span>
            </div>
            <button
              v-if="seg.target.type === 'contact'"
              type="button"
              class="mention-card__go"
              :aria-label="`查看 ${seg.target.name} 详情`"
              @click="goDetail(seg.target)"
            >
              <AppIcon name="chevron-right" :size="18" />
            </button>
          </div>
          <template v-if="seg.target.type === 'contact' && contactFor(seg.target)">
            <p v-if="birthdayLabel(contactFor(seg.target)!)" class="mention-card__row">
              <AppIcon name="calendar" :size="14" />
              {{ birthdayLabel(contactFor(seg.target)!) }}
            </p>
            <p v-if="pendingLabel(contactFor(seg.target)!)" class="mention-card__row">
              <AppIcon name="bell" :size="14" />
              {{ pendingLabel(contactFor(seg.target)!) }}
            </p>
            <p v-if="!birthdayLabel(contactFor(seg.target)!) && !pendingLabel(contactFor(seg.target)!)" class="mention-card__empty">
              暂无更多信息
            </p>
          </template>
          <template v-else-if="seg.target.type === 'user' && memberFor(seg.target)">
            <p v-if="memberFor(seg.target)!.relation" class="mention-card__row">
              <AppIcon name="user" :size="14" />
              {{ memberFor(seg.target)!.relation }}
            </p>
          </template>
        </div>
      </span>
    </template>
  </span>
</template>

<style scoped>
.mention-renderer {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.mention-chip-wrap {
  position: relative;
  display: inline;
}

.mention-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  vertical-align: middle;
  padding: 1px 8px 1px 3px;
  margin: 0 1px;
  border-radius: var(--at-radius-pill);
  background: color-mix(in srgb, var(--at-color-primary) 12%, transparent);
  color: var(--at-color-primary);
  font-size: 0.92em;
  font-weight: 500;
  line-height: 1.5;
  cursor: pointer;
}

.mention-chip:hover {
  background: color-mix(in srgb, var(--at-color-primary) 20%, transparent);
}

.mention-chip__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 160px;
}

.mention-card {
  position: absolute;
  left: 0;
  top: calc(100% + 6px);
  z-index: var(--at-z-sticky);
  width: 220px;
  padding: var(--at-space-3);
  background: var(--at-bg-surface);
  border: 1px solid var(--at-color-hairline);
  border-radius: var(--at-radius-lg);
  box-shadow: var(--at-shadow-dialog);
}

.mention-card__head {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
}

.mention-card__who {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.mention-card__type {
  font-size: 11px;
  color: var(--at-color-muted);
}

.mention-card__go {
  flex: none;
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border-radius: var(--at-radius-md);
  color: var(--at-color-muted);
}

.mention-card__go:hover {
  background: var(--at-bg-surface-strong);
  color: var(--at-text-primary);
}

.mention-card__row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: var(--at-space-2);
  font-size: var(--at-text-micro);
  color: var(--at-text-secondary);
}

.mention-card__empty {
  margin-top: var(--at-space-2);
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}
</style>
