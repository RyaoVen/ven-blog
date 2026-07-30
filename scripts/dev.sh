#!/usr/bin/env bash
# ven-blog 本地开发一键启动：Node SSR worker（后台）+ Go 网关（前台，日志跟随）
# 用法：bash scripts/dev.sh        停止：Ctrl+C（会一并停掉 Node worker）
set -euo pipefail

cd "$(dirname "$0")/.."

# 本地私有环境变量（gitignored；模板见 .env.local.example）
if [[ -f .env.local ]]; then
    set -a
    source .env.local
    set +a
fi

: "${BLOG_MYSQL_DSN:?未配置 BLOG_MYSQL_DSN——请 cp env.local.example .env.local 并填入本机 MySQL 密码}"

# Node worker 必须先起（Go 启动时拉取路由表）
echo "[dev] starting Node SSR worker on :3000 ..."
(cd frame/node && node dist/main.js) &
NODE_PID=$!
trap 'echo "[dev] stopping Node worker ..."; kill $NODE_PID 2>/dev/null || true' EXIT

echo "[dev] waiting for Node worker ..."
for _ in $(seq 1 30); do
    if curl -sf http://127.0.0.1:3000/pages -H "X-Ven-Internal-Token: ${VEN_INTERNAL_TOKEN:-development-token}" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

echo "[dev] starting Go gateway on :8080 ..."
cd frame/go && go run .
