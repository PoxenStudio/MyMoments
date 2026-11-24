.PHONY: dev api web web-build build test fmt lint check docker clean

GO ?= go
NPM ?= npm

dev:
	@echo "Run 'make api' and 'make web' in two terminals."

api:
	$(GO) run ./cmd/server

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
	rm -rf build web/dist internal/webui/static/assets
