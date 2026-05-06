#!/usr/bin/env bash
set -euo pipefail

log() { echo "==> $*"; }

log "go:   $(go version 2>/dev/null || echo 'NOT FOUND')"
log "pnpm: $(pnpm --version 2>/dev/null || echo 'NOT FOUND')"
log "node: $(node --version 2>/dev/null || echo 'NOT FOUND')"

log "Installing web dependencies"
npm --dir web install

log "Building web assets"
npm --dir web build

log "Generating static site"
go run main.go generate

log "Done — output in out/"
