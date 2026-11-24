# 编码与 WebUI 规范

| 项目 | 内容 |
|---|---|
| 文档编号 | STD-ENG-001 |
| 文档版本 | v1.1 |
| 制定依据 | Google Style Guide（有对应语言时优先）、语言/框架官方规范、本项目设计文档 |
| 强制程度 | 标注“必须”的条目在代码评审与 CI 中强制检查；其余为约定 |

## 1. 优先级与冲突处理

规范优先级从高到低：

1. 本文档中的**硬性约定**（注释上限、基础组件复用、目录结构、安全红线）。
2. 对应语言的 Google Style Guide。
3. 语言/框架官方规范（如 Rust API Guidelines、Vue 官方风格指南）。
4. 同一模块内既有代码的一致写法。

冲突时按上述顺序裁决，不做“个人习惯优先”。无法覆盖的边界情况在评审中记录理由，不新增隐式约定。

## 2. 规范来源

| 范围 | 采用规范 | 说明 |
|---|---|---|
| Go | [Google Go Style Guide](https://google.github.io/styleguide/go/) | 命名、包结构、错误处理、测试；以 `gofmt` 输出为准。 |
| JavaScript | [Google JavaScript Style Guide](https://google.github.io/styleguide/jsguide.html) | 2 空格、语句必须以分号结尾、字符串用单引号；由 ESLint 固化。 |
| TypeScript | [Google TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html) | 禁用 `any`、显式返回类型用于公共 API。 |
| HTML/CSS | [Google HTML/CSS Style Guide](https://google.github.io/styleguide/htmlcssguide.html) | 类名小写连字符；禁止内联样式（动态计算除外）。 |
| SQL | [Google SQL Style Guide](https://google.github.io/styleguide/sqlguide.html) | 关键字大写、缩进 2 空格、snake_case。 |
| Protobuf | [Protobuf Style Guide](https://protobuf.dev/programming-guides/style/) | 消息 CamelCase、字段 snake_case、枚举 UPPER_SNAKE。 |
| Shell | [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html) | 仅用于运维脚本；业务逻辑禁止用 Shell。 |
| Markdown | [Google Markdown Style Guide](https://google.github.io/styleguide/docguide/style.html) | 文档统一 ATX 标题、代码块带语言标签。 |
| JSON/YAML | [Google JSON Style Guide](https://google.github.io/styleguide/jsoncstyleguide.html) | JSON 2 空格；YAML 2 空格、禁止 Tab。 |
| Rust | [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/) + `rustfmt` + `clippy` | Google 无官方 Rust 规范，采用 Rust 官方生态标准。 |
| Vue | [Vue 官方风格指南](https://vuejs.org/style-guide/)（优先级 A/B 必守） | Google 无 Vue 规范；组件与样式另见本文第 15–18 节。 |

**工具优先原则：** 格式化争议一律以仓库内配置文件输出为准，评审不讨论空格、换行、引号等机器可判定问题。

## 3. 通用原则

- 可读性优先于技巧性；不使用需要“聪明解读”的写法。
- 单一职责：一个函数只做一件事；函数体超过 60 行需说明理由。
- 早返回，减少嵌套；嵌套深度不超过 3 层。
- 显式优于隐式：不依赖隐式类型转换、隐式全局状态、隐式副作用。
- 命名表达意图，不用缩写（行业通用缩写除外：ID、URL、HTTP、CPU、SQL）。
- 不写“将来可能用到”的抽象；不为假设需求预留扩展点。
- 删除死代码，不保留注释掉的代码；历史由版本库保存。
- 业务常量集中定义，禁止魔法数字散落；协议字段除外（需注释来源）。

## 4. 注释规范（硬性）

### 4.1 基本原则

- **默认不写注释**：通过命名、类型、函数拆分表达意图。
- **确实需要时不超过 2 行**（含文档注释、行内注释、块注释）。
- 注释解释“为什么”，不复述“做了什么”；能被代码直接读出的内容不注释。
- 禁止：注释掉的代码、变更日志式注释、分隔横幅、ASCII 图案、表情符号。
- 禁止：与代码不一致的过期注释；改代码必须同步改注释，无法同步则删除。
- 注释语言为中文，标识符与协议字段保持英文；专有名词与产品文档一致。

### 4.2 允许注释的场景

| 场景 | 示例 |
|---|---|
| 外部协议/硬件约束 | `// 设备要求写后延迟 50ms 再读，否则返回旧值` |
| 安全/合规原因 | `// 危险指令不重试，避免二次动作` |
| 非显然的算法边界 | `// 使用左闭右开区间，避免跨窗口重复计数` |
| 兼容性/临时规避 | `// TODO(zhangsan): 数据中心 2.3 修复后移除，见 ISSUE-142` |
| 公共 API 契约 | `// 返回值在超时后为 nil，调用方需判空` |

### 4.3 格式

```go
// 补值只补值不补时间，采集时间必须保留原值
func (f *Filler) Fill(p Point) (Point, error) { ... }
```

```rust
// 写后需等待 50ms，设备固件会返回上一周期的值
const WRITE_SETTLE: Duration = Duration::from_millis(50);
```

```ts
// 数值列右对齐，避免长数字阅读错位
const NUMERIC_CELL_CLASS = 'text-right tabular-nums'
```

- 行内注释与代码间隔至少一个空格；行内注释只用于单行说明。
- `TODO` 必须带责任人（或团队）与问题编号；无归属的 TODO 不允许合入。
- 公共 API 文档注释最多 2 行，只写契约、边界与错误语义，不写用法教程。

## 5. 命名规范

| 对象 | 规则 | 示例 |
|---|---|---|
| Go 包 | 小写单词，无下划线、无复数 | `ingest`、`unitregistry` |
| Go 变量/函数 | `mixedCaps`，首字母缩写全大写 | `endpointID`、`parseMQTTTopic` |
| Go 接口 | 单方法用 `-er` 名词 | `Reader`、`Validator` |
| Go 错误 | `ErrXxx` 哨兵，错误文本小写无标点 | `ErrQueueFull` |
| Rust 类型/trait | `CamelCase` | `ChangeDetector`、`ProtocolDriver` |
| Rust 函数/模块/变量 | `snake_case` | `fill_value`、`ingest_pipeline` |
| Rust 常量 | `SCREAMING_SNAKE_CASE` | `MAX_BATCH_POINTS` |
| TS 变量/函数 | `camelCase`；布尔用 `is/has/can` 前缀 | `isLoading`、`canSubmit` |
| Vue 组件 | `PascalCase`，基础组件 `App` 前缀 | `AppDialog.vue`、`AppPanel.vue` |
| Vue 组合式函数 | `use` 前缀 | `usePagination` |
| Pinia store | `useXxxStore` | `useAlarmStore` |
| CSS 类 | 小写连字符，基础组件用 `at-` 前缀 | `at-panel__header` |
| SQL 表/列 | `snake_case`；模块前缀保留 | `st_point_queue`、`cc.task_step` |
| 主题变量 | `--at-` 前缀 + 语义名 | `--at-color-danger` |

- 不使用拼音命名；业务术语以产品文档为准。
- 同一概念全项目同名：`endpoint`、`gateway`、`station`、`device`、`attribute`，不混用 `point/sensor/tag`。
- 集合用复数，单条用单数；时间字段统一 `_at`（时间点）或 `_ms`（时长）。

## 6. Go 规范

- 入口 `main` 只做装配，不写业务；依赖通过构造函数注入。
- `context.Context` 必须是第一个参数，不存入结构体；所有外部调用必须带超时。
- 错误必须包装：`fmt.Errorf("load config: %w", err)`；错误文本小写、无结尾标点。
- 禁止在库代码中 `panic`；`panic` 仅用于不可恢复的初始化错误。
- 接口由使用方定义，不用接口预造抽象；接口不超过 3 个方法。
- 不使用全局可变状态；确需缓存时用显式结构体持有。
- goroutine 必须有退出路径：由 `ctx`、关闭通道或 `WaitGroup` 收敛；禁止裸 `go func()`。
- 通道必须有明确所有权与关闭方；生产端关闭，消费端不关闭。
- 大对象传递用指针；小结构体按值；避免在热路径分配。
- 结构体字段按逻辑分组，导出字段需有 2 行内注释说明契约。
- 测试使用表驱动，命名 `TestXxx_场景`；并发测试启用 `-race`。

## 7. Rust 规范

- 库代码禁止 `unwrap`/`expect`（测试与启动期常量除外）；错误用 `thiserror` 定义，边界层用 `anyhow`。
- 异步代码禁止阻塞调用；必须使用 `spawn_blocking` 或专用线程。
- 所有协议调用必须 `timeout`；重连使用带抖动的指数退避。
- 不滥用 `clone`；热路径用借用与 `Arc` 明确共享。
- 模块按职责拆分，`lib.rs` 只做导出；`unsafe` 需注释安全前提（不超过 2 行）并通过评审。
- 公共类型实现 `Debug`；敏感字段自定义 `Debug` 脱敏。
- 错误类型不吞掉上下文；跨层转换保留 `source()`。
- `clippy` 警告视为错误；`rustfmt` 统一格式。

## 8. TypeScript / Vue 规范

- 启用 `strict`；禁止 `any`、`@ts-ignore`；必要时用 `unknown` + 类型守卫。
- 对外 API 与 composable 显式声明返回类型。
- 使用 `<script setup lang="ts">` + Composition API；不使用 Options API 新增代码。
- Props/Emits 必须显式声明类型；禁止 `defineProps` 隐式 any。
- 组件不直接调用 `fetch`；统一走 `src/api/` 封装与错误处理。
- 业务状态放 Pinia；组件本地状态留在组件内；禁止跨组件直接改 store 内部状态。
- 模板中不做复杂表达式；超过一个运算即提取为 `computed`。
- `v-for` 必须带稳定 `:key`；禁止用索引作为可变更列表的 key。
- 禁止 `v-html`（除受控富文本，需 sanitize）。
- 路由懒加载；页面级组件放 `views/`，基础组件放 `components/base/`。
- 单元测试用 Vitest；组件测试优先测行为，不测实现细节。

## 9. SQL / Protobuf / 配置规范

**SQL**

- 关键字大写，标识符小写；缩进 2 空格；每行一个字段。
- 表必须有主键；外键显式声明；时间统一 `timestamptz`（PG）或 `DateTime64(3,'UTC')`（CH）。
- 迁移只向前追加，不修改已发布迁移；破坏性变更必须附备份与回滚说明。
- 索引命名 `idx_<表>_<列>`，唯一索引 `uq_<表>_<列>`；索引必须对应查询场景。
- 禁止 `SELECT *` 进入生产查询；禁止在事务中做远程调用。

**Protobuf**

- 包名 `things.<域>.v1`；消息与枚举 CamelCase / UPPER_SNAKE。
- 字段编号一经发布不复用；删除字段用 `reserved`。
- 枚举首值为 `*_UNSPECIFIED = 0`；新增枚举值需兼容旧客户端。
- 破坏性变更必须升版本（`v2`），不静默改语义。

**配置**

- YAML 2 空格缩进；键名 `snake_case`；禁止 Tab 与锚点滥用。
- 密钥、证书、令牌不得写入仓库；用环境变量或密钥文件挂载。
- 配置必须有默认值与校验；启动时快速失败并输出明确错误。

## 10. 错误处理与日志

- 错误分层：底层保留原因，中层补充上下文，顶层决定用户可见信息。
- 对外错误码稳定（见各设计文档错误码表），不把内部堆栈返回给调用方。
- 可重试与不可重试必须显式区分；重试必须带退避、上限与幂等键。
- 日志使用结构化字段：`trace_id/task_id/endpoint_id/station_id` 等，禁止字符串拼接关键字段。
- 日志级别：`error` 仅用于需要人工介入；`warn` 用于可自恢复异常；禁止在循环内刷 `error`。
- 禁止记录密码、令牌、证书私钥、完整报文中的敏感字段；必要时脱敏后记录。
- 不吞错误：捕获后要么处理，要么包装上抛，要么记录并明确降级。

## 11. 并发与资源

- 所有队列有界；队列满的行为必须明确（阻塞/拒绝/丢弃并记录），默认选择背压。
- 每个外部依赖设置超时、并发上限与熔断；禁止无上限 goroutine/线程池。
- 共享状态用锁或消息传递二选一，不混用；锁内不做 IO。
- 优雅退出：停止接收 → 处理在途 → 刷盘 → 关闭连接；退出超时可配。
- 资源使用与设计文档指标对齐；新增热点路径需给出基准测试。

## 12. 测试规范

### 12.1 测试分层与准入门槛

| 层级 | 范围 | 工具 | 执行时机 | 准入要求 |
|---|---|---|---|---|
| 单元测试 | 纯函数、协议解析、状态机、校验、编排决策 | Go `testing`、Rust `cargo test`、Vitest | 每次提交/CI | 新增或修改逻辑必须覆盖；核心模块分支覆盖 ≥ 80% |
| 集成测试 | 仓储、NATS、PostgreSQL/ClickHouse、gRPC 插件、配置下发 | testcontainers / docker compose | CI | 关键链路必须通过 |
| 契约测试 | OpenAPI、protobuf、跨模块消息 | 生成物比对 + 消费方契约用例 | CI | 协议字段变更必须同步 |
| 端到端 | 网关 → 数据站 → 数据中心 → 控制中心 → 处理 → 回写 | 容器化环境 | Nightly / 发版前 | 核心场景通过 |
| 压力测试 | 吞吐、延迟、积压、恢复、长稳 | k6/vegeta、`go test -bench`、`cargo bench`、场景压测器 | Nightly / 发版前 | 达到设计文档指标，无泄漏与无界增长 |

### 12.2 单元测试（硬性）

- 新增业务逻辑必须同时提交单元测试；修复缺陷必须先提交可复现测试（红 → 绿）。
- 测试命名表达场景与期望：Go `TestXxx_场景`，Rust `test_场景`，TS `describe('场景')` + `it('期望')`；禁止 `Test1`、`testFoo`。
- 优先表驱动；至少覆盖正常、边界、错误、并发、幂等五类场景。
- 测试必须确定性：不依赖真实网络、真实时钟、随机数与全局状态；时间、随机数、ID 生成通过接口注入。
- 并发测试必须启用 `-race`（Go）；不得用 `sleep` 等待异步结果，改用通道、`WaitGroup` 或轮询断言。
- 外部依赖使用 fake/stub；单元测试中禁止启动真实 PostgreSQL/ClickHouse/NATS/MinIO。
- 断言必须校验返回值、错误码与副作用，禁止只断言“未 panic”或仅覆盖 happy path。
- 覆盖率不作为唯一指标，但核心模块（协议解析、补值、幂等、路由、告警、权限）分支覆盖 ≥ 80%，整体 ≥ 60%；未达标需在评审中说明原因。
- 测试文件与被测代码同包/同目录；公共测试辅助代码放 `testutil`，不得进入生产构建。

### 12.3 压力与性能测试（硬性）

- 触发条件：新增或修改热点路径、容量相关配置、队列/并发模型、序列化协议、数据库索引或批量写入策略。
- 压测目标必须引用设计文档指标，至少覆盖：网关单节点上行 TPS、端点采集周期、站级批量落盘、控制中心任务吞吐（基线 2000 任务/秒）、数据中心写入吞吐与查询 P95、告警投递延迟。
- 压测场景至少包含：
  - **稳态**：目标负载持续 30 分钟，P95/P99 延迟、错误率、CPU/内存/IO 稳定。
  - **峰值**：2× 目标负载 10 分钟，无 OOM、无队列无界增长，背压生效。
  - **突发**：5× 瞬时流量 1 分钟，限流、熔断、拒绝策略符合预期。
  - **恢复**：停止流量后 5 分钟内积压消化，无重复、无丢失（幂等校验）。
  - **长稳**：24 小时浸泡，RSS、文件句柄、连接数无持续增长。
- 压测报告必须记录：版本、硬件、数据集规模、并发模型、指标曲线、瓶颈定位、结论与后续项；随 PR 或发版归档。
- 压测数据必须脱敏且可重复生成，禁止使用生产数据；压测在隔离环境执行，使用独立 schema、主题与桶前缀。
- 性能回归：关键 benchmark 偏离基线超过 10% 需说明并评审；重量级压测放 Nightly，CI 只跑轻量基准与冒烟。

### 12.4 集成测试与故障注入

- 集成测试使用容器化依赖（PostgreSQL/ClickHouse/NATS/MinIO），可本地一键启动，用例结束清理。
- 关键链路（补值、幂等、重试、死信、告警路由、指令白名单、配置下发、断网补传）必须有故障注入测试。
- 故障注入至少覆盖：依赖超时、连接中断、节点重启、消息重复/乱序、磁盘满、队列满、时钟偏移。

### 12.5 测试数据与隔离

- 测试数据使用固定种子生成；跨用例隔离，使用独立 schema、主题、桶或表前缀。
- 禁止向共享环境或生产环境写入测试数据；禁止提交真实设备凭据与证书私钥。
- 用例结束后清理对象、消息与表，避免污染长稳与 Nightly 环境。

## 13. 代码组织与依赖

- 目录按职责分层，不按技术细节堆平：

```text
gateway/   src/{driver,collector,store,uplink,command}
station/   internal/{broker,ingest,rule,alarm,batch,config,admin}
control/   internal/{ingress,orchestrator,unit,alarm,command,access}
data/      internal/{ingest,writer,access,metadata,backup,admin}
web/       src/{api,components/base,components/business,composables,stores,styles,views}
```

- 依赖方向单向：`driver → collector → store → uplink`；禁止反向依赖与循环依赖。
- 共享代码放独立包，禁止复制粘贴；三处以上重复才抽象。
- 第三方依赖需评估许可证（GPL 类需法务确认）、维护活跃度与离线可用性。
- 锁定依赖版本，提交 lockfile；升级单独提交并说明影响。

## 14. 提交与评审

- 提交信息使用祈使句，说明“为什么”：`fix(station): 避免补值缺失时伪造数据`。
- 一个提交只做一件事；格式化与逻辑修改分开提交。
- 评审关注：正确性、边界、幂等、资源上限、错误处理、测试与压测结论、注释是否必要且 ≤ 2 行、UI 是否复用基础组件。
- 评审通过条件：CI 全绿、无新增 lint 警告、关键路径有测试、热点路径有基准或压测结论、文档同步更新。

## 15. WebUI 设计令牌

所有视觉值来自令牌，组件内禁止硬编码颜色、间距、圆角与阴影。

### 15.1 颜色（语义）

| 令牌 | 用途 | 亮色参考 | 暗色参考 |
|---|---|---|---|
| `--at-color-primary` | 主操作、选中态 | `#2563EB` | `#60A5FA` |
| `--at-color-success` | 正常、恢复 | `#16A34A` | `#4ADE80` |
| `--at-color-warning` | 预警 | `#D97706` | `#FBBF24` |
| `--at-color-danger` | 严重、破坏性操作 | `#DC2626` | `#F87171` |
| `--at-color-info` | 提示 | `#0891B2` | `#22D3EE` |
| `--at-bg-page` | 页面背景 | `#F6F7F9` | `#0F1115` |
| `--at-bg-surface` | 卡片/面板 | `#FFFFFF` | `#171A21` |
| `--at-bg-elevated` | 弹层/抽屉 | `#FFFFFF` | `#1F232B` |
| `--at-text-primary` | 主文本 | `#111827` | `#E5E7EB` |
| `--at-text-secondary` | 次要文本 | `#6B7280` | `#9CA3AF` |
| `--at-border` | 边框/分割线 | `#E5E7EB` | `#2A2F3A` |

- 语义色只通过 Vuetify theme 与 CSS 变量使用；组件不得直接写十六进制。
- 状态色与文字组合需满足对比度 ≥ 4.5:1。

### 15.2 间距、圆角、阴影、字体

| 类别 | 令牌 | 值 |
|---|---|---|
| 间距 | `--at-space-1/2/3/4/5/6/8` | 4/8/12/16/20/24/32px |
| 圆角 | `--at-radius-sm/md/lg` | 4/8/12px |
| 阴影 | `--at-shadow-panel` | `0 1px 2px rgba(0,0,0,.06)` |
| 阴影 | `--at-shadow-dialog` | `0 12px 32px rgba(0,0,0,.18)` |
| 字体 | `--at-font-family` | `Inter, "Noto Sans SC", system-ui` |
| 字号 | `--at-font-xs/sm/md/lg/xl` | 12/13/14/16/20px |
| 行高 | `--at-line-tight/normal` | 1.35/1.6 |
| 层级 | `--at-z-drawer/dialog/toast/tooltip` | 2300/2400/2500/2600 |

### 15.3 断点与密度

- 断点沿用 Vuetify：`sm 600 / md 960 / lg 1280 / xl 1600`。
- 表格密度：`compact 32px`、`default 40px`；管理页默认 `default`。
- 触控/点击目标最小 40×40px；图标按钮最小 36px。

## 16. WebUI 基础组件

基础组件位于 `web/src/components/base/`，统一 `App` 前缀。业务组件不得复制基础组件样式另起一套。

### 16.1 AppPage（页面骨架）

| 项 | 定义 |
|---|---|
| 结构 | 面包屑 → 页头（标题/描述/操作区）→ 内容区 |
| Props | `title`、`description`、`breadcrumbs`、`loading`、`maxWidth`（`fluid/1200/960`） |
| Slots | `actions`、`default` |
| 规则 | 页头高度 56px；操作区不超过 3 个主按钮，其余收进“更多”；内容区统一左右 24px 内边距。 |

### 16.2 AppPanel（面板/卡片）

| 项 | 定义 |
|---|---|
| 结构 | header（标题/副标题/工具区）→ body → footer（可选） |
| Props | `title`、`subtitle`、`collapsible`、`defaultCollapsed`、`loading`、`bordered`、`padded`、`dense` |
| Slots | `header`、`actions`、`default`、`footer`、`empty` |
| Events | `toggle` |
| 规则 | header 48px、标题单行省略；body 默认 16px 内边距；无内容时显示 `AppEmptyState`；面板之间间距 16px。 |

### 16.3 AppDialog（对话框）

| 项 | 定义 |
|---|---|
| 结构 | header（标题/关闭）→ body（可滚动）→ footer（操作） |
| Props | `modelValue`、`title`、`size`（`sm 480 / md 640 / lg 880 / full`）、`variant`（`default/danger`）、`loading`、`persistent`、`closeOnEsc`、`closeOnBackdrop`、`showClose` |
| Slots | `default`、`footer`、`title` |
| Events | `update:modelValue`、`confirm`、`cancel`、`opened`、`closed` |
| 规则 | 桌面端垂直居中、移动端全宽；body 最大高度 `70vh` 并滚动；footer 右对齐，取消在左、主操作在右；`loading` 时禁止关闭与重复提交；禁止嵌套 Dialog（用 Drawer 或页内步骤替代）；破坏性操作必须 `variant=danger` 且按钮文案为动词。 |

```vue
<AppDialog v-model="visible" title="停用端点" size="sm" variant="danger" @confirm="disable">
  <p>停用后该端点不再采集，历史数据保留。</p>
</AppDialog>
```

### 16.4 AppConfirmDialog

| 项 | 定义 |
|---|---|
| 用途 | 二次确认，封装 AppDialog 的确认语义 |
| Props | `modelValue`、`title`、`message`、`confirmText`、`cancelText`、`danger`、`requireText`（需输入指定文本才可确认） |
| Events | `confirm`、`cancel` |
| 规则 | 删除、停用、回滚、下发控制指令必须使用；破坏性操作按钮不使用“确定”，使用“删除/停用/回滚”。 |

### 16.5 AppTable（数据表）

| 项 | 定义 |
|---|---|
| 结构 | 工具区 → 表头 → 行 → 空态/错误态 → 分页 |
| Props | `columns`、`rows`、`loading`、`error`、`density`、`selectable`、`rowKey`、`serverPagination`、`total`、`page`、`pageSize`、`stickyHeader`、`emptyText` |
| Slots | `toolbar`、`cell-<key>`、`row-actions`、`empty`、`error` |
| Events | `update:page`、`update:pageSize`、`selectionChange`、`sortChange`、`retry` |
| 规则 | 服务端分页为默认；数字列右对齐并使用等宽数字；列宽设置最小值，文本超长省略并 Tooltip；行操作超过 3 个收进更多菜单；加载用骨架屏，不用全屏遮罩；空态与错误态必须区分。 |

### 16.6 AppForm / AppFormField

| 项 | 定义 |
|---|---|
| 结构 | 字段标签 → 控件 → 帮助/错误文本 |
| Props（Form） | `model`、`rules`、`labelWidth`、`layout`（`vertical/inline`）、`disabled`、`loading` |
| Props（Field） | `label`、`required`、`hint`、`error`、`help`、`span` |
| 规则 | 新增/编辑页用 `vertical`，筛选栏用 `inline`；必填用星号而非文字；错误文本单行；帮助文本不超过一行；提交失败聚焦首个错误字段。 |

### 16.7 其他基础组件

| 组件 | 用途 | 关键约定 |
|---|---|---|
| `AppToolbar` | 列表/面板工具条 | 左侧筛选、右侧操作；换行优先于横向滚动。 |
| `AppStatusTag` | 状态标签 | `success/warning/danger/info/neutral` 五态；只传语义，不传颜色。 |
| `AppEmptyState` | 空态 | 图标 + 一句说明 + 可选主操作；禁止只用“暂无数据”。 |
| `AppErrorState` | 错误态 | 错误摘要 + 重试按钮；不展示堆栈。 |
| `AppLoadingOverlay` | 局部加载 | 仅覆盖目标区域；超过 300ms 才显示，避免闪烁。 |
| `AppDrawer` | 侧滑详情/编辑 | `sm 400 / md 560 / lg 720`；用于替代嵌套 Dialog。 |
| `AppToast` | 轻提示 | 成功 3s、错误 6s；同一消息合并；错误需带可定位信息。 |
| `AppCodeBlock` | 只读代码/报文 | 等宽字体、最大高度滚动、复制按钮。 |

**组件 API 约定：**

- Props 只表达状态与外观，不承载业务逻辑；业务数据通过 props 传入，事件向上抛出。
- 双向绑定统一 `modelValue` + `update:modelValue`；复杂表单用 `model` + `update:model`。
- 扩展优先 props/slots，禁止复制组件改一份；确需变体时新增受控 prop。
- 基础组件不依赖 Pinia、不直接请求 API、不读路由。
- 事件命名用动词过去式或语义动作：`confirm`、`cancel`、`selectionChange`。
- 所有交互元素必须有 `aria-label` 或可见文本；键盘可达，焦点样式使用主色描边。

## 17. WebUI 布局与交互规则

- 间距节奏：组件内 8/12/16px，模块间 16px，页面区块间 24px；不出现 5px、13px 等非令牌值。
- 对齐：同组元素左对齐；操作按钮右对齐；表格数字右对齐、文本左对齐。
- 标题层级：页面标题 20px/600，面板标题 14px/600，正文 14px/400，辅助 12px。
- 弹层顺序：页面 → Drawer → Dialog → Toast；避免同层级多个 Dialog。
- 表单提交：提交中禁用按钮并显示进度；失败保留用户输入；成功后关闭并刷新列表。
- 破坏性操作：必须二次确认；批量操作需显示影响数量。
- 加载与空态：列表、图表、详情都必须定义 loading/empty/error 三态。
- 响应式：≥1280 三栏，960–1279 两栏，<960 单栏；表格允许横向滚动，不隐藏关键列。
- 深色模式：只使用语义令牌；新增组件必须同时验证亮/暗两套主题。
- 国际化预留：文案不拼接字符串；日期时间统一格式并标注时区。

## 18. WebUI 工程结构与样式纪律

```text
web/src/
  api/            # 请求封装与错误归一化
  components/
    base/         # AppDialog/AppPanel/AppTable/AppForm...
    business/     # 业务组件，仅组合基础组件
  composables/    # usePagination/useDialog/usePolling
  stores/         # Pinia
  styles/
    tokens.css    # 设计令牌（唯一颜色/间距来源）
    base.css      # reset 与全局排版
  views/          # 页面
```

- 样式使用 `<style scoped>`；全局样式只允许在 `styles/` 中定义。
- 禁止 `!important`；禁止覆盖 Vuetify 内部类名，优先使用组件 props 与主题变量。
- 业务组件不得定义新的颜色/圆角/阴影，只能引用令牌。
- 图标统一使用一套图标库；尺寸 16/20/24px；纯图标按钮必须有 Tooltip 与 `aria-label`。
- 组件文件命名与组件名一致；一个文件一个组件；目录不按页面复制组件。
- 新增基础组件需同时提供：API 表、亮/暗主题截图、至少一个使用示例。

## 19. 工具链与 CI

| 语言/范围 | 工具 | 命令 |
|---|---|---|
| Rust | rustfmt | `cargo fmt --all -- --check` |
| Rust | clippy | `cargo clippy --all-targets -- -D warnings` |
| Rust | 单元测试 | `cargo test --all` |
| Rust | 基准测试 | `cargo bench` |
| Go | gofmt/goimports | `gofmt -l .` + `goimports -l .` |
| Go | golangci-lint | `golangci-lint run` |
| Go | 单元测试 | `go test -race ./...` |
| Go | 集成测试 | `go test -tags=integration ./...`（容器化依赖） |
| Go | 基准测试 | `go test -bench=. -benchmem ./...` |
| TS/Vue | ESLint + eslint-plugin-vue | `eslint . --max-warnings 0` |
| TS/Vue | 类型检查 | `vue-tsc --noEmit` |
| TS/Vue | 单元测试 | `vitest run` |
| 压测 | 场景脚本 | `make stress`（Nightly，隔离环境） |
| CSS | stylelint | `stylelint "src/**/*.{css,vue}"` |
| SQL | sqlfluff（可选） | `sqlfluff lint migrations` |
| 文档 | markdownlint（可选） | `markdownlint docs` |

- CI 必须包含格式化、静态检查、类型检查、单元测试、集成测试与迁移校验；压测与 24 小时长稳在 Nightly 或发版流水线执行。
- 压测任务使用独立环境与独立 schema/主题/桶，禁止与开发、测试共享。
- pre-commit 执行格式化与 lint 的快速子集；不允许 `--no-verify` 绕过。
- lint 规则调整需团队评审并记录原因，禁止个人本地禁用规则。

## 20. 检查清单

**提交前**

- [ ] 命名符合第 5 节，概念与设计文档一致。
- [ ] 注释均为必要注释，且每处 ≤ 2 行；无注释代码、无无主 TODO。
- [ ] 外部调用有超时、重试与幂等；队列有界；goroutine/任务有退出路径。
- [ ] 错误被包装或处理，未吞错；日志无敏感信息。
- [ ] 新增/修改逻辑有单元测试，覆盖边界、错误、并发与幂等；缺陷有回归测试。
- [ ] 核心模块分支覆盖 ≥ 80%；未达标已在评审说明。
- [ ] 热点路径或容量变更已跑基准/压测，指标达标且报告归档。
- [ ] 测试使用隔离环境与脱敏数据，用例结束已清理。
- [ ] 格式化与 lint 全绿。

**WebUI**

- [ ] 使用 `AppDialog/AppPanel/AppTable/AppForm` 等基础组件，未复制样式。
- [ ] 颜色、间距、圆角、阴影全部来自令牌。
- [ ] loading/empty/error 三态完整；破坏性操作有二次确认。
- [ ] 键盘可操作，图标按钮有 `aria-label`，对比度达标。
- [ ] 亮色与暗色主题均验证；960px 以下布局可用。
