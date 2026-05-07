#!/usr/bin/env bash
set -euo pipefail

log() { echo "==> $*"; }

log "go:   $(go version 2>/dev/null || echo 'NOT FOUND')"
log "pnpm: $(pnpm --version 2>/dev/null || echo 'NOT FOUND')"
log "node: $(node --version 2>/dev/null || echo 'NOT FOUND')"

log "Building web assets"
cd web
pnpm run build

log "Generating static site"
cd ../
go run main.go generate

log "Done — output in out/"
