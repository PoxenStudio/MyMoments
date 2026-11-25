# 07 · 部署与运维

## 1. 交付形态

- 单应用容器（PRD §18.3）：Go 服务内嵌 Vue 构建产物；SQLite 与媒体同卷持久化。
- 镜像标签化发布；`linux/amd64` + `linux/arm64` 双架构。
- 标准部署：准备 `.env` → `docker compose up -d`。

## 2. Dockerfile（多阶段）

```dockerfile
# 阶段 1：构建前端
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 阶段 2：构建后端
FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./web/dist      # 内嵌前端产物
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# 阶段 3：运行时
FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/server /usr/local/bin/mymoments
USER app
ENV DATA_DIR=/data
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["mymoments"]
```

- `CGO_ENABLED=0` 依赖 `modernc.org/sqlite`（纯 Go）免 CGO，交叉编译 amd64/arm64 无需工具链。
- 非 root 用户 `app` 运行；数据写入 `/data`。

## 3. docker-compose.yml

```yaml
services:
  mymoments:
    image: mymoments:latest
    restart: unless-stopped
    env_file: .env
    ports:
      - "8080:8080"
    volumes:
      - mymoments-data:/data
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://127.0.0.1:8080/readyz"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 15s

volumes:
  mymoments-data:
```

`.env.example`（不含秘密）：

```ini
MYMOMENTS_ADDR=:8080
MYMOMENTS_DATA_DIR=/data
MYMOMENTS_LOG_DIR=                    # 可选，默认 $MYMOMENTS_DATA_DIR/logs
MYMOMENTS_SESSION_SECRET=            # 必填，启动时若缺失且非初始化则报错
MYMOMENTS_DEFAULT_TZ=Asia/Shanghai
MYMOMENTS_WEATHER_PROVIDER=
MYMOMENTS_WEATHER_API_KEY=
```

## 4. 健康检查与就绪

| 端点 | 语义 |
|---|---|
| `/healthz` | 存活：进程在、HTTP 可响应 |
| `/readyz` | 就绪：迁移成功、SQLite 可读写、媒体目录可写 |

迁移失败时 `/readyz` 返回 503，`/healthz` 仍可响应（用于诊断），服务不报告健康、不覆盖旧数据（PRD §18.3）。Compose `healthcheck` 使用 `/readyz`。

## 5. 数据持久化与备份

- `/data` 含 `db/moments.db`（SQLite，WAL 模式）+ `media/`（媒体）+ `logs/`（运行日志）；同一卷、同一备份点（PRD §18.3）。
- **备份**：`sqlite3 /data/db/moments.db ".backup backup.db"` + 同步复制 `media/`；或停容器后整体复制卷。
- **日志**：默认写入 `/data/logs/moments.log`，单文件 5MB，保留 7 个文件（当前 + 6 个归档 `moments.N.log`），可用 `MYMOMENTS_LOG_DIR` 覆盖目录。
- **恢复**：停止容器 → 替换卷内容 → 启动；迁移在启动时自动前向执行。
- 文档明确说明备份/恢复/迁移步骤（PRD §18.3、§19.6）。

## 6. 迁移与升级

- 启动时执行 `migrations/` 前向迁移（见 [02-data-model](02-data-model.md) §4）。
- 升级前提示备份；回滚以恢复已备份数据卷为准。
- 迁移失败：非零退出、`/readyz` 503、输出可诊断错误。

## 7. HTTPS 与反向代理

- 容器仅监听内网端口；HTTPS 由用户自选反向代理（Caddy/Nginx/Traefik）终止（PRD §18.3）。
- 文档说明：首次初始化、端口访问、HTTPS 对浏览器定位（`geolocation` 需安全上下文）与通知能力的影响。
- PWA 安装与 Service Worker 要求 HTTPS（本机 `localhost` 例外，PRD §18.1 开发例外）。

## 8. 首次初始化

- 空库打开 Web → 进入初始化向导，创建管理员（PRD §19.1）。
- 禁止预置通用管理员/默认密码（PRD §18.3）。
- 并发初始化请求最多创建一名管理员：初始化流程在数据库事务 + 唯一约束下完成（PRD §19.1.2）。

## 9. 与 PRD §21 待确认项的对应

| 待确认项 | 本文档处理 |
|---|---|
| §21.1 天气服务商 | 预留 `MYMOMENTS_WEATHER_PROVIDER/API_KEY`，代理层解耦，实现期选定 |
| §21.3 TLS/反代/恢复 | §7 说明，实现期补操作手册 |
| §21.5 Web Push | [05-pwa](05-pwa.md) §6 分期，首期不承诺 |
| §21.6 媒体编码/容量 | [03-backend-api](03-backend-api.md) §7 与 [06-security](06-security.md) §6，上限实现期确定 |
| §21.8 产品层设计系统 | [04-frontend](04-frontend.md) §6 令牌映射，组件库随开发补齐 |
| §21.9 时光轴性能预算 | [03-backend-api](03-backend-api.md) §8，实现期基准确定 |
