#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
GOOS=js GOARCH=wasm go build -o web/web-computer.wasm ./cmd/web
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js
printf '\nBuilt web/web-computer.wasm. Serve with:\n  python3 -m http.server 8000 --directory web\n'
