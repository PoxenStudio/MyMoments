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
# 打包时传入当前分支名作为版本号，未传参时使用默认版本
ARG GIT_BRANCH=v0.1.0
RUN addgroup -S app && adduser -S -G app app \
    && apk add --no-cache ca-certificates tzdata wget supervisor \
    && mkdir -p /data && chown app:app /data \
    && echo "${GIT_BRANCH}" > /version.txt && chmod 644 /version.txt
COPY --from=build /out/server /usr/local/bin/mymoments
COPY --chmod=755 docker/start.sh /usr/local/bin/start.sh
COPY conf/supervisor/supervisord.conf /etc/supervisord.conf
COPY conf/supervisor/mymoments.conf /etc/supervisor/conf.d/
ENV MYMOMENTS_DATA_DIR=/data \
    MYMOMENTS_ADDR=:8080 \
    MYMOMENTS_DEFAULT_TZ=Asia/Shanghai \
    MYMOMENTS_WEATHER_PROVIDER=open-meteo \
    MYMOMENTS_HOUSEHOLD_QUOTA=1073741824 \
    MYMOMENTS_TRUST_PROXY=false \
    PUID=1000 \
    PGID=1000 \
    TZ=Asia/Shanghai
VOLUME ["/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
  CMD if [ -n "$MYMOMENTS_TLS_CERT" ]; then wget --no-check-certificate -qO- https://127.0.0.1:8080/readyz >/dev/null || exit 1; else wget -qO- http://127.0.0.1:8080/readyz >/dev/null || exit 1; fi
ENTRYPOINT ["/usr/local/bin/start.sh"]
