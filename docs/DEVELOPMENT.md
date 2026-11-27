# 构建与测试说明

本文档说明朝夕（MyMoments）各模块的构建、整体构建与测试构建方法。所有命令以仓库根目录为工作目录，统一通过根目录的 [Makefile](../Makefile) 编排；也可直接使用 `go` / `npm` / `cargo` 原始命令。

## 模块划分

| 模块 | 目录 | 技术栈 | 构建产物 |
| --- | --- | --- | --- |
| 后端服务 | `cmd/server`、`internal/*`、`migrations/*` | Go 1.23+（`net/http` + `modernc.org/sqlite`） | 单文件二进制 `build/mymoments` |
| 前端 Web | `web/` | Vue 3 + TypeScript + Vite + PWA | 静态资源 `web/dist`，嵌入后端 |
| 原生壳 | `apps/`（`apps/ui` + `apps/src-tauri`） | Tauri 2 + Rust | macOS `.dmg` / `.app`、iOS `.ipa`、Android `.apk` |

三个模块相互独立：后端不依赖前端源码即可编译（静态资源通过 `go:embed` 嵌入占位页）；前端可独立用 Vite 开发；原生壳是连接既有服务器地址的 webview 容器，不打包前端。

## 前置依赖

| 工具 | 版本要求 | 用途 |
| --- | --- | --- |
| Go | 1.23+ | 后端编译、测试 |
| Node.js | 20+（CI 使用 22） | 前端与原生壳脚本 |
| npm | 随 Node | 依赖安装、脚本执行 |
| Rust | 1.77.2+ | 原生壳编译（仅 `apps/`） |
| Docker | 可选 | 镜像构建、多架构打包 |

原生壳还依赖平台 SDK：

- macOS：Xcode Command Line Tools（`xcode-select --install`）。
- iOS：Xcode 与 CocoaPods（`brew install cocoapods`）。
- Android：JDK 17、Android SDK platform-tools、NDK 27（`27.0.12077973`）。

## 各模块构建

### 后端（Go）

```bash
# 编译并运行（开发，数据写入 ./data）
go run ./cmd/server

# 仅编译，输出到 build/mymoments（不含前端资源时为占位页）
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o build/mymoments ./cmd/server
```

后端为纯 Go、免 CGO（SQLite 使用 `modernc.org/sqlite`），便于 amd64 / arm64 交叉编译。`migrations/` 通过 `go:embed` 嵌入向前迁移脚本，`internal/webui/` 通过 `go:embed all:static` 嵌入前端产物；未执行前端构建时提供占位页。

### 前端 Web

```bash
cd web
npm ci                # 按 package-lock.json 安装依赖（等价 make web-build 的第一步）
npm run dev           # 启动 Vite 开发服务器（默认 5173，/api 代理到 127.0.0.1:8080）
npm run build         # vue-tsc --noEmit 类型检查 + vite build，输出到 web/dist
npm run typecheck     # 仅类型检查
```

`web/` 下存在独立的 `go.mod`（模块名 `mymoments/web`），用于划清模块边界，避免根目录 Go 工具链扫描 `node_modules`。

### 原生壳（Tauri）

原生壳的 UI 是 `apps/ui/` 下的本地引导页，不依赖 `web/`。首次运行会先编译 Rust：

```bash
cd apps
npm install                 # 安装 @tauri-apps/cli
npm run tauri -- dev        # 本地运行 macOS 桌面壳（等价 make app）
```

构建安装包（未签名）：

```bash
# macOS（arm64 dmg）
cd apps && npm run tauri -- build --target aarch64-apple-darwin --bundles dmg

# iOS（先初始化工程，再构建未签名 ipa）
cd apps && npm run tauri -- ios init --ci
npm run tauri -- ios build --no-sign

# Android（先初始化工程，再构建未签名 apk）
cd apps && npm run tauri -- android init --ci
NDK_HOME=$ANDROID_HOME/ndk/27.0.12077973 npm run tauri -- android build --apk --target aarch64
```

从品牌 Logo 重新生成应用图标：

```bash
cd apps && npm run tauri -- icon ../docs/logo/mymoment.png
```

