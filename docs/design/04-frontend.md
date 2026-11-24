# 04 · 前端架构

## 1. 工程结构

```text
web/
├─ index.html
├─ vite.config.ts          # 含 vite-plugin-pwa
├─ src/
│  ├─ main.ts
│  ├─ router/              # 路由 + deep-link 处理
│  ├─ api/                 # 请求封装、错误归一化、幂等键
│  ├─ stores/              # Pinia
│  ├─ composables/         # usePagination / useDialog / useUndo / useMedia
│  ├─ components/
│  │  ├─ base/             # App* 基础组件
│  │  └─ business/         # 业务组件（卡片、列表、表单）
│  ├─ cards/               # 首页卡片注册表（见 §5）
│  ├─ views/               # 页面级组件
│  ├─ styles/
│  │  ├─ tokens.css        # 设计令牌（唯一视觉来源）
│  │  └─ base.css          # reset + 全局排版
│  └─ utils/
│     ├─ chrono.ts         # date-only / datetime / tz 统一封装
│     └─ id.ts             # 幂等键、client id
└─ public/
   ├─ manifest.webmanifest
   ├─ icons/               # 192/512/maskable/apple-touch
   └─ offline.html
```

依赖单向：`views → business → base`；`api` 与 `stores` 不依赖视图。基础组件不依赖 Pinia、不读路由、不发请求（CodingStandards §16）。

## 2. 路由与 deep-link

- 一级路由：`/`（首页）、`/calendar`、`/tasks`、`/timeline`、`/family`、`/messages`、`/notes`、`/settings`。
- 详情 deep-link 统一契约 `?open=<对象id>&date=YYYY-MM-DD`（PRD §5.3）：路由守卫解析后打开对应详情组件并定位到发生日期，返回时还原原日期/筛选/滚动上下文。
- 页面级组件 `views/` 懒加载；`scrollBehavior` 保存/还原列表位置。

## 3. 状态管理

| Store | 职责 |
|---|---|
| `useAuthStore` | 当前用户、会话状态、退出 |
| `useDashboardStore` | 卡片布局（个人 + 家庭默认）、卡片摘要缓存 |
| `useFamilyStore` | 成员列表（含停用成员展示名） |
| `usePreferenceStore` | 语言、日期格式、天气位置 |

业务数据以列表页本地分页 + 服务端游标为主，不全局缓存可写实体；跨页面一致性靠「操作后局部失效 + 重新拉取」而非全局镜像（PRD §19.2.6）。

## 4. 基础组件库（App*）

沿用 CodingStandards §16 的组件纪律，视觉令牌用 DESIGN.md。核心组件：

| 组件 | 用途 | 关键规则 |
|---|---|---|
| `AppPage` | 页面骨架 | 标题/描述/操作区；内容区 24px 边距 |
| `AppPanel` | 卡片/面板 | header/body/footer；空态用 `AppEmptyState` |
| `AppDialog` | 对话框 | 移动端全宽；破坏性用 `variant=danger` |
| `AppConfirmDialog` | 二次确认 | 删除/停用必用；按钮用动词文案 |
| `AppTable` | 数据表 | 服务端分页；数字右对齐等宽 |
| `AppForm` / `AppFormField` | 表单 | 常用字段前置，低频折叠「更多设置」 |
| `AppDrawer` | 侧滑详情 | 替代嵌套 Dialog |
| `AppToast` | 轻提示 | 成功 3s/错误 6s；合并同消息 |
| `AppStatusTag` | 状态标签 | 只传语义不传颜色 |
| `AppEmptyState` / `AppErrorState` | 空/错态 | 区分呈现；错误不泄露堆栈 |
| `AppUndoToast` | 统一撤销 | 见 PRD §5.3（5–10s 有效，离开失效） |

### 表单字段折叠（PRD §16.3）

标题、日期、负责人等常用字段前置；重复规则、提醒、可见范围、颜色/图标折叠到「更多设置」，一旦设置后不再折叠。

## 5. 首页卡片系统

卡片 = 功能入口 + 数据摘要，不是独立业务数据（PRD §1）。

```ts
// cards/registry.ts
export interface CardDef {
  id: string
  title: string
  component: Component
  defaultVisible: boolean
  supportsResize: boolean
}
export const cardRegistry: Record<string, CardDef> = { /* task, calendar, timeline, birthday, reminder, message, note, weather */ }
```

