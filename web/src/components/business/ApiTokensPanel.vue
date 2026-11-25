<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { ApiError, api } from '@/api/client';
import { useToastStore } from '@/stores/toast';
import AppButton from '@/components/base/AppButton.vue';
import AppConfirmDialog from '@/components/base/AppConfirmDialog.vue';
import AppDialog from '@/components/base/AppDialog.vue';
import AppEmptyState from '@/components/base/AppEmptyState.vue';
import AppField from '@/components/base/AppField.vue';
import AppIcon from '@/components/base/AppIcon.vue';
import AppInput from '@/components/base/AppInput.vue';
import AppPanel from '@/components/base/AppPanel.vue';
import AppSelect from '@/components/base/AppSelect.vue';
import type { ApiToken } from '@/types';

const toast = useToastStore();

const tokens = ref<ApiToken[]>([]);
const loading = ref(false);
const loadError = ref('');

const createOpen = ref(false);
const creating = ref(false);
const createError = ref('');
const name = ref('');
const expiry = ref('90');
const scopes = reactive<Record<string, string[]>>({});
const plaintext = ref('');
const plaintextOpen = ref(false);
const revokeTarget = ref<ApiToken | null>(null);

const modules = [
  { id: 'tasks', label: '待办', actions: ['read', 'write', 'delete'] },
  { id: 'calendar', label: '行事历', actions: ['read', 'write', 'delete'] },
  { id: 'birthdays', label: '生日', actions: ['read', 'write', 'delete'] },
  { id: 'contacts', label: '通讯录', actions: ['read', 'write', 'delete'] },
  { id: 'messages', label: '留言', actions: ['read', 'write', 'delete'] },
  { id: 'notes', label: '便签', actions: ['read', 'write', 'delete'] },
  { id: 'family', label: '家庭成员', actions: ['read'] },
  { id: 'timeline', label: '时光轴', actions: ['read'] },
  { id: 'reminders', label: '提醒', actions: ['read'] },
  { id: 'weather', label: '天气', actions: ['read'] },
];

const actionLabels: Record<string, string> = { read: '读取', write: '编辑', delete: '删除' };

const expiryOptions = [
  { value: '30', label: '30 天' },
  { value: '90', label: '90 天' },
  { value: '180', label: '180 天' },
  { value: '365', label: '365 天' },
  { value: '0', label: '永不过期' },
];

const moduleLabels = Object.fromEntries(modules.map((m) => [m.id, m.label]));

const scopeSummary = computed(() => {
  return Object.entries(scopes)
    .filter(([, actions]) => actions.length)
    .map(([id, actions]) => `${moduleLabels[id] ?? id}：${actions.map((a) => actionLabels[a] ?? a).join('/')}`);
});

function hasScope(module: string, action: string): boolean {
  return scopes[module]?.includes(action) ?? false;
}

function toggleScope(module: string, action: string): void {
  const current = scopes[module] ?? [];
  scopes[module] = current.includes(action) ? current.filter((a) => a !== action) : [...current, action];
}

function applyPreset(kind: 'read' | 'write'): void {
  for (const module of modules) {
    scopes[module.id] = kind === 'read' ? ['read'] : ['read', 'write'];
  }
}

function resetForm(): void {
  name.value = '';
  expiry.value = '90';
  createError.value = '';
  for (const module of modules) {
    scopes[module.id] = ['read'];
  }
}

function openCreate(): void {
  resetForm();
  createOpen.value = true;
}

async function load(): Promise<void> {
  loading.value = true;
  loadError.value = '';
  try {
    const res = await api.get<{ items: ApiToken[] }>('/tokens');
    tokens.value = res.items;
  } catch (err) {
    loadError.value = err instanceof ApiError ? err.message : '读取 Token 失败';
  } finally {
    loading.value = false;
  }
}

async function createToken(): Promise<void> {
  if (!name.value.trim()) {
    createError.value = '请填写 Token 名称';
    return;
  }
  const payload: Record<string, string[]> = {};
  for (const [module, actions] of Object.entries(scopes)) {
    if (actions.length) payload[module] = actions;
  }
  if (!Object.keys(payload).length) {
    createError.value = '至少选择一个权限';
    return;
  }
  creating.value = true;
  createError.value = '';
  try {
    const res = await api.post<{ token: string; item: ApiToken }>('/tokens', {
      name: name.value.trim(),
      scopes: payload,
      expiresInDays: Number(expiry.value),
    });
    tokens.value = [res.item, ...tokens.value];
    createOpen.value = false;
    plaintext.value = res.token;
    plaintextOpen.value = true;
  } catch (err) {
    createError.value = err instanceof ApiError ? err.message : '创建 Token 失败';
  } finally {
    creating.value = false;
  }
}

