# Stage 1: Database + migrations

## Context

`godaily` is currently a Go CLI that fans out a daily digest to a single
hardcoded `EMAIL_SEND_ADDRESS`. There is no persistence: issues are sent
and forgotten, news items are recomputed each run, and "subscribers" is
one env var. We want:

- A real subscribers list with public signup and unsubscribe
- An archive of past issues
- Per-news-item pages (deep links, future SEO)
- A public website fronting all of the above

The work is delivered in three stages. This is Stage 1.

- Stage 1: Database + migrations (this file)
- Stage 2: API ([stage-2-api.md](stage-2-api.md))
- Stage 3: Web ([stage-3-web.md](stage-3-web.md))

## Recommendation: pure Go (rejected SvelteKit)

Given a free-hosting constraint that is not hard, plus public signups
and per-item pages, Go + templ wins over SvelteKit-on-Vercel because:

- godaily is already Go, so one language, one repo, one deploy
- sqlc + templ give type safety end to end with no API contract drift
- The cron writer (already Go on GH Actions) and the website share the
  same query package: write a query once, both sides reuse it
- Turso embedded replicas in Go give near-zero-latency reads
- Public signup is a simple POST handler, no second deploy or
  cross-origin auth boundary
- Hosting is trivially cheap on Fly.io free hobby tier

## Goal

Schema in place, queries generated, cron persists each run, no HTTP
surface yet. Self-contained and shippable.

## Schema (3 tables)

Please bear in mind we shouldn't store any social share text, this is purely
for issues. I use the social share text for me only.

```sql
CREATE TABLE issues (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  slug            TEXT NOT NULL UNIQUE,        -- e.g. 2026-04-27
  sent_at         TIMESTAMP NOT NULL,
  subject         TEXT NOT NULL,
  summary         TEXT,                        -- one-liner from synth
  html_body       TEXT NOT NULL,
  text_body       TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'sent' -- sent|skipped|failed
);

CREATE TABLE items (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id  INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
  source    TEXT NOT NULL,                     -- hn|reddit|lobsters|...
  title     TEXT NOT NULL,
  url       TEXT NOT NULL,
  author    TEXT,
  score     INTEGER,
  summary   TEXT,
  position  INTEGER NOT NULL,                  -- order within issue
);
CREATE INDEX idx_items_issue ON items(issue_id);

CREATE TABLE subscribers (
  id                INTEGER PRIMARY KEY AUTOINCREMENT,
  email             TEXT NOT NULL UNIQUE,
  confirm_token     TEXT NOT NULL UNIQUE,
  unsubscribe_token TEXT NOT NULL UNIQUE,
  confirmed_at      TIMESTAMP,
  unsubscribed_at   TIMESTAMP,
  created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_subscribers_active
  ON subscribers(confirmed_at) WHERE unsubscribed_at IS NULL;
```

`unsubscribe_token` is a stable, per-row secret embedded in every email
so any digest can carry a one-click unsubscribe link without auth.

## Tooling

| Concern    | Choice                                       |
|------------|----------------------------------------------|
| Driver     | `github.com/tursodatabase/libsql-client-go`  |
| Queries    | `sqlc` (sqlite engine)                       |
| Migrations | `pressly/goose`, embedded via `embed.FS`     |

## Package layout

Two packages, separate concerns:

- `internal/db/` owns the `*sql.DB` lifecycle and schema (migrations).
  Anything in here is about *getting a connection* and *evolving the
  schema*, not about querying.
- `internal/store/` owns typed query access. One file per domain, plus
  the sqlc-generated companion. Hand-written `*.go` only appears when
  there is logic beyond a single query (multi-step tx, token
  generation, validation).

```
internal/
├── db/
│   ├── db.go                 # New(ctx, url, token) (*sql.DB, error)
│   ├── migrate.go            # Migrate(ctx, *sql.DB) error (embedded goose)
│   └── migrations/
│       └── 0001_init.sql
└── store/
    ├── store.go              # New(*sql.DB) *Store, shared helpers, tx wrapper
    ├── issues.sql            # sqlc input
    ├── issues.sql.go         # sqlc output
    ├── issues.go             # hand-rolled domain logic (only if needed)
    ├── items.sql / .sql.go / .go
    └── subscribers.sql / .sql.go / .go
```

## Files

To create:

- `internal/db/db.go`: `New(ctx, url, token) (*sql.DB, error)` using `enforce` per AGENTS.md
- `internal/db/migrate.go`: `Migrate(ctx, *sql.DB) error`, goose with embedded migrations
- `internal/db/migrations/0001_init.sql`: schema above
- `internal/store/store.go`: `New(*sql.DB) *Store`, shared tx helper
- `internal/store/issues.sql`, `items.sql`, `subscribers.sql`: sqlc query files
- `internal/store/*.sql.go`: sqlc output (per-domain, generated)
- `sqlc.yaml`: configures per-file output into `internal/store`

To modify:

- `cmd/godaily/main.go`: add `migrate` subcommand, thread `*sql.DB` into `run`
- `internal/cron/run.go`: at the end of a successful run, insert one
  `issues` row plus all `items`. Keep `EMAIL_SEND_ADDRESS` as the
  current single recipient; subscriber-driven fan-out is Stage 2.
- `Makefile`: `make sqlc`, `make migrate-up`, `make migrate-down`
- `.env.example`: `TURSO_URL`, `TURSO_AUTH_TOKEN`
- `AGENTS.md`: DB layout, migration workflow

## Verification

1. `turso db create godaily-dev` (or `turso dev` for local libsql)
2. `go run ./cmd/godaily migrate`, confirm via `turso db shell`
3. `make run-dry` then `make run`, confirm `issues` and `items` rows
4. `make all` clean, including new unit tests in `internal/db` against
   ephemeral file-backed SQLite
