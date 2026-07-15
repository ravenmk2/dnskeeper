#!/usr/bin/env bash
set -euo pipefail

OUT="${1:-dnskeeper}"

go build -trimpath -ldflags="-s -w" -o "$OUT" ./cmd/dnskeeper

echo "built: $OUT"
