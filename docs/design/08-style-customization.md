# 08 · 样式自定义方案

> 状态：已实现（Implemented）
> 范围：首页页面色调、卡片圆角、页面背景图/背景色、卡片角图、头像与昵称、卡片显示、三层级样式继承与快速恢复默认。

## 1. 目标

本项目对 UI 样式自定义要求高。目标是在不破坏现有「暖黑墨色编辑部品牌」的前提下，让每个成员都能把界面调成自己舒服的样子，同时管理员能为全家设定一个统一的默认样式。

核心诉求（来自需求）：

1. 右上角图标作为「样式快速设置」入口：调页面色调、卡片圆角，结果存到用户数据，**任何位置优先使用用户自定义设置**。
2. 系统设置中新增「样式设置」：默认色调 + 指定背景色；预设多张不同色调背景图备用；卡片背景（右下角角图，非整卡）；花纹与卡通形象样图；背景图虚化程度 + 预览。
3. 个人设置：头像、昵称、显示的卡片（管理员可在家庭管理中为用户设置）、自定义样式、一个「默认使用系统配置」的开关。
4. 可快速恢复默认，提供「使用默认配置」开关。

## 2. 设计原则

1. **三层级继承，越近越优先**：`个人自定义 > 家庭默认（管理员）> 系统内置默认`。任何界面读取样式都走同一解析函数，天然满足「任何位置优先使用用户自定义设置」。
2. **样式是「氛围」不是「品牌」**：色调只改画布、卡片、描边、氛围装饰，**不改变品牌墨色主 CTA（`--at-color-primary: #292524`）与语义色**。沿用 04-frontend.md §6 的品牌红线。
3. **所见即所得 + 即时预览**：所有改动先应用到当前页面，再持久化；背景图虚化、卡片角图都有预览。
4. **默认即安全**：新用户零配置直接得到系统/家庭默认；一键恢复默认；「使用默认配置」开关默认开启。
5. **预设优先，不开放自由上传**：色调、背景图、角图均为前端内置预设资产，通过 key 引用，避免引入额外的图片上传与治理成本（头像除外，头像为唯一允许用户上传的图片）。

## 3. 样式模型（ThemeConfig）

一个统一的 JSON 结构，三层级复用。字段均为可选，缺失字段向下层合并（partial override）。

```ts
// web/src/types.ts 新增
export interface ThemeConfig {
  useDefault?: boolean;          // 仅个人层使用：true=跟随家庭/系统默认
  tone: ThemeTone;               // 色调预设 key
  cardRadius: CardRadius;        // 卡片圆角档位
  pageBackground: {
    kind: 'color' | 'image';     // 背景类型
    color?: string;              // kind=color 时：指定背景色
    image?: string;              // kind=image 时：预设背景图 key
    blur?: number;               // 虚化 0–20（px）
  };
  cardCorner: {
    image: string | null;        // 预设角图 key（花纹/卡通）；null=无
    opacity: number;             // 0.05–0.25
  };
}

export type CardRadius = 'sm' | 'md' | 'lg' | 'xl';
export type ThemeTone =
  | 'default'
  | 'warm-sun' | 'warm-sand' | 'warm-peach' | 'warm-cream' | 'warm-rose'
  | 'cartoon-macaron' | 'cartoon-sky' | 'cartoon-mint' | 'cartoon-berry' | 'cartoon-lemon';
```

### 3.1 色调预设（暖色 + 卡通）

内置 11 个预设，分两组。每个预设定义一组 CSS 变量覆盖（见 §6），只改氛围，不改品牌墨色。

| key | 名称 | 定位 | 画布基调 |
|---|---|---|---|
| `default` | 默认 | 现有编辑部白 | `#f5f5f5` |
| `warm-sun` | 暖阳 | 暖色 | 浅杏 `#f7efe2` |
| `warm-sand` | 沙色 | 暖色 | 暖灰褐 `#f1e8db` |
| `warm-peach` | 蜜桃 | 暖色 | 粉杏 `#f9ece2` |
| `warm-cream` | 奶咖 | 暖色 | 奶油咖啡 `#f5ece0` |
| `warm-rose` | 暖粉 | 暖色 | 柔粉 `#f9e9ec` |
| `cartoon-macaron` | 马卡龙 | 卡通 | 柔粉薄荷混搭 `#f6eef2` |
| `cartoon-sky` | 天空蓝 | 卡通 | 淡蓝 `#e9f1f8` |
| `cartoon-mint` | 薄荷绿 | 卡通 | 淡绿 `#eaf4ef` |
| `cartoon-berry` | 浆果粉 | 卡通 | 莓粉 `#f7e6ec` |
| `cartoon-lemon` | 柠檬黄 | 卡通 | 淡黄 `#f8f2dd` |

