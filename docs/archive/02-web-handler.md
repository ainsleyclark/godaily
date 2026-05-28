# Plan 02 — Web Handler & Route

**Goal:** add `GET /browse/` that parses querystring filters, calls the store,
and renders the page. All filter state lives in the URL (SSR-first, shareable,
works with no JS).

**Depends on:** Plan 01 (store methods + `ItemListOptions`).

## Files

- New: `web/handlers/browse.go`
- Edit: `web/server/server.go` (register route)
- The page template comes from Plan 03; this handler builds the view model.

## Route registration (`web/server/server.go`)

Alongside the existing routes (server.go:46-51):

```go
kit.Get("/browse/", handlers.Browse(a))
```

## Handler (`web/handlers/browse.go`)

Follow the existing pattern in `web/handlers/issues.go`:

```go
func Browse(a *godaily.App) webkit.Handler {
    return func(c *webkit.Context) error {
        ctx := c.Context()

        opts := parseBrowseQuery(c) // -> news.ItemListOptions

        items, err := a.Repository.Items.ListBrowse(ctx, opts)
        if err != nil { return renderErr(c) }

        total, _      := a.Repository.Items.Count(ctx)
        sourceCounts, _ := a.Repository.Items.SourceCounts(ctx)
        tagCounts, _  := a.Repository.Items.TagCounts(ctx)

        return c.Render(pages.Browse(pages.BrowseProps{
            Items:        items,
            Total:        total,
            SourceCounts: sourceCounts,
            TagCounts:    tagCounts,
            Filters:      opts,           // echo back for active states
            Page:         opts.Page,
            HasNext:      len(items) == int(opts.PerPage),
        }))
    }
}
```

(Use `c.RenderWithStatus(http.StatusInternalServerError, pages.Error(...))` on
error, matching issues.go.)

## Query parsing (`parseBrowseQuery`)

Map querystring → `news.ItemListOptions`. Suggested params:

| Param | Maps to | Notes |
|-------|---------|-------|
| `tab` / `section` | `Tags` | one section tag; validate against `news.SectionTags` |
| `source` | `Sources` | repeatable (`?source=hacker_news&source=reddit`); validate against `news.Sources` |
| `q` | `Search` | trim; cap length (e.g. 100 chars) |
| `sort` | `Sort` | `new`/`top`/`hot`; default `new` |
| `range` | `From`/`To` | `today`/`week`/`month`/`year`/`all` → compute window |
| `digest` | `InDigest` | `1`/`0`/absent → true/false/nil |
| `page` | `Page` | default 1; clamp ≥1 |

- **Validate/whitelist every value** against the known enums; ignore unknowns.
  Don't pass arbitrary strings to the store as a source/tag.
- Set `PerPage` to the page default (e.g. 30) here.
- Date `range` → reuse a small helper to turn the keyword into `*time.Time`
  bounds; `all` leaves both nil.

## "Next digest in" / stat cards

- `Total` (stories indexed) → from `Count`.
- "Next digest in" → derive from the digest cron schedule (constant/known time);
  no DB needed. If unknown, omit.
- "+N today" / "pulled in last hour" → require the `created_at` column
  (Plan 01 optional). Omit if not added.

## Run / verify

```
go test ./...
golangci-lint run ./... --fix --config=.golangci.yaml
```

Manually: start the web server, hit `/browse/`, `/browse/?source=hacker_news`,
`/browse/?sort=top`, `/browse/?digest=1`, `/browse/?range=week&q=generics`,
`/browse/?page=2`, and confirm filtering/sorting/paging works via URL alone.

## Acceptance criteria

- `/browse/` renders with no params (all items, page 1, sort=new).
- Every filter is driveable from the querystring and combinable.
- Invalid/unknown param values are ignored, not passed through to SQL.
- Handler has a test (mirror `web/handlers` patterns / mock the repo).