> 移动端需允许局域网 `http://` 明文访问：iOS 在 `apps/src-tauri/gen/apple` 的 `Info.plist` 配置 ATS（`NSAllowsLocalNetworking`），Android 在 `apps/src-tauri/gen/android/app/build.gradle.kts` 设置 `usesCleartextTraffic = true`。详见 [09-tauri-apps.md](design/09-tauri-apps.md) 与 [release-apps.yml](../.github/workflows/release-apps.yml)。

## 整体构建

### 源码构建单文件二进制

```bash
make build
```

等价于：`make web-build`（安装依赖 + 前端构建并复制到 `internal/webui/static`）+ 后端编译为 `build/mymoments`。

单独重新嵌入前端资源：

```bash
make web-build
```

`internal/webui/static` 必须整体重建（先 `rm -rf` 再整目录拷贝），只替换 `index.html` 会让它引用到不存在的 `assets/*.js`，后端回退返回 HTML 导致页面白屏。`make api`、`make api-up`、`make dev-up` 会先执行 `webui-check`，发现引用与文件不一致时自动重建；嵌入发生在 Go 编译期，重建后需重启 API 进程。

### 本地联调（前后端分离）

```bash
make dev-up      # 后台启动 API（:8080）与 Vite（:5173）
make dev-logs    # 跟踪日志
make dev-status  # 查看状态
make dev-down    # 停止
```

或手动分终端启动：

```bash
make api   # 终端一：后端，http://127.0.0.1:8080
make web   # 终端二：前端，http://127.0.0.1:5173
```

### Docker 镜像

```bash
make docker-build            # 按当前分支打 tag
docker compose up -d --build # 直接以 Compose 启动（推荐部署方式）
```

多架构（amd64 / arm64）本地打包：

```bash
make setup-multiarch         # 初始化 buildx（仅一次）
make build-multiarch-local
```

## 测试构建

### 后端测试

```bash
go test ./...          # 全部包
go test -race ./...    # 带竞态检测
go test ./internal/httpapi -run TestXxx -v   # 指定包 / 用例
```

测试文件位于各 `internal/*` 包下（如 `internal/httpapi`、`internal/holiday`、`internal/store`、`internal/weather`、`internal/logging`）。

### 前端测试与静态检查

```bash
cd web
npm run test       # Vitest（环境 jsdom，匹配 src/**/*.test.ts）
npm run lint       # ESLint（--max-warnings 0）
npm run typecheck  # vue-tsc --noEmit
```

### 一键全量检查

```bash
make check   # 等价 fmt + lint + test
make fmt     # gofmt -w ./cmd ./internal ./migrations
make lint    # go vet ./...  + 前端 npm run lint
make test    # go test -race ./...  + 前端 npm run test
```

## 常用 Make 目标速查

| 目标 | 说明 |
| --- | --- |
| `help` | 列出全部目标 |
| `dev-up` / `dev-down` / `dev-status` / `dev-logs` | 本地测试页面编排 |
| `api` / `api-up` / `api-down` / `api-logs` | 后端启停 |
| `web` | 启动前端 Vite 开发服务器 |
| `web-build` | 构建前端并嵌入静态资源 |
| `build` | 构建单文件二进制 |
| `app` / `app-build` / `app-icons` | 原生壳运行 / 构建 / 图标 |
| `test` / `fmt` / `lint` / `check` | 测试与静态检查 |
| `docker-build` / `setup-multiarch` / `build-multiarch-local` | 镜像构建 |
| `clean` | 清理 `build`、`web/dist`、`internal/webui/static` |

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `MYMOMENTS_DATA_DIR` | `./data` | 数据目录（数据库、媒体、日志） |
| `MYMOMENTS_ADDR` | `:8080` | 监听地址 |
| `MYMOMENTS_DEV` | 空 | 本地联调标记（`scripts/dev.sh` 注入） |
| `MYMOMENTS_DEFAULT_TZ` | `Asia/Shanghai` | 家庭默认时区 |
| `MYMOMENTS_SESSION_SECRET` | 自动生成 | 会话签名密钥 |

完整环境变量见 [README.md](../README.md) 与 [.env.example](../.env.example)。

## CI 参考

[ci.yml](../.github/workflows/ci.yml) 校验后端（gofmt 差异检查、`go vet ./...`、`go test -race ./...`）与前端（`npm ci`、`npm run lint`、`npm run build`）；[release-apps.yml](../.github/workflows/release-apps.yml) 构建 macOS / iOS / Android 未签名安装包并上传 GitHub Release。提交前建议在本地先跑 `make check`。
