#!/usr/bin/env bash
# 校验 go:embed 的前端产物与 index.html 引用是否一致，缺失时重新构建。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATIC="$ROOT/internal/webui/static"

log() { printf '[webui] %s\n' "$*"; }

if [[ ! -f "$STATIC/index.html" ]]; then
  log "缺少 $STATIC/index.html，先执行 make web-build"
  exit 1
fi

missing=""
refs=$(grep -oE '(src|href)="/(assets|icons|brand|manifest\.webmanifest|favicon\.ico|registerSW\.js)[^"]*"' "$STATIC/index.html" \
  | sed -E 's/.*="\/([^"]*)"/\1/' | sort -u)
for ref in $refs; do
  [[ -e "$STATIC/$ref" ]] || missing="$missing $ref"
done

if [[ -z "$missing" ]]; then
  exit 0
fi

log "内嵌资源与 index.html 不一致，缺失:$missing"
log "重新构建前端并覆盖 static…"

cd "$ROOT/web"
if [[ ! -d node_modules ]]; then
  npm ci --no-audit --no-fund
fi
npm run build
rm -rf "$STATIC"
cp -R "$ROOT/web/dist" "$STATIC"
log "已更新内嵌资源，如 API 正在运行请重启使其重新嵌入"
