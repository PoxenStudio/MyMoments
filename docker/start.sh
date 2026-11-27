#!/bin/sh
set -eu

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${PATH}"

DATA_DIR="${MYMOMENTS_DATA_DIR:-/data}"
echo "[MyMoments] starting, data dir: ${DATA_DIR}"

# bind mount 进来的目录属主常与容器内 app 不一致，统一修正；用标记文件避免每次启动都递归 chown
mkdir -p "$DATA_DIR"
APP_UID="$(id -u app)"
APP_GID="$(id -g app)"
MARKER="$DATA_DIR/.permission"
if [ ! -f "$MARKER" ] || [ "$(cat "$MARKER" 2>/dev/null)" != "${APP_UID}:${APP_GID}" ]; then
  echo "[MyMoments] adjusting ownership of ${DATA_DIR} to app (${APP_UID}:${APP_GID})"
  if ! chown -R app:app "$DATA_DIR"; then
    echo "[MyMoments] error: cannot change ownership of ${DATA_DIR} (read-only mount?)" >&2
    exit 1
  fi
  printf '%s' "${APP_UID}:${APP_GID}" > "$MARKER"
  chown app:app "$MARKER"
fi

# 以 app 身份做写检查：只读挂载、磁盘满或权限错配时立即失败，而不是等运行时才报错
if ! su -s /bin/sh app -c "touch '${DATA_DIR}/.write_test'" 2>/dev/null; then
  echo "[MyMoments] error: ${DATA_DIR} is not writable by app" >&2
  exit 1
fi
rm -f "${DATA_DIR}/.write_test"

# 启用内置 HTTPS 时提前确认证书对 app 可读，避免 supervisord 反复重启才暴露问题
if [ -n "${MYMOMENTS_TLS_CERT:-}" ]; then
  for f in "${MYMOMENTS_TLS_CERT}" "${MYMOMENTS_TLS_KEY:-}"; do
    if [ -z "$f" ] || ! su -s /bin/sh app -c "test -r '${f}'" 2>/dev/null; then
      echo "[MyMoments] error: TLS file is missing or not readable by app: ${f:-<empty>}" >&2
      exit 1
    fi
  done
fi

# 允许用 docker run <image> <cmd> 临时覆盖入口完成调试
if [ "$#" -gt 0 ]; then
  exec "$@"
fi

echo "[MyMoments] prepared, handing over to supervisord"
exec /usr/bin/supervisord -c /etc/supervisord.conf
