# Plan 03 — Page & Components (Templ)

**Goal:** the `pages.Browse` template + components — the 3-column layout from
the mockup (source sidebar · feed · trending sidebar), section tabs, sort bar,
item rows with the **digest marker**, and pagination. SSR-first; all controls
are links/forms that set querystring params (Plan 02).

**Depends on:** Plan 02 (the `BrowseProps` view-model contract).

## Files

- New: `web/views/pages/browse.templ`
- New: `web/views/components/browse_item.templ` (or extend `digest_item.templ`)
- New: `web/views/components/browse_filters.templ` (tabs, source list, sort bar)
- Reuse: `layouts.Base`, `components.Header`, `components.Container`,
  `components.SourceMark`, `components.Tag`, `components.SubscribeCTA`.

## View model (must match Plan 02)

```go
type BrowseProps struct {
    Items        []news.Item
    Total        int64
    SourceCounts []news.SourceCount
    TagCounts    []news.TagCount
    Filters      news.ItemListOptions // for active states + building links
    Page         int64
    HasNext      bool
}
```

## Layout (mirror the mockup)

```
@layouts.Base(meta)
  @components.Header()
  <section class="section browse">
    [hero: label "BROWSE", title, sub]
    [search box -> form GET /browse/ with name="q"]
    [stat cards: Total ("stories indexed"), Next digest in, (today/last-hour if created_at exists)]
    [tabs: All + one per news.SectionTags, each a link ?tab=… with TagCounts badge]
    <div class="browse__grid">  // 3 cols
      <aside class="browse__sources"> SourceCounts as checkable links ?source=… </aside>
      <div class="browse__feed">
        [sort bar: Hot/Top/New links ?sort=… ; range select ?range=… ; digest toggle ?digest=1]
        [result count "N stories"]
        for item := range Items { @BrowseItem(item) }
        [pagination: prev/next via ?page=…]
      </div>
      <aside class="browse__trending"> top items (see note) </aside>
    </div>
  @components.SubscribeCTA(...)
```

## The digest marker (the key new bit)

`news.Item.InDigest` (added in Plan 01) drives a small badge on each row:

```templ
templ BrowseItem(item news.Item) {
  <article class="browse-item">
    <div class="browse-item__badge">@SourceMark(item.Source)</div>
    <div class="browse-item__body">
      <a class="browse-item__title" href={ templ.URL(item.URL) }
         target="_blank" rel="noopener noreferrer">{ item.Title }</a>
      if item.InDigest {
        @Tag("In digest", TagProps{Dot: true})   // the visual marker
      }
      if item.Snippet != "" { <p class="browse-item__summary">{ item.Snippet }</p> }
      <div class="browse-item__meta">
        @Tag(string(item.Tag.Section()), TagProps{})
        // source name · published date
      </div>
    </div>
  </article>
}
```

- Reuse the existing `components.Tag` for the "In digest" chip so styling is
  consistent. `TagProps{Dot: true}` gives it the dot accent.
- **Optional enhancement:** if we also fetch the issue slug for digested items
  (a join or a second lookup in Plan 01/02), make the chip link to
  `/issues/{slug}/`. Confirm with owner — adds a join. Default: non-linked chip.
- Do **not** show comment counts (out of scope — no data).

## Tabs & filters as links (no JS required)

- Each tab/source/sort control is an `<a href>` that rebuilds the querystring,
  preserving the other active filters. Write a small templ helper
  `browseURL(filters, override...)` to construct these safely with
  `net/url.Values`.
- Mark the active tab/source/sort with an `--active` class by comparing against
  `Filters`.
- Source sidebar items show the count from `SourceCounts` (the `412`, `386`
  numbers in the mockup).

## Trending sidebar

Two options — pick the cheap one first:

- **Cheap (no new query):** reuse the same feed sorted by `Top`/`Hot`, take the
  first 5. Render as a numbered list.
- **Richer (later):** wire to `engagement.ItemList` (top-clicked) — that's the
  true "trending right now". Needs the engagement repo passed to the handler.

Start with the cheap version; leave a TODO for engagement-backed trending.

## Run / verify

```
templ generate   // if the project pre-generates; otherwise build handles it
go test ./...
golangci-lint run ./... --fix --config=.golangci.yaml
```

Visually check `/browse/` renders all sections of the mockup and that the
"In digest" chip appears only on items with `issue_id` set.

## Acceptance criteria

- Page renders with the 3-column layout and all controls as working links.
- "In digest" chip shows iff `item.InDigest`.
- Tabs/sources/sort reflect active state and preserve other filters when clicked.
- Empty state handled (no items match → friendly message, like issues.templ).
- Pagination prev/next work and disable at boundaries.
