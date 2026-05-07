#!/usr/bin/env bash
set -euo pipefail

log() { echo "==> $*"; }

if ! command -v go &> /dev/null; then
	log "Go not found — installing go1.26.2"
	curl -sSfL https://go.dev/dl/go1.26.2.linux-amd64.tar.gz | tar -xz -C /tmp/
	export PATH="/tmp/go/bin:$PATH"
fi

log "go:   $(go version 2>/dev/null || echo 'NOT FOUND')"
log "pnpm: $(pnpm --version 2>/dev/null || echo 'NOT FOUND')"
log "node: $(node --version 2>/dev/null || echo 'NOT FOUND')"

log "Installing web dependencies"
cd web
pnpm install

log "Building web assets"
pnpm run build

log "Generating static site"
cd ../
go run main.go generate

log "Done — output in out/"
