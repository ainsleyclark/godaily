# Stage 3: Web

## Context

Builds on Stage 1 ([stage-1-database.md](stage-1-database.md)) and
Stage 2 ([stage-2-api.md](stage-2-api.md)). The DB and cron persistence
are in place, the subscriber API exists, and the cron fans out to real
subscribers. This stage adds the public website: archive, per-issue,
per-item, and a signup form.

- Stage 1: Database + migrations ([stage-1-database.md](stage-1-database.md))
- Stage 2: API ([stage-2-api.md](stage-2-api.md))
- Stage 3: Web (this file)

## Goal

A `cmd/web` binary serving SSR pages backed by the same DB and (where
useful) the Stage 2 API.

## Tech

- Templates: `a-h/templ`
- HTTP: `github.com/ainsleydev/webkit/pkg/webkit`
- Static assets via `embed.FS`
- Same `Handler func(c *webkit.Context) error` signature as the API

## Routes

| Method | Path             | Purpose                                  |
|--------|------------------|------------------------------------------|
| GET    | `/`              | Latest issue summary + recent archive    |
| GET    | `/issues`        | Paginated archive                        |
| GET    | `/issues/{slug}` | Past issue page                          |
| GET    | `/items/{id}`    | Per-item page (SEO)                      |
| POST   | `/subscribe`     | HTML form action, posts to Stage 2 API   |
| GET    | `/confirm`       | Friendly confirmation page (calls API)   |
| GET    | `/unsubscribe`   | Friendly unsubscribe page (calls API)    |
| GET    | `/healthz`       | Liveness                                 |

The web binary can either call the API over HTTP (clean separation) or
import `internal/db` and `internal/subscribers` directly (less moving
parts in dev). Default to direct package imports; the API stays useful
for external consumers and for keeping the seam testable.

## Files

To create:

- `cmd/web/main.go`
- `internal/web/handler.go`
- `internal/web/views/*.templ`: `layout`, `home`, `issue`, `item`, `archive`, `subscribe_*`
- `internal/web/static/`: minimal CSS, embedded

To modify:

- `Makefile`: `make web`, `make templ` (regen)
- `AGENTS.md`: document `cmd/web`, templ workflow, deploy notes

## Verification

1. `make web`, visit `/`, archive renders from real Stage 1 data
2. Submit subscribe form, end-to-end confirm and unsubscribe via the UI
3. `/issues/{slug}` and `/items/{id}` render with no JS required
4. Lighthouse pass on a representative issue page (SEO basics)
5. `make all` clean

## Hosting (applies after all stages)

- Turso for the DB (free tier)
- Fly.io app for `cmd/api` and `cmd/web` (single-binary deploys, free hobby)
- GH Actions cron unchanged in shape, gets `TURSO_URL` and
  `TURSO_AUTH_TOKEN` secrets so its runtime writes land in Turso
- Migrations run on startup of `cmd/api` and `cmd/web`, and at the
  start of each cron run (idempotent via goose)
