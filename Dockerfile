FROM docker.1ms.run/node:22-alpine AS web
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM docker.1ms.run/golang:1.23-alpine AS build
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./internal/webui/static
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM docker.1ms.run/alpine:3.20
RUN addgroup -S app && adduser -S -G app app \
    && apk add --no-cache ca-certificates tzdata wget supervisor \
    && mkdir -p /data && chown app:app /data
COPY --from=build /out/server /usr/local/bin/mymoments
COPY conf/supervisor/supervisord.conf /etc/supervisord.conf
COPY conf/supervisor/mymoments.conf /etc/supervisor/conf.d/
ENV MYMOMENTS_DATA_DIR=/data MYMOMENTS_ADDR=:8080
VOLUME ["/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
  CMD wget -qO- http://127.0.0.1:8080/readyz >/dev/null 2>&1 || wget --no-check-certificate -qO- https://127.0.0.1:8080/readyz >/dev/null || exit 1
ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