async function revoke(): Promise<void> {
  const target = revokeTarget.value;
  revokeTarget.value = null;
  if (!target) return;
  try {
    await api.del(`/tokens/${target.id}`);
    tokens.value = tokens.value.filter((t) => t.id !== target.id);
    toast.success('Token 已撤销');
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '撤销失败');
  }
}

async function copyToken(): Promise<void> {
  try {
    await navigator.clipboard.writeText(plaintext.value);
    toast.success('已复制到剪贴板');
  } catch {
    toast.error('复制失败，请手动选择文本');
  }
}

function formatTime(ts: number | null): string {
  if (!ts) return '—';
  return new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date(ts * 1000));
}

function scopeText(token: ApiToken): string {
  return Object.entries(token.scopes)
    .map(([id, actions]) => `${moduleLabels[id] ?? id}(${actions.join('/')})`)
    .join('、');
}

onMounted(() => {
  void load();
});
</script>

<template>
  <AppPanel title="AI 与 API" subtitle="用 API Token 让 AI Agent 读取或整理家庭信息，可随时撤销。">
    <div class="api">
      <div class="api__intro">
        <p class="api__desc">
          Token 只代表你本人的权限，并受可见范围限制。创建时请只授予需要的模块与操作；删除操作默认不开启。
        </p>
        <div class="api__links">
          <a href="/skill.md" download class="api__link">下载 AI Skill<AppIcon name="arrow-right" :size="14" /></a>
        </div>
      </div>

      <div v-if="loading" class="api__state">正在读取 Token…</div>
      <p v-else-if="loadError" class="api__error">{{ loadError }}</p>
      <template v-else>
        <ul v-if="tokens.length" class="api__list">
          <li v-for="token in tokens" :key="token.id" class="api__row">
            <div class="api__row-main">
              <div class="api__row-head">
                <span class="api__name">{{ token.name }}</span>
                <code class="api__prefix">{{ token.prefix }}…</code>
              </div>
              <p class="api__scopes">{{ scopeText(token) }}</p>
              <p class="api__meta">
                创建 {{ formatTime(token.createdAt) }} · 到期 {{ formatTime(token.expiresAt) }} · 最近使用 {{ formatTime(token.lastUsedAt) }}
              </p>
            </div>
            <AppButton size="sm" variant="ghost" @click="revokeTarget = token">撤销</AppButton>
          </li>
        </ul>
        <AppEmptyState
          v-else
          icon="key"
          title="还没有 API Token"
          description="为 ChatGPT、Claude 等 AI Agent 创建一个专用 Token。"
        />
        <div class="api__actions">
          <AppButton size="sm" variant="outline" @click="openCreate"><AppIcon name="plus" :size="15" />创建 Token</AppButton>
        </div>
      </template>
    </div>

    <AppDialog v-model="createOpen" title="创建 API Token" size="md">
      <div class="api-form">
        <AppField label="名称" for-id="token-name" hint="例如：Claude 家庭助理">
          <AppInput id="token-name" v-model="name" placeholder="给这个 Token 起个名字" />
        </AppField>
        <AppField label="有效期" for-id="token-expiry">
          <AppSelect id="token-expiry" v-model="expiry" :options="expiryOptions" />
        </AppField>

        <div class="api-form__presets">
          <span>快速选择</span>
          <AppButton size="sm" variant="ghost" @click="applyPreset('read')">全部只读</AppButton>
          <AppButton size="sm" variant="ghost" @click="applyPreset('write')">全部读写</AppButton>
        </div>

        <ul class="api-form__modules">
          <li v-for="module in modules" :key="module.id" class="api-form__module">
            <span class="api-form__module-name">{{ module.label }}</span>
            <div class="api-form__actions">
              <button
                v-for="action in module.actions"
                :key="action"
                type="button"
                class="api-form__chip"
                :class="{ 'is-active': hasScope(module.id, action), 'is-danger': action === 'delete' && hasScope(module.id, action) }"
                :aria-pressed="hasScope(module.id, action)"
                @click="toggleScope(module.id, action)"
              >
                {{ actionLabels[action] }}
              </button>
            </div>
          </li>
        </ul>

        <p class="api-form__summary">已选择：{{ scopeSummary.length ? scopeSummary.join('；') : '无' }}</p>
        <p v-if="createError" class="api__error">{{ createError }}</p>
      </div>
      <template #footer>
        <AppButton variant="ghost" @click="createOpen = false">取消</AppButton>
        <AppButton :loading="creating" @click="createToken">创建</AppButton>
      </template>
    </AppDialog>

    <AppDialog v-model="plaintextOpen" title="Token 已创建" size="sm">
      <div class="api-secret">
        <p class="api-secret__warn">这串 Token 只会显示一次，请立即复制并妥善保存。</p>
        <code class="api-secret__value">{{ plaintext }}</code>
        <AppButton variant="outline" size="sm" @click="copyToken"><AppIcon name="copy" :size="15" />复制</AppButton>
      </div>
      <template #footer>
        <AppButton @click="plaintextOpen = false">我已保存</AppButton>
      </template>
    </AppDialog>

    <AppConfirmDialog
      :model-value="!!revokeTarget"
      title="撤销 Token"
      :message="`撤销后，使用「${revokeTarget?.name ?? ''}」的应用会立即失去访问权限。`"
      confirm-text="撤销"
      @update:model-value="!$event && (revokeTarget = null)"
      @confirm="revoke"
    />
  </AppPanel>
