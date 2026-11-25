.PHONY: help dev dev-up dev-down dev-status dev-logs api api-up api-down api-restart api-status api-logs web web-build build test fmt lint check docker clean

GO ?= go
NPM ?= npm

help: ## 列出可用目标
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

dev: dev-up

dev-up: ## 启动本地测试页面（API + Vite）
	@sh scripts/dev.sh up

dev-down: ## 停止本地测试页面
	@sh scripts/dev.sh down

dev-status: ## 查看本地测试服务状态
	@sh scripts/dev.sh status

dev-logs: ## 跟踪本地测试服务日志
	@sh scripts/dev.sh logs

api:
	$(GO) run ./cmd/server

api-up: ## 后台启动 API（日志写入 .dev/api.log）
	@sh scripts/dev.sh api-up

api-down: ## 停止 API
	@sh scripts/dev.sh api-down

api-restart: api-down ## 重启 API
	@sh scripts/dev.sh api-up

api-status: ## 查看 API 状态
	@sh scripts/dev.sh api-status

api-logs: ## 跟踪 API 日志
	@sh scripts/dev.sh api-logs

web:
	cd web && $(NPM) run dev

web-build:
	cd web && $(NPM) ci && $(NPM) run build
	rm -rf internal/webui/static
	cp -R web/dist internal/webui/static

build: web-build
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o build/mymoments ./cmd/server

test:
	$(GO) test -race ./...
	cd web && $(NPM) run test

fmt:
	gofmt -w ./cmd ./internal ./migrations

lint:
	$(GO) vet ./...
	cd web && $(NPM) run lint

check: fmt lint test

docker:
	docker build -t mymoments:latest .

clean:
	rm -rf build web/dist internal/webui/static
