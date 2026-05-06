#!/usr/bin/env bash
set -euo pipefail

log() { echo "==> $*"; }

log "go:   $(go version 2>/dev/null || echo 'NOT FOUND')"
log "pnpm: $(pnpm --version 2>/dev/null || echo 'NOT FOUND')"
log "node: $(node --version 2>/dev/null || echo 'NOT FOUND')"
log "templ: $("$(go env GOPATH)/bin/templ" version 2>/dev/null || echo 'NOT FOUND')"

log "Generating templ files"
"$(go env GOPATH)/bin/templ" generate --path=./web
log "Templ files generated: $(find web -name '*_templ.go' | wc -l | tr -d ' ')"

log "Installing web dependencies"
pnpm --dir web install

log "Building web assets"
pnpm --dir web build

log "Generating static site"
go run main.go generate

log "Done — output in out/"
