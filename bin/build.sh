#!/usr/bin/env bash
set -euo pipefail

log() { echo "==> $*"; }

log "go:   $(go version 2>/dev/null || echo 'NOT FOUND')"
log "pnpm: $(pnpm --version 2>/dev/null || echo 'NOT FOUND')"
log "node: $(node --version 2>/dev/null || echo 'NOT FOUND')"

log "Installing pnpm"
npm install -g pnpm

log "Installing web dependencies"
pnpm --dir web install

log "Building web assets"
pnpm --dir web build

log "Generating static site"
go run main.go generate

log "Done — output in out/"
