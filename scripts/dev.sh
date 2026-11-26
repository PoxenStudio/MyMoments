#!/usr/bin/env bash
# 本地联调编排：Go API 与 Vite 开发服务器。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEV_DIR="$ROOT/.dev"
API_PORT=8080
WEB_PORT="${WEB_PORT:-5173}"
DATA_DIR="${MYMOMENTS_DATA_DIR:-$ROOT/data}"

log() { printf '[dev] %s\n' "$*"; }
fail() { printf '[dev] %s\n' "$*" >&2; exit 1; }

port_open() {
  nc -z -w 1 127.0.0.1 "$1" >/dev/null 2>&1
}

wait_http() {
  local url="$1" name="$2" attempts="${3:-90}"
  for ((i = 0; i < attempts; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  log "$name 未在 ${attempts}s 内就绪（${url}）"
  return 1
}

pid_alive() {
  local pidfile="$1"
  [[ -f "$pidfile" ]] && kill -0 "$(cat "$pidfile")" 2>/dev/null
}

start_api() {
  local pidfile="$DEV_DIR/api.pid"
  if pid_alive "$pidfile"; then
    log "API 已在运行（pid $(cat "$pidfile")）"
    return 0
  fi
  if port_open "$API_PORT"; then
    fail "端口 $API_PORT 已被占用，请先停止占用进程（可执行 make dev-status 查看）"
  fi
  log "启动 API（数据目录 ${DATA_DIR}）…"
  (
    cd "$ROOT"
    MYMOMENTS_DATA_DIR="$DATA_DIR" MYMOMENTS_DEV=1 exec go run ./cmd/server
  ) >"$DEV_DIR/api.log" 2>&1 &
  echo $! >"$pidfile"
  wait_http "http://127.0.0.1:$API_PORT/healthz" "API" 120 || {
    tail -n 40 "$DEV_DIR/api.log" >&2 || true
    fail "API 启动失败，请检查上方日志"
  }
}

start_web() {
  local pidfile="$DEV_DIR/web.pid"
  if pid_alive "$pidfile"; then
    log "Web 已在运行（pid $(cat "$pidfile")）"
    return 0
  fi
  if port_open "$WEB_PORT"; then
    fail "端口 $WEB_PORT 已被占用，请先停止占用进程（可执行 make dev-status 查看）"
  fi
  if [[ ! -d "$ROOT/web/node_modules" ]]; then
    log "安装 web 依赖…"
    (cd "$ROOT/web" && npm install --no-audit --no-fund)
  fi
  log "启动 Vite 开发服务器…"
  (
    cd "$ROOT/web"
    exec npm run dev -- --host 127.0.0.1 --port "$WEB_PORT" --strictPort
  ) >"$DEV_DIR/web.log" 2>&1 &
  echo $! >"$pidfile"
  wait_http "http://127.0.0.1:$WEB_PORT/" "Web" 90 || {
    tail -n 40 "$DEV_DIR/web.log" >&2 || true
    fail "Web 启动失败，请检查上方日志"
  }
}

stop_one() {
  local name="$1" pidfile="$DEV_DIR/$1.pid"
  [[ -f "$pidfile" ]] || return 0
  local pid
  pid="$(cat "$pidfile")"
  if kill -0 "$pid" 2>/dev/null; then
    pkill -TERM -P "$pid" 2>/dev/null || true
    kill -TERM "$pid" 2>/dev/null || true
    log "已停止 $name"
  fi
  rm -f "$pidfile"
}

cmd="${1:-up}"
case "$cmd" in
  up)
    mkdir -p "$DEV_DIR"
    start_api
    start_web
    log "本地测试页面就绪："
    log "  页面  http://127.0.0.1:$WEB_PORT"
    log "  API   http://127.0.0.1:$API_PORT"
    log "  日志  $DEV_DIR/api.log 与 $DEV_DIR/web.log"
    log "  首次访问会进入初始化引导，创建管理员账号后即可登录"
    ;;
  down)
    stop_one web
    stop_one api
    ;;
  status)
    for name in api web; do
      pidfile="$DEV_DIR/$name.pid"
      if pid_alive "$pidfile"; then
        log "$name 运行中（pid $(cat "$pidfile")）"
      else
        log "$name 未运行"
      fi
    done
    ;;
  api-up)
    mkdir -p "$DEV_DIR"
    start_api
    log "API 已就绪：http://127.0.0.1:$API_PORT"
    ;;
  api-down)
    stop_one api
    ;;
  api-status)
    if pid_alive "$DEV_DIR/api.pid"; then
      log "api 运行中（pid $(cat "$DEV_DIR/api.pid")）"
    else
      log "api 未运行"
    fi
    ;;
  api-logs)
    tail -n 50 -f "$DEV_DIR/api.log"
    ;;
  *)
    fail "用法：$0 {up|down|status|logs|api-up|api-down|api-status|api-logs}"
    ;;
esac
