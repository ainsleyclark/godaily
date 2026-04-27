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

```sql
CREATE TABLE issues (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  slug            TEXT NOT NULL UNIQUE,        -- e.g. 2026-04-27
  sent_at         TIMESTAMP NOT NULL,
  subject         TEXT NOT NULL,
  summary         TEXT,                        -- one-liner from synth
  html_body       TEXT NOT NULL,
  text_body       TEXT NOT NULL,
  social_x        TEXT,
  social_linkedin TEXT,
  status          TEXT NOT NULL DEFAULT 'sent' -- sent|skipped|failed
);

CREATE TABLE news_items (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id  INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
  source    TEXT NOT NULL,                     -- hn|reddit|lobsters|...
  title     TEXT NOT NULL,
  url       TEXT NOT NULL,
  author    TEXT,
  score     INTEGER,
  summary   TEXT,
  position  INTEGER NOT NULL,                  -- order within issue
  raw_json  TEXT                               -- original item for debugging
);
CREATE INDEX idx_news_items_issue ON news_items(issue_id);

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

## Files

To create:

- `internal/db/db.go`: `New(ctx, url, token) (*sql.DB, error)` constructor using `enforce` per AGENTS.md
- `internal/db/migrations/0001_init.sql`: schema above
- `internal/db/queries/issues.sql`, `news_items.sql`, `subscribers.sql`
- `internal/db/gen/`: sqlc output (`*.gen.go`)
- `sqlc.yaml`

To modify:

- `cmd/godaily/main.go`: add `migrate` subcommand, thread `*sql.DB` into `run`
- `internal/cron/run.go`: at the end of a successful run, insert one
  `issues` row plus all `news_items`. Keep `EMAIL_SEND_ADDRESS` as the
  current single recipient; subscriber-driven fan-out is Stage 2.
- `Makefile`: `make sqlc`, `make migrate-up`, `make migrate-down`
- `.env.example`: `TURSO_URL`, `TURSO_AUTH_TOKEN`
- `AGENTS.md`: DB layout, migration workflow

## Verification

1. `turso db create godaily-dev` (or `turso dev` for local libsql)
2. `go run ./cmd/godaily migrate`, confirm via `turso db shell`
3. `make run-dry` then `make run`, confirm `issues` and `news_items` rows
4. `make all` clean, including new unit tests in `internal/db` against
   ephemeral file-backed SQLite
