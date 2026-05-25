# API Redesign — Single Vercel Function Entrypoint

## Goal

Collapse the per-file `api/*.go` handlers into a single Vercel serverless
function. Eliminates the aggregate-binary-size problem from
[vercel-lambda-binary-size.md](./vercel-lambda-binary-size.md) by replacing
~25 statically-linked binaries with one.

Rationale: one ~162 MB binary is well under Vercel's deployment ceiling; the
issue was 25 of them at once. Keeping collect separate buys nothing once
there's only one binary.

## Target layout

```
api/
  index.go              ← only file; Vercel's single function entrypoint
pkg/
  api/
    handlers/           ← all Handle* funcs live here
      healthz.go
      ...
      digest/
        subscribe.go
        collect.go        ← imports pkg/source; fine, only one binary
        issues.go
      ...
      metrics/
        summary.go
        issues.go
        ...
      social/
        featured.go
        rotation.go
        metrics.go
      webhooks/
        resend.go
    mux/
      mux.go            ← wires every handler, including collect
    app.go, auth.go, ...  (unchanged)
```

Key invariant: nothing under `api/` except `index.go`. That's what Vercel's
`api/**/*.go` glob picks up as a function entrypoint.

As part of this refactor, we're also going to group common "digest" api routes such as collectig,
building and sending together under `/digest` (vercel.json will need updating.)

## api/index.go

```go
package api

import (
	"net/http"
	"sync"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api/mux"
)

var (
	handler     http.Handler
	handlerOnce sync.Once
)

func Handler(w http.ResponseWriter, r *http.Request) {
	handlerOnce.Do(func() {
		app := godaily.Bootstrap()
		handler = http.StripPrefix("/api", mux.Handler(app))
	})
	handler.ServeHTTP(w, r)
}
```

Fluid Compute reuses the instance across invocations, so `sync.Once`
bootstraps once per cold start.

## pkg/app.go

**No changes.** Leave `pkg/app.go` exactly as it is today, including the
`HasSources()` guard around `Materialise`. The previous P1 was caused by
moving this import around — there is no reason to take that risk again.

The `_ "pkg/source"` blank import stays in the collect handler file (now
`pkg/api/handlers/collect.go` instead of `api/collect.go`). The import chain
becomes:

```
api/index.go → pkg/api/mux → pkg/api/handlers (incl. collect.go) → pkg/source
```

`init()` still runs, sources still register, and `HasSources()` always
returns `true` in the consolidated binary. The guard becomes effectively a
no-op but remains as defence against a future refactor accidentally dropping
the import.

## vercel.json

```json
"rewrites": [
  { "source": "/healthz",  "destination": "/api/index" },
  { "source": "/api/(.*)", "destination": "/api/index" }
],
"crons": [
  { "path": "/api/collect",         "schedule": "0 1 * * *" },
  { "path": "/api/build",           "schedule": "0 2 * * *" },
  ...
],
"functions": {
  "api/index.go": {
    "runtime": "@vercel/go@3.6.0",
    "maxDuration": 300
  }
}
```

Crons still hit their public paths; the rewrite funnels them into `index`.

Keep the existing `/api/issues/:slug → /api/issues/slug?slug=:slug` rewrites
(or switch the mux to chi with `{slug}` patterns) — either works. Current
`http.ServeMux` setup uses query params.

## Work order

1. Move every `Handle*` from `api/**/*.go` into `pkg/api/handlers/` (mirror
   the existing subdirs: `metrics/`, `social/`, `webhooks/`, `issues/`,
   `items/`). Tests move with the handlers.
2. Update `pkg/api/mux/mux.go` import path from
   `apihandlers "github.com/ainsleyclark/godaily/api"` to
   `apihandlers "github.com/ainsleyclark/godaily/pkg/api/handlers"` (and the
   nested package imports accordingly).
3. Delete every `api/*.go` and `api/*/` directory except the new `index.go`.
4. Leave `pkg/app.go` untouched. Keep `_ "pkg/source"` in the relocated
   `pkg/api/handlers/collect.go` so the registry still populates.
5. Update `vercel.json`:
   - rewrites: catch-all `/api/(.*) → /api/index`
   - `functions` glob narrowed to `api/index.go`
6. `go test ./...`
7. `golangci-lint run ./... --fix --config=.golangci.yaml`

## Build tags

The `GOFLAGS=-tags=serverless` Vercel env var stays — it's unrelated to this
consolidation. SQLite is still dead weight in the Lambda, so the
`//go:build !serverless` gate on `pkg/db/sqlite.go` continues to earn its
keep.

## Verification

After the move, the single binary should still build clean and include
lingua-go (now expected, not avoided):

```bash
go build -tags=serverless -o /tmp/index ./api/index.go
ls -lh /tmp/index   # expect ~160 MB
go list -deps -tags=serverless ./api/... | grep lingua  # expect a hit
```

One binary at ~160 MB vs the previous 25 × 19 MB + 1 × 162 MB ≈ 637 MB —
roughly a 4× reduction in total artifact weight and a dramatic cut in the
parallel packaging work Vercel does during "Deploying outputs".