每个预设还附带 1–2 枚「氛围球」渐变装饰色，复用现有 `--at-gradient-*` 的思路，仅用于顶部氛围，不作用于文字/按钮。

### 3.2 卡片角图（右下角）

角图是**右下角小尺寸装饰图**，不是整卡背景。渲染为绝对定位、`pointer-events: none`、低透明度，叠在卡片内容之上但视觉弱于内容。内置两类：

- 花纹（pattern）：藤蔓、叶片、波纹、星星点。
- 卡通形象（character）：小熊、兔子、小猫、云朵、太阳。

角图 key 常量：`corner-vine` / `corner-leaf` / `corner-wave` / `corner-dot` / `corner-bear` / `corner-rabbit` / `corner-cat` / `corner-cloud` / `corner-sun`。默认 `null`（无角图）。

### 3.3 背景图 + 虚化

预设背景图（柔和渐变/低对比墙纸），key 常量：`bg-warm-glow` / `bg-sand-hills` / `bg-peach-blossom` / `bg-sky-cloud` / `bg-mint-leaf` / `bg-berry-dots`。背景图作用于**页面画布**（`--at-bg-page` 之上），叠加 `blur`（0–20px）后作为 `body`/`#app` 的装饰层；虚化越大越像纯色氛围。背景色（`kind=color`）与背景图互斥，二选一。

## 4. 解析优先级（Resolution）

```ts
// web/src/stores/theme.ts
function resolveTheme(personal: ThemeConfig | null, household: ThemeConfig | null): ThemeConfig {
  const system = SYSTEM_THEME;                       // 内置常量
  if (personal && personal.useDefault === false) return merge(system, personal);
  if (household) return merge(system, household);
  return system;
}
```

`merge(base, override)` 做浅层覆盖：`override` 中未显式提供的字段沿用 `base`。个人层 `useDefault === true` 或 `personal === null` 时，整体回落到家庭默认（再回落系统）。

「使用默认配置」开关 = 把个人层 `useDefault` 置 `true`（同时保留上次自定义，方便再次切回）。「恢复默认」= 清除个人自定义（置 `null` 或 `useDefault:true` 且清空字段）。

## 5. 数据模型与迁移

### 5.1 迁移 `migrations/0005_style_settings.sql`

仅追加两列（向前迁移，遵循 CodingStandards §9）：

```sql
ALTER TABLE household_settings ADD COLUMN style TEXT;   -- 家庭默认样式 JSON，可空
ALTER TABLE user_preferences ADD COLUMN style TEXT;      -- 个人样式 JSON，可空
```

无需独立表：样式是低基数、单值、可整体覆盖的 JSON，沿用现有 `default_dashboard` 的存储范式。预设资产（色调/背景图/角图）为前端静态资源，不落库、不建媒体记录。

### 5.2 模型变更（internal/model/model.go）

- `Household` 增加 `Style *string json:"style"`。
- `Preferences` 增加 `Style *string json:"style"`。

### 5.3 头像存储

`users.avatar_key` 已存在。新增头像上传/读取：

- 存储：`{MediaDir}/avatars/<random-token>.<ext>`，替换时删除旧文件。
- 读取：`GET /api/v1/avatars/{key}`（`requireAuth`，家庭成员可见）。
- 尺寸：限制边长 ≤ 2048px，体积 ≤ `MaxImageBytes`，生成 96px 缩略图用于列表。

## 6. 设计令牌扩展（tokens.css）

新增一组「运行时样式变量」，由 `theme.ts` 在 `document.documentElement` 上写入，未设置时回落到现有静态值：

