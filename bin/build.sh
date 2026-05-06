#!/usr/bin/env bash
set -euo pipefail

go tool templ generate --path=./web
pnpm --dir web install
pnpm --dir web build
go run main.go generate
