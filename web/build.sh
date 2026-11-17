#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
GOOS=js GOARCH=wasm go build -o web/web-computer.wasm ./cmd/web
GOOS=js GOARCH=wasm go build -o web/syntax-check.wasm ./cmd/syntaxcheck
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js
mkdir -p web/examples
cp scripts/python/*.py web/examples/
printf '\nBuilt web/web-computer.wasm and web/syntax-check.wasm. Serve with:\n  python3 -m http.server 8000 --directory web\n'