```css
:root {
  /* 卡片圆角运行时可覆盖 */
  --at-radius-card: var(--at-radius-lg);   /* sm/md/lg/xl 映射 8/12/16/24px */

  /* 色调覆盖层（由预设写入） */
  --at-bg-page-override: none;
  --at-bg-surface-override: none;
  --at-color-hairline-override: none;
  --at-color-hairline-strong-override: none;
  --at-color-muted-override: none;
  --at-color-muted-soft-override: none;
}
```

`.at-card` 的 `border-radius` 改为 `var(--at-radius-card)`；`body`/画布背景读取 `var(--at-bg-page-override, var(--at-bg-page))`。这样既保留现有所有页面，又能整体换肤。

色调预设采用 `data-theme="warm-sun"` 属性选择器 + CSS 变量块实现，`applyTheme()` 只做两件事：设置 `html[data-theme]`、写入圆角与覆盖变量。

## 7. 前端实现

### 7.1 新增/变更文件

```text
web/src/
├─ data/theme.ts                 # SYSTEM_THEME、tone 预设、角图/背景图资产表
├─ stores/theme.ts               # useThemeStore：解析/应用/持久化
├─ components/base/AppSlider.vue # 虚化程度滑块（新增基础组件）
├─ components/business/
│  ├─ StyleQuickMenu.vue         # 右上角快速设置（色调/圆角/恢复默认）
│  └─ StyleSettingsPanel.vue     # 完整样式设置（默认色/背景图+虚化预览/角图）
├─ components/AppShell.vue       # 顶栏增加「样式」图标入口
└─ views/SettingsView.vue        # 新增「样式设置」面板（个人 + 管理员家庭默认）
```

依赖方向：`views/business → stores/theme → data/theme`，基础组件 `AppSlider` 不依赖 Pinia（符合 CodingStandards §16）。

### 7.2 主题 Store（useThemeStore）

```ts
state: {
  personal: ThemeConfig | null;      // 当前用户个人样式
  household: ThemeConfig | null;     // 家庭默认样式（管理员可见/可编辑）
  effective: ThemeConfig;            // 解析后的生效样式
}
actions: {
  load(): Promise<void>;             // 从 /bootstrap 或 /me/preferences 读取
  setPersonal(partial): Promise<void>;   // 个人自定义（写 user_preferences.style）
  setUseDefault(flag): Promise<void>;    // 「使用默认配置」开关
  resetPersonal(): Promise<void>;        // 恢复默认
  setHousehold(partial): Promise<void>;  // 管理员写家庭默认（household_settings.style）
  apply(): void;                     // 解析并写入 DOM（data-theme + CSS 变量）
}
```

`apply()` 在 `load()`、任何 `set*` 成功后调用；`App.vue` 挂载时 `await theme.load()` 再渲染，避免首屏闪烁（或先以系统默认渲染、加载后平滑过渡）。

### 7.3 右上角快速设置入口

在 `AppShell.vue` 的 `topbar__actions` 中，头像菜单之前新增 `AppIconButton icon="palette" label="样式"`。点击弹出 `StyleQuickMenu`（下拉/浮层，复用现有 `avatar-menu__panel` 的定位风格）：

- 色调 swatch 网格（11 个，含「默认」）。
- 圆角四档（预览小方块）。
- 「恢复默认」按钮 + 「更多样式设置」跳转 `/settings`。

所有改动即时 `apply()` 并节流持久化；关闭浮层不丢未保存值（已在点击时写入）。

### 7.4 系统设置 · 样式面板

`SettingsView` 新增 `AppPanel`：

- **家庭默认样式**（仅管理员，写 `household_settings.style`）：默认色调、背景色（取色器）、背景图（预设缩略图选择 + 虚化滑块 + 实时预览）、卡片角图（花纹/卡通缩略图选择，默认无）。
- **我的样式**（所有成员，写 `user_preferences.style`）：顶部「使用默认配置」开关；关闭后展示与家庭默认相同的编辑控件；「恢复默认」按钮。

预览区：面板内嵌一个迷你卡片 + 画布，实时反映当前选择，做到所见即所得。

### 7.5 个人设置（SettingsView 个人资料区扩展）

在现有「个人资料」面板扩展：

- 头像：`AppAvatar` 可点击 → 上传/选择头像（`POST /api/v1/me/avatar`）。
- 昵称：沿用现有「称呼」字段（`displayName`）。
- 显示的卡片：跳转首页「整理首页」（现有 `DashboardCustomizer`），或在此内嵌卡片可见性清单。
- 自定义样式：见 §7.4「我的样式」。

