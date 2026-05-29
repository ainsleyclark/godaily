# Plan 01 — Query Layer

**Goal:** make it possible to query `items` directly for the Browse page —
filtered by source/section, sorted, paginated, with total + per-source counts,
and with a `InDigest` flag derived from `issue_id`.

**Depends on:** nothing. Do this first.

## Background

Today the store rejects an unfiltered list:

```go
// pkg/store/items/store.go:49
func (s Store) List(ctx, opts) {
    if opts.IssueID != nil { ... }
    if opts.From != nil && opts.To != nil { ... }
    return nil, fmt.Errorf("...at least one filter...must be set")
}
```

`ItemListOptions` is only `{IssueID, From, To}` (`pkg/domain/news/item.go:31`).
`items` columns: `id, issue_id, source, title, url, tag, author_*, score,
summary, position, original_url, published` (no `created_at`, no `comments`).

## Changes

### 1. Domain model (`pkg/domain/news/item.go`)

Add an `InDigest bool` field to `Item` (set from `issue_id IS NOT NULL`).

Extend `ItemListOptions`:

```go
type ItemListOptions struct {
    IssueID *int64
    From    *time.Time
    To      *time.Time
    Sources   []Source   // OR-match across sources
    Tags      []Tag      // section tags (use Tag.Section() semantics)
    Search    string     // LIKE over title + summary
    Sort      ItemSort   // New | Top | Hot ; default New
    InDigest  *bool      // nil = all; true = only digested; false = only raw
    Page      int64      // 1-based; 0 = no pagination
    PerPage   int64      // 0 = default (e.g. 30)
}

type ItemSort string
const (
    ItemSortNew ItemSort = "new" // published DESC
    ItemSortTop ItemSort = "top" // score DESC
    ItemSortHot ItemSort = "hot" // score with recency decay
)
```

> Reuse the `Page/PerPage` semantics from `pkg/store/store.go:25`
> (`ListOptions`) so pagination behaves consistently across the app.

### 2. sqlc queries (`pkg/store/items/query.sql`)

SQLite + `database/sql` (see `sqlc.yaml`). Because sqlc can't easily do dynamic
`IN`/optional filters, prefer **a few explicit named queries** over one
mega-query, or build the WHERE dynamically in Go with `sqlc`'s `sqlc.slice()`
where supported. Pragmatic approach for SQLite/`database/sql`: write the
dynamic SQL in the store layer (parameterised, never string-interpolated user
input) for the flexible list, and keep sqlc for the simple count queries.

Add:

```sql
-- name: ItemCount :one
SELECT COUNT(*) FROM items;

-- name: ItemSourceCounts :many
SELECT source, COUNT(*) AS count
FROM items
GROUP BY source
ORDER BY count DESC;
```

(Section/tag counts for the tabs can be a similar `GROUP BY tag` query — add
`ItemTagCounts` the same way.)

For the main filtered list, add a `ListBrowse` method in the **store** that
builds SQL dynamically:

- `SELECT id, issue_id, source, tag, title, url, original_url, author_*,
  score, summary, published, (issue_id IS NOT NULL) AS in_digest FROM items`
- WHERE clauses appended only when the option is set:
  - `source IN (?, ?, …)` (bind each value)
  - `tag IN (?, …)`
  - `(title LIKE ? OR summary LIKE ?)` with `%term%`
  - `issue_id IS [NOT] NULL` when `InDigest != nil`
- ORDER BY:
  - New → `published DESC`
  - Top → `score DESC`
  - Hot → `score / (julianday('now') - julianday(published) + 2)` (simple
    time-decay; tune later) — keep it a single SQL expression
- `LIMIT ? OFFSET ?` from Page/PerPage.

Use `?` placeholders and `[]any` args — **never** format user input into the
SQL string.

### 3. Store (`pkg/store/items/store.go`)

- Update `List` to dispatch to the new browse path when none of
  `IssueID`/`From+To` are set (instead of erroring), OR add a distinct
  `ListBrowse(ctx, opts)` method and call it from the handler. Distinct method
  is cleaner — the existing `List` keeps its contract.
- Add `Count(ctx) (int64, error)` and `SourceCounts(ctx) ([]SourceCount, error)`
  (define `SourceCount{Source news.Source; Count int64}` in the domain).
- Update `transformItem` to set `InDigest` from the row's `issue_id` validity.

### 4. Repository interface (`pkg/domain/news/item.go`)

Add the new methods to `ItemRepository`:

```go
ListBrowse(ctx context.Context, opts ItemListOptions) ([]Item, error)
Count(ctx context.Context) (int64, error)
SourceCounts(ctx context.Context) ([]SourceCount, error)
TagCounts(ctx context.Context) ([]TagCount, error)
```

Then regenerate the mock (interface has a `//go:generate` directive):

```
go generate ./pkg/domain/... ./pkg/services/...
```

### 5. (Optional) `created_at` migration for time-based stats

Only if we want "+N today" / "pulled in last hour" stat cards. Add a migration
(Goose format — see CLAUDE.md) adding `created_at TIMESTAMP` to `items`,
backfill `created_at = published` for existing rows, and set it in `ItemCreate`.
If we skip this, drop those two stat cards (see designer-notes.md). **Confirm
with the owner before adding** — it's the only schema change in the whole
feature.

## Run / verify

```
sqlc generate
go generate ./pkg/domain/... ./pkg/services/...
go test ./...
golangci-lint run ./... --fix --config=.golangci.yaml
```

## Acceptance criteria

- `ListBrowse` returns paginated, filtered, sorted items with `InDigest` set.
- `Count`, `SourceCounts`, `TagCounts` return correct aggregates.
- Existing `List` (IssueID / From+To) behaviour unchanged; existing tests pass.
- New unit tests in `store_test.go` cover: source filter, tag filter, search,
  each sort, `InDigest` true/false/nil, pagination boundaries.
- No raw user input is concatenated into SQL.