- **布局数据**：`user_preferences.dashboard`（个人，NULL=跟随家庭默认）+ `household_settings.default_dashboard`（管理员发布）。
- **摘要拉取**：`GET /api/v1/dashboard` 一次返回布局 + 各可见卡片的摘要；服务端按卡片分块计算，单卡片失败返回该卡片的 `error` 状态而非整页失败（PRD §18）。
- **默认可见性**：今日概览固定；待办/行事历/生日/我的提醒默认开；家庭时光轴/留言/便签/天气默认关（PRD §6.2）。
- **无障碍**：排序支持键盘上移/下移（不依赖拖拽）；触控目标 ≥44px。

## 6. 设计令牌

以 DESIGN.md 为品牌来源，用 `--at-` 前缀承载（CodingStandards §15 命名），数值替换为暖黑编辑部品牌。`tokens.css` 为唯一颜色/间距来源。

```css
:root {
  /* 颜色（来自 DESIGN.md） */
  --at-color-primary: #292524;          /* 暖黑墨色，主操作 */
  --at-color-primary-active: #0c0a09;
  --at-color-ink: #0c0a09;
  --at-color-body: #4e4e4e;
  --at-color-muted: #777169;
  --at-color-muted-soft: #a8a29e;
  --at-color-hairline: #e7e5e4;
  --at-color-hairline-strong: #d6d3d1;
  --at-bg-page: #f5f5f5;                /* canvas */
  --at-bg-surface: #ffffff;             /* surface-card */
  --at-bg-elevated: #ffffff;
  --at-text-primary: #0c0a09;
  --at-text-secondary: #4e4e4e;
  --at-color-success: #16a34a;
  --at-color-danger: #dc2626;
  --at-border: #e7e5e4;

  /* 圆角（DESIGN.md rounded） */
  --at-radius-xs: 4px; --at-radius-sm: 6px; --at-radius-md: 8px;
  --at-radius-lg: 12px; --at-radius-xl: 16px; --at-radius-xxl: 24px;
  --at-radius-pill: 9999px;

  /* 间距（DESIGN.md spacing，4 基准） */
  --at-space-1: 4px; --at-space-2: 8px; --at-space-3: 12px; --at-space-4: 16px;
  --at-space-5: 20px; --at-space-6: 24px; --at-space-8: 32px;

  /* 阴影 */
  --at-shadow-panel: 0 1px 2px rgba(0,0,0,.04);
  --at-shadow-dialog: 0 12px 32px rgba(0,0,0,.12);

  /* 字体 */
  --at-font-display: 'EB Garamond', 'Times New Roman', serif;  /* Waldenburg 替代 */
  --at-font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
}
```

**品牌要点（DESIGN.md Do/Don't）**：display 用衬线字重 300（替代 EB Garamond，不商用 Waldenburg）；主 CTA 为墨色胶囊，**不引入高饱和主色**；渐变 orb 仅作氛围装饰，不作按钮/文字/组件底色；display 永不加粗。

**品牌色红线（硬性）**：主色固定为 DESIGN.md 暖黑墨色 `--at-color-primary: #292524`（按下态 `#0c0a09`），画布 `#f5f5f5`；**禁止引入高饱和品牌色、蓝色 `#2563EB` 或任何第三方组件库默认主色**；语义色仅保留 success `#16a34a` / danger `#dc2626` 两枚，不扩展。此红线覆盖 CodingStandards §15.1 的蓝色表，后者在 MyMoments 中不生效。

**断点**：`<640px` / `640–1024px` / `>=1024px`（DESIGN.md），与 PRD §16.3 一致（不沿用 CodingStandards 的 Vuetify 600/960/1280）。

## 7. 错误态分层（PRD §5.3）

`AppErrorState` 按原因呈现三类：

1. 网络失败 → 「网络连接失败」+ 重试。
2. 会话过期（`auth.session_expired`）→ 引导重新登录。
3. 服务端错误（`server.unavailable`/5xx）→ 「服务暂时不可用」+ 稍后再试。

不得把以上任一类显示为「暂无数据」（空态）。

## 8. 日期工具（`utils/chrono.ts`）

统一封装三类时间语义（date-only / local datetime / timestamp）的解析、格式化与家庭时区换算，供首页、日历、时光轴、提醒复用，避免各页面各自实现一套日期计算（PRD §17.1）。