### 7.6 家庭成员管理（FamilyView 扩展，管理员）

`openEdit` 弹窗增加（仅管理员对他人可见）：

- 头像（复用头像上传，写目标用户 `avatar_key`）。
- 昵称（现有 `displayName`）。
- 显示的卡片：为成员设定默认卡片集（写该成员 `user_preferences.dashboard`，复用现有卡片布局 JSON）。

## 8. 后端实现

### 8.1 路由新增（internal/httpapi/server.go）

```go
mux.Handle("POST "+api+"/me/avatar", s.requireAuth(http.HandlerFunc(s.handleAvatarUpload)))
mux.Handle("GET "+api+"/avatars/{key}", s.requireAuth(http.HandlerFunc(s.handleAvatarGet)))
mux.Handle("PATCH "+api+"/family/members/{id}/style", s.requireAdmin(http.HandlerFunc(s.handleMemberStylePatch)))   // 管理员为成员设默认卡片集
```

### 8.2 现有处理器扩展

- `householdPatchRequest` 增加 `Style *string`，校验 JSON 后写入 `household_settings.style`。
- `preferencesPatch` 增加 `Style *string`，校验 JSON 后写入 `user_preferences.style`。
- `mePatchRequest` 已支持 `AvatarKey`，无需变更（头像上传后写 `avatar_key`）。

### 8.3 校验

- `Style` 走现有 `validateJSONField` 校验 JSON 合法性；服务端不解析主题语义（语义由前端承担），仅保证可序列化。
- 头像：复用 `detectMedia` 校验 MIME 仅允许 `image/*`；`MaxBytesReader` 限流；生成缩略图。

## 9. API 契约（新增）

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/api/v1/me/avatar` | 登录 | multipart `file`，返回 `{ avatarKey }` |
| GET | `/api/v1/avatars/{key}` | 登录 | 返回头像图片 |
| PATCH | `/api/v1/family/members/{id}/style` | 管理员 | `{ dashboard?: string }`，为成员设默认卡片集 |
| PATCH | `/api/v1/household` | 管理员 | 新增字段 `style`（家庭默认样式） |
| PATCH | `/api/v1/me/preferences` | 登录 | 新增字段 `style`（个人样式） |

## 10. 无障碍与响应式

- 色调 swatch 提供 `aria-label`（中文名称）+ `aria-pressed` 状态（复用 `DashboardCustomizer` 的 swatch 模式）。
- 圆角/虚化用 `AppSlider`，提供可见数值与键盘方向键调节，触控目标 ≥ 44px。
- 角图/背景图均为纯装饰，`aria-hidden` + `pointer-events: none`，不影响对比度（正文仍用品牌墨色）。
- 尊重 `prefers-reduced-motion`：色调/圆角切换不做动画，仅背景图虚化过渡在非 reduced-motion 下渐变。

## 11. 验收清单（Review 用）

- [ ] 右上角样式图标可打开快速设置；改色调/圆角立即生效并持久化到个人。
- [ ] 个人开启「使用默认配置」后，界面跟随家庭默认/系统默认；关闭后恢复个人自定义。
- [ ] 管理员在系统设置可改家庭默认色调/背景色/背景图（含虚化预览）/卡片角图，全员（未自定义者）生效。
- [ ] 卡片角图仅出现在卡片右下角、为装饰、不遮内容、默认无。
- [ ] 个人设置可上传头像、改昵称、选显示卡片、开关自定义样式；管理员可在家庭管理为成员设置头像/昵称/卡片集。
- [ ] 「恢复默认」一键回到系统默认，且不误删其他成员的个性化。
- [ ] 所有改动在移动端（<640px）可操作、可预览。
- [ ] 品牌红线不破：主 CTA 仍为墨色，无高饱和品牌色/蓝色引入。

## 12. 与现有文档关系

- 数据模型沿用 [02-data-model.md](02-data-model.md) 的 JSON 列范式，新增迁移 `0005`。
- 前端令牌扩展见 [04-frontend.md](04-frontend.md) §6；组件纪律沿用 CodingStandards §16。
- 不引入新依赖（滑块自研 `AppSlider`，头像复用现有媒体检测/缩略图逻辑）。
