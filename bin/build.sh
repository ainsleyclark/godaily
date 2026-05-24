#!/usr/bin/env bash
set -euo pipefail

log() { echo "==> $*"; }

if ! command -v go &> /dev/null; then
	log "Go not found — installing go1.26.3"
	curl -sSfL https://go.dev/dl/go1.26.3.linux-amd64.tar.gz | tar -xz -C /tmp/
	export PATH="/tmp/go/bin:$PATH"
fi

# Compile one package at a time and trigger GC more aggressively to stay
# inside Vercel's build-machine memory ceiling.
export GOGC=20
export GOFLAGS="-p=1"

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
go build -trimpath -ldflags="-s -w" -o /tmp/godaily_gen main.go
APP_ENV=production /tmp/godaily_gen generate

log "Done — output in out/"