</template>

<style scoped>
.api {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.api__desc {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
  line-height: 1.65;
}

.api__links {
  display: flex;
  flex-wrap: wrap;
  gap: var(--at-space-4);
  margin-top: var(--at-space-2);
}

.api__link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: var(--at-text-caption);
  color: var(--at-color-body-strong);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.api__state,
.api__error {
  font-size: var(--at-text-caption);
  color: var(--at-color-muted);
}

.api__error {
  color: var(--at-color-danger);
}

.api__list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.api__row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--at-space-3);
  padding: var(--at-space-3) 0;
  border-top: 1px solid var(--at-color-hairline);
}

.api__row-head {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
}

.api__name {
  font-size: var(--at-text-body-sm);
  color: var(--at-color-body-strong);
}

.api__prefix {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
  background: var(--at-bg-surface-strong);
  padding: 1px 6px;
  border-radius: var(--at-radius-pill);
}

.api__scopes {
  margin-top: 4px;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
  line-height: 1.5;
}

.api__meta {
  margin-top: 2px;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted-soft);
}

.api__actions {
  display: flex;
  justify-content: flex-end;
}

.api-form {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-4);
}

.api-form__presets {
  display: flex;
  align-items: center;
  gap: var(--at-space-2);
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
}

.api-form__modules {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
  border-top: 1px solid var(--at-color-hairline);
}

.api-form__module {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--at-space-3);
  min-height: 42px;
  border-bottom: 1px solid var(--at-color-hairline);
}

.api-form__module-name {
  font-size: var(--at-text-caption);
  color: var(--at-color-body-strong);
}

.api-form__actions {
  display: flex;
  gap: 4px;
}

.api-form__chip {
  padding: 4px 10px;
  border: 1px solid var(--at-color-hairline-strong);
  border-radius: var(--at-radius-pill);
  background: transparent;
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
  transition: background var(--at-duration-fast) var(--at-ease), color var(--at-duration-fast) var(--at-ease);
}

.api-form__chip.is-active {
  background: var(--at-color-ink);
  border-color: var(--at-color-ink);
  color: var(--at-bg-surface);
}

.api-form__chip.is-danger.is-active {
  background: var(--at-color-danger);
  border-color: var(--at-color-danger);
}

.api-form__summary {
  font-size: var(--at-text-micro);
  color: var(--at-color-muted);
  line-height: 1.6;
}

.api-secret {
  display: flex;
  flex-direction: column;
  gap: var(--at-space-3);
  align-items: flex-start;
}

.api-secret__warn {
  font-size: var(--at-text-caption);
  color: var(--at-color-danger);
  line-height: 1.6;
}

.api-secret__value {
  display: block;
  width: 100%;
  padding: var(--at-space-3);
  border-radius: var(--at-radius-md);
  background: var(--at-bg-surface-strong);
  font-size: 13px;
  word-break: break-all;
  user-select: all;
}
</style>
