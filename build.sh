#!/usr/bin/env bash
set -euo pipefail

OUT="${1:-dnskeeper}"
ROOT="$(cd "$(dirname "$0")" && pwd)"

# 构建前端到 web/dist(go:embed 依赖)
if [ -d "$ROOT/web/node_modules" ]; then
  echo "building frontend..."
  (cd "$ROOT/web" && npm run build)
elif command -v npm >/dev/null 2>&1 && [ -f "$ROOT/web/package-lock.json" ]; then
  echo "installing + building frontend..."
  (cd "$ROOT/web" && npm ci && npm run build)
else
  echo "warn: npm 不可用,跳过前端构建(将嵌入占位,前端不可用)"
fi

go build -trimpath -ldflags="-s -w" -o "$OUT" ./cmd/dnskeeper

echo "built: $OUT"
