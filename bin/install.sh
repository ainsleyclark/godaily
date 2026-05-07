#!/usr/bin/env bash
set -euo pipefail

log() { echo "==> $*"; }

if ! command -v go &> /dev/null; then
	log "Go not found — installing go1.26.2"
	curl -sSfL https://go.dev/dl/go1.26.2.linux-amd64.tar.gz | tar -xz -C /tmp/
	export PATH="/tmp/go/bin:$PATH"
fi

log "go:   $(go version)"
log "pnpm: $(pnpm --version)"
log "node: $(node --version)"

log "Installing web dependencies"
cd web
pnpm install
