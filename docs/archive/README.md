# Browse Archive — Build Plan

A public, filterable archive page showing **every news item the pipeline has
collected**, queried directly off the `items` table. Not grouped by issue —
it's the raw, deduped universe of stories. Items that made it into a sent
digest get a small visual marker.

Route: `GET /browse/`

## Why this is mostly a query + one-page job

The data already exists. The collection pipeline persists every fetched item
to `items` with `issue_id = nil` (`pkg/services/digest/collect.go:101`). The
`UNIQUE (url, tag)` constraint dedups across runs. The only place a cap is
applied is **email render time** (`pkg/services/digest/email.go:151-160`,
`news.SectionLimits`) — storage is uncapped. So "show everything" means
*query items directly and don't apply `SectionLimits`*.

### The funnel (store → link → send)

| Stage | File | What it does | Cap |
|-------|------|--------------|-----|
| Store (Collect) | `collect.go` | Persists all fetched, date-valid items, `issue_id = nil` | none (date window only) |
| Link (Build) | `build.go` | Flips window items' `issue_id` to the new issue via `ON CONFLICT DO UPDATE` | none |
| Send (Email) | `email.go` | Renders top-N per section | `news.SectionLimits` |

### The digest marker

`items.issue_id` is nullable. `issue_id IS NOT NULL` ⇒ the item appeared in a
built/sent digest. That boolean is the entire signal behind the "made the
digest" badge. We surface it as a `bool` on the domain model and a chip in the
UI. No join to `issues` is required for the marker itself (though we can
optionally fetch the issue slug to link the badge to the issue page — see
plan 03).

## Scope notes (read before starting)

- **Comments / "Discussed" sort: OUT OF SCOPE.** `items` has no `comments`
  column and the collect path never writes one. Skip the Discussed tab and any
  comment counts for now.
- **Free-form hashtag tags (`#go1.25 #release`) in the mockup: OUT OF SCOPE.**
  The model has a single section `Tag`, not multiple free tags. Treat the chips
  in the design as the section tag only, unless we later add a tags table.
- **Stat cards needing collection time** ("+47 today", "pulled in last hour"):
  the `items` table has **no `created_at`** — `published` is the article's date,
  not when we pulled it. These specific stats need a new `created_at` column
  (migration). See plan 01 (optional) and designer-notes.md. "Stories indexed"
  (total count) and "next digest in" (schedule-derived) need no schema change.

## Plans & dependency order

1. **01-query-layer.md** — extend `ItemListOptions`, add sqlc queries + store
   methods (filter/sort/paginate/count/source-counts), surface `InDigest` on
   the model, regenerate sqlc + mocks. *Foundation — do first.*
2. **02-web-handler.md** — `GET /browse/` handler: parse querystring → options
   → render. Depends on 01.
3. **03-page-and-components.md** — Templ page + components (item row with digest
   marker, tabs, source sidebar, sort bar, pagination). Depends on 02's view
   model contract.
4. **04-styling.md** — SCSS for the 3-column layout, stat cards, chips.
   Depends on 03's markup/class names.
5. **05-progressive-enhancement.md** — optional JS (live filtering, infinite
   scroll, ⌘K search). SSR works without it. Depends on 03/04.

Plans 01→02→03 are sequential. 04 can start once 03's class names are agreed.
05 is last and optional.

## Conventions (all plans)

- Branch: `claude/bold-gates-XSqwS`.
- After editing any `.sql`: run `sqlc generate`.
- After changing a mocked interface: `go generate ./pkg/domain/... ./pkg/services/...`.
- Before committing: `go test ./...` and
  `golangci-lint run ./... --fix --config=.golangci.yaml`.
- Conventional commits: `feat: Add browse archive query layer`, etc.
- `pkg/store/internal/sqlc/` is generated — never hand-edit.
