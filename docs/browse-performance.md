# Browse page performance

The `/browse` archive is an HTMX-driven page: tab, filter, sort, search and
pagination controls issue a `GET /api/browse` request that swaps `#browse-main`
(`hx-swap="outerHTML"`) plus a couple of out-of-band regions. See
`web/views/pages/browse.templ` and `web/handlers/browse.go`.

## What's done

**Loading feedback (UX).** Interactions used to feel frozen — a click took a
beat with no signal that anything was happening. The browse `<section>` now
carries:

- `hx-indicator="#browse-progress, #browse-main"` — an indeterminate progress
  bar (`.browse__progress`, styled in `web/assets/scss/components/_browse.scss`)
  animates under the header while a request is in flight, and the stale results
  column dims to ~50% so it reads as "updating".
- `hx-disabled-elt="this"` — the control that fired the request is disabled for
  the request's lifetime, preventing a second filter firing mid-swap.

This makes the page *feel* responsive but does not change how long the request
actually takes.

## What's left (backend latency)

The 2–3s wall-clock cost lives in `BuildBrowseProps`
(`web/handlers/browse.go`), which runs 6–7 DB queries **sequentially** on every
request. In rough priority order:

1. **`matchingCount` re-runs an unpaginated `List`.** When you're past page 1
   (or the page is full), `matchingCount` (`browse.go:208`) calls
   `items.List` with no pagination purely to `len()` the result — pulling every
   matching row into memory to display a count. `digestPicksCount`
   (`browse.go:222`) does the same. These should be `SELECT COUNT(*)` queries
   with the same `WHERE` clause. **Biggest single win.**

2. **Missing indexes.** Browse filters put `source` and `tag` in
   `WHERE ... IN (...)` (see `pkg/store/items/store.go`), but neither column is
   indexed (`pkg/db/migrations/0001_init.sql`). Add indexes on `items(source)`
   and `items(tag)`; consider a composite covering the common
   filter-plus-sort paths. Requires a Goose migration (see `CLAUDE.md`).

3. **Full-table aggregates every request.** `SourceCounts()` and `TagCounts()`
   are `GROUP BY` scans over the whole `items` table, run on every browse hit to
   populate the sidebar counts. They change slowly — cache them with a short
   TTL (or precompute) rather than recomputing per request.

4. **Sequential execution.** Even after the above, the independent queries
   (`List`, `Count`, `SourceCounts`, `TagCounts`, `Latest`) can run concurrently
   via `errgroup`.
