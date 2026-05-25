# Vercel Lambda Binary Size — Findings & Scaling Constraints

## Problem summary

After PR #152 (social rotation), Vercel deployments began failing at the
**"Deploying outputs"** phase — after Go compilation completed successfully
(~2 min) but before any function started. The root cause was an OOM kill
triggered by the total size of all Lambda binaries Vercel must load into
memory simultaneously during packaging/upload.

## Root cause

`pkg/app.go` contained a blank import:

```go
_ "github.com/ainsleyclark/godaily/pkg/source"
```

This import is required only by `/api/collect` — it registers news-source
fetchers via `init()`. However, because `Bootstrap()` lives in `pkg/app.go`
and every handler calls `Bootstrap`, the import was compiled into every
Lambda binary.

`pkg/source` transitively imports `github.com/pemistahl/lingua-go`, a
language-detection library that **embeds ~140 MB of model data** directly
into the binary. PR #152 added one more handler (rotation), crossing the
threshold where the aggregate binary size exceeded Vercel's deployment memory
budget.

### Binary size measurements (linux/amd64)

| Configuration | Binary size | 25 handlers total |
|---|---|---|
| Before fix (baseline) | ~162 MB | ~4,050 MB |
| SQLite tag only (`-tags=serverless`) | ~157 MB | ~3,925 MB |
| SQLite tag + no `pkg/source` in non-collect | **~19 MB** | **~475 MB** |

The `modernc.org/sqlite` driver (`//go:build !serverless`) contributed only
~5 MB per binary — it was not the primary driver.

## Fix applied

1. **`api/collect.go`** — moved `_ "pkg/source"` here. Lingua-go is now
   compiled into only the collect Lambda.
2. **`pkg/app.go`** — removed the import; wrapped `Materialise` with
   `news.HasSources()` so `Bootstrap` skips source materialisation on
   handlers where no sources are registered.
3. **`pkg/domain/news/registry.go`** — added `HasSources()` which returns
   `false` when `pkg/source`'s `init()` functions have not run.
4. **`pkg/db/sqlite.go`** — new file gated with `//go:build !serverless`;
   keeps the SQLite driver available for local dev and tests but excluded
   from Vercel Lambda builds.
5. **Vercel dashboard** — add `GOFLAGS=-tags=serverless` as a build
   environment variable so `@vercel/go` picks up the build tag during
   Lambda compilation.
6. **`bin/build.sh`** — added `GOGC=20` and `GOFLAGS=-p=1` to cap memory
   growth during the static-site generation step (separate from Lambda
   compilation).

## Scaling constraints to watch

### Handler count

Each `.go` file under `api/` becomes a separate Lambda binary. As of this
fix, 24 handlers × ~19 MB = ~456 MB and 1 collect handler × ~162 MB ≈
**~618 MB total** — well within budget. The safe ceiling before revisiting
this is roughly 50 non-collect handlers (50 × 19 MB = 950 MB), but monitor
this as dependencies grow.

### Adding new heavy dependencies to `pkg`

Any import added to a package in the `pkg/` transitive closure of `pkg/app.go`
affects **every Lambda binary**. Before adding a new dependency, check its
binary size contribution:

```bash
# Build a single handler with and without the new import and compare sizes
go build -tags=serverless -o /tmp/before ./api/healthz.go
# add the import, then:
go build -tags=serverless -o /tmp/after ./api/healthz.go
ls -lh /tmp/before /tmp/after
```

If the delta is > 5 MB, consider whether it belongs behind a build tag or
moved to only the handler(s) that need it.

### lingua-go / collect binary size

The collect binary (~162 MB) is unlikely to grow further unless `pkg/source`
adds a second ML library. If it does, the collect binary may need to be moved
to its own Vercel project or replaced with an external job runner (GitHub
Actions, Railway, Fly.io) that is not subject to Vercel's deployment memory
ceiling.

### Vercel's deployment memory budget

Vercel does not publish an official limit for the "Deploying outputs" phase.
Empirically, ~4 GB triggered the OOM. If the total artifact size approaches
~2 GB, run a trial deployment and monitor the build logs for signs of memory
pressure (slow upload, unexpected restarts).

### `pkg/source` registration pattern

The blank-import / `init()` pattern means the source registry is a global
mutable map. If a future refactor introduces a second entry point that also
needs sources (e.g. a preview handler), it must also blank-import `pkg/source`
explicitly. There is no compile-time check that enforces this — the runtime
error from `a.Runner.Collect` returning zero items is the only signal.

A more robust long-term design would be to pass sources explicitly through
`Bootstrap` or `App`, making the dependency visible in the type system rather
than via side-effectful imports.

## Practical recommendations

Vercel support (Thaer Bashir) confirmed the failure mode: each `api/*.go`
file is compiled as its own Lambda that statically links the full Go
dependency graph, and the CLI packages and uploads those binaries in
parallel while also uploading `out/` — peaking past the 8 GB ceiling on
the standard build machine. They suggested three remedies, ordered here
by effort vs. durability:

1. **Enable Enhanced Build Machines (immediate unblock).** Project
   Settings → Builds → enable the larger build machine. Available on
   our Pro Plus plan without a plan change. Use this first to confirm
   the diagnosis and clear any in-flight failing deploy. If a build
   still fails on the larger runner, capture the deployment URL and
   reply on the support thread.
2. **Consolidate `api/*.go` behind a single chi router (durable fix).**
   We already depend on `go-chi`. Collapse the per-file handlers into
   one entrypoint (e.g. `api/index.go`) that mounts every route on a
   single `chi.Router`, with `vercel.json` rewriting `/api/*` to it.
   This collapses ~15 statically-linked binaries into one, dramatically
   cutting both the parallel packaging work and the total artifact
   weight. Keep `api/collect.go` separate so lingua-go stays out of the
   shared binary (see "lingua-go / collect binary size" above).
3. **Trim heavy unused dependencies (parallel cleanup).** Audit
   `go.mod` for libraries that are not on a production Lambda path —
   Vercel flagged the `charmbracelet/*` packages as likely dev-only.
   Move dev tooling behind build tags or into `tools/` with a separate
   module, and verify with `go mod why <pkg>` before removing. Track
   the result with the binary-size check in the "Adding new heavy
   dependencies" section.

### Recommended order

Enable Enhanced Build Machines now to unblock deploys, then schedule
the chi consolidation as the durable fix, and run the dependency audit
alongside it. Once consolidated, re-measure the totals in the binary-size
table above and update the "safe ceiling" estimate.
