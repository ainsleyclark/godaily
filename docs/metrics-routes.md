# Metrics API

GoDaily exposes a flat `/api/metrics/` namespace for reading engagement analytics. All routes
require API key authentication.

> **Note:** `/api/social/metrics` is a separate cron *writer* that refreshes platform stats.
> The `/api/metrics/...` routes here are *readers* — they query what's already stored.

---

## Routes

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/metrics/summary` | Yes | Headline numbers for a period — delivered, opens, clicks, rates |
| GET | `/api/metrics/issues` | Yes | Per-issue engagement stats with filtering and sorting |
| GET | `/api/metrics/issues/:slug` | Yes | Single issue stats + top-clicked links |
| GET | `/api/metrics/items` | Yes | Top-clicked items, enriched with title/tag/source |
| GET | `/api/metrics/tags` | Yes | Clicks aggregated by item tag |
| GET | `/api/metrics/sources` | Yes | Clicks aggregated by item source |
| GET | `/api/metrics/trend` | Yes | Time series for a chosen metric, bucketed daily/weekly |
| GET | `/api/metrics/subscribers` | Yes | Subscriber growth and churn over time |

---

## Common query parameters

These apply to every list endpoint (everything except `/issues/:slug`).

| Param | Type | Default | Notes |
|---|---|---|---|
| `from` | ISO date `YYYY-MM-DD` | — | Inclusive lower bound. Filters on `email_events.occurred_at` for click endpoints; on `issues.sent_at` for `/issues`. |
| `to` | ISO date `YYYY-MM-DD` | — | Exclusive upper bound. |
| `period` | `day` \| `week` \| `month` \| `year` \| `all` | `all` | Shorthand for "last N days from now": `day`=1, `week`=7, `month`=30, `year`=365. Ignored when `from` or `to` is set. Math is rolling, not calendar-aligned. |
| `sort` | string | endpoint default | Endpoint-specific allowlist — see each route below. Always sorts descending. |
| `limit` | int | `10` | Capped at `100`. |

**Validation errors (400):** `from` > `to`; `from`/`to` not parseable as `YYYY-MM-DD`; unknown `period`; unknown `sort`; `limit` < 1 or > 100.

---

## GET /api/metrics/summary

Headline rollup for a single time window — useful as the first call when you want one-glance
"how are we doing?" numbers.

**Query params:** `from`, `to`, `period` (see [Common query parameters](#common-query-parameters)).
`sort` and `limit` are ignored.

**Response**

```json
{
  "data": {
    "from": "2026-04-23",
    "to": "2026-05-23",
    "issues_sent": 21,
    "delivered": 6450,
    "unique_opens": 2010,
    "total_opens": 2890,
    "unique_clicks": 1120,
    "total_clicks": 1610,
    "bounced": 31,
    "complained": 1,
    "open_rate": 0.312,
    "click_rate": 0.174,
    "unique_subscribers_engaged": 1380
  }
}
```

`unique_subscribers_engaged` = count of distinct `subscriber_id`s with any open or click in
the window.

---

## GET /api/metrics/issues

Lists aggregate email engagement per issue. Supports filtering by `sent_at` window and sorting by
any engagement metric — use this to answer questions like *"which issue had the highest click rate
last week?"*.

**Query params:** `from`, `to`, `period`, `sort`, `limit` (see [Common query parameters](#common-query-parameters)).

**Sort allowlist:** `click_rate`, `open_rate`, `total_clicks`, `unique_clicks`, `total_opens`,
`unique_opens`, `delivered`, `sent_at` (default `sent_at`).

**Response**

```json
{
  "data": [
    {
      "issue_id": 7,
      "slug": "2026-05-22",
      "sent_at": "2026-05-22T08:00:00Z",
      "delivered": 312,
      "unique_opens": 98,
      "total_opens": 140,
      "unique_clicks": 55,
      "total_clicks": 72,
      "bounced": 2,
      "complained": 0,
      "delayed": 1,
      "failed": 0,
      "suppressed": 0,
      "open_rate": 0.314,
      "click_rate": 0.176
    }
  ]
}
```

`open_rate` = `unique_opens / delivered`. Treat as directional only — Apple Mail Privacy
Protection pre-fetches images and inflates open counts.

`click_rate` = `unique_clicks / delivered`. This is the primary engagement signal.

Issues with `issue_id IS NULL` events (non-digest sends such as confirmation emails) are excluded.

---

## GET /api/metrics/issues/:slug

Returns full stats and the top-clicked links for a single issue.

**Path param:** `slug` — the issue date slug, e.g. `2026-05-22`.

**Response**

```json
{
  "stats": {
    "issue_id": 7,
    "delivered": 312,
    "unique_opens": 98,
    "total_opens": 140,
    "unique_clicks": 55,
    "total_clicks": 72,
    "bounced": 2,
    "complained": 0,
    "delayed": 1,
    "failed": 0,
    "suppressed": 0,
    "open_rate": 0.314,
    "click_rate": 0.176
  },
  "links": [
    { "url": "https://go.dev/blog/...", "clicks": 18 },
    { "url": "https://pkg.go.dev/...", "clicks": 11 }
  ]
}
```

`links` is ordered by click count descending, capped at 10 entries.

**Error responses**

| Status | Meaning |
|--------|---------|
| 400 | `slug` param missing |
| 404 | No issue found for that slug |

---

## GET /api/metrics/items

Top-clicked news items across all issues, enriched with item metadata so the response is directly
human-readable (no follow-up `/items/:id` calls needed). Use this for *"what were the top
performing links in the last month?"* style questions.

**Query params:** `from`, `to`, `period`, `limit` (see [Common query parameters](#common-query-parameters)).

**Sort:** `clicks` desc only (not user-configurable).

`item_id` on `email_events` is populated on a best-effort basis — it's set only when a click
resolves to a known item record. Rows without a resolved item are excluded from this endpoint;
use the raw per-issue links list (`/api/metrics/issues/:slug`) if you need uncategorised URLs.

**Response**

```json
{
  "data": [
    {
      "item_id": 42,
      "title": "Go 1.24 release notes",
      "url": "https://go.dev/doc/go1.24",
      "tag": "release",
      "source": "hn",
      "clicks": 18
    }
  ]
}
```

---

## GET /api/metrics/tags

Total clicks grouped by item `tag` (e.g. `release`, `proposal`, `article`, `video`). Use for
*"which tag was most popular last week?"* style questions.

**Query params:** `from`, `to`, `period`, `limit` (see [Common query parameters](#common-query-parameters)).

**Sort:** `clicks` desc only.

Requires `email_events.item_id IS NOT NULL` (best-effort join). Clicks not resolvable to a known
item are excluded.

**Response**

```json
{
  "data": [
    { "tag": "release",  "clicks": 142 },
    { "tag": "proposal", "clicks": 98 },
    { "tag": "article",  "clicks": 71 }
  ]
}
```

---

## GET /api/metrics/sources

Total clicks grouped by item `source` (e.g. `hn`, `reddit`, `lobsters`, `github`, `youtube`). Use
for *"which source drives the most clicks?"* style questions.

**Query params:** `from`, `to`, `period`, `limit` (see [Common query parameters](#common-query-parameters)).

**Sort:** `clicks` desc only.

Requires `email_events.item_id IS NOT NULL` (best-effort join). Clicks not resolvable to a known
item are excluded.

**Response**

```json
{
  "data": [
    { "source": "hn",       "clicks": 220 },
    { "source": "reddit",   "clicks": 84  },
    { "source": "lobsters", "clicks": 41  }
  ]
}
```

---

## GET /api/metrics/trend

Time series for a chosen engagement metric, bucketed by day or week. Designed for chart UIs and
"how is X trending?" questions.

**Query params:** `from`, `to`, `period` (see [Common query parameters](#common-query-parameters)),
plus two endpoint-specific params:

| Param | Values | Default | Notes |
|---|---|---|---|
| `metric` | `delivered`, `unique_opens`, `total_opens`, `unique_clicks`, `total_clicks`, `open_rate`, `click_rate` | `click_rate` | The series value per bucket. |
| `bucket` | `day`, `week` | `day` | Bucket size. Buckets align to UTC day or ISO week start (Monday). |

`sort` and `limit` are ignored — the response is always chronologically ascending and includes
every bucket in the window, even if zero.

**Response**

```json
{
  "data": {
    "metric": "click_rate",
    "bucket": "day",
    "points": [
      { "bucket_start": "2026-05-17", "value": 0.18,  "delivered": 310 },
      { "bucket_start": "2026-05-18", "value": 0.165, "delivered": 305 },
      { "bucket_start": "2026-05-19", "value": 0.0,   "delivered": 0   }
    ]
  }
}
```

`delivered` is always included alongside the requested metric so callers can spot low-volume
buckets where rate values are noisy.

---

## GET /api/metrics/subscribers

Subscriber growth and churn bucketed over time. Useful for understanding list health independent
of engagement on a given issue.

**Query params:** `from`, `to`, `period` (see [Common query parameters](#common-query-parameters)),
plus:

| Param | Values | Default | Notes |
|---|---|---|---|
| `bucket` | `day`, `week`, `month` | `day` | Bucket size. |

`sort` and `limit` are ignored — the response is always chronologically ascending.

`subscribers.created_at` drives `new`, `confirmed_at` drives `confirmed`, `unsubscribed_at` drives
`unsubscribed`, and `bounced_at`+`suppressed_at` drive `lost`. `net_change` = `confirmed - lost`.
`active_at_end` is the running total of confirmed minus lost as of the bucket end.

**Response**

```json
{
  "data": {
    "bucket": "day",
    "points": [
      {
        "bucket_start": "2026-05-22",
        "new": 12,
        "confirmed": 10,
        "unsubscribed": 2,
        "lost": 3,
        "net_change": 7,
        "active_at_end": 514
      }
    ]
  }
}
```

---

## Future routes

The `/api/metrics/` namespace is intentionally open-ended. Likely additions:

| Route | Description |
|-------|-------------|
| `GET /api/metrics/authors` | Clicks aggregated by `items.author_name` — useful once author coverage improves |
| `GET /api/metrics/domains` | Clicks aggregated by URL hostname — includes non-item clicks |
| `GET /api/metrics/positions` | Average clicks per digest position |

---

## Implementation notes

### Reusable query struct

All list endpoints share the same five query params, so they should share a single struct and
parser. Add this to `pkg/api/metricsparams.go` (alongside `params.go` and `pagination.go`):

```go
// MetricsQuery holds the common query parameters accepted by every /api/metrics list endpoint.
type MetricsQuery struct {
    From  *time.Time // nil = open-ended lower bound
    To    *time.Time // nil = open-ended upper bound
    Sort  string     // validated against the per-endpoint allowlist; empty = use defaultSort
    Limit int        // clamped to [1, MaxPerPage]
}

// ParseMetricsQuery reads from/to/period/sort/limit off the request, resolves period to a
// concrete (from, to) window if from/to are absent, validates sort against the allowlist, and
// clamps limit. Returns a 400-shaped HTTP error on any validation failure.
func ParseMetricsQuery(r *http.Request, allowedSorts []string, defaultSort string) (MetricsQuery, *HTTPError)
```

Handlers then collapse to:

```go
q, err := api.ParseMetricsQuery(r, []string{"click_rate", "open_rate", ...}, "sent_at")
if err != nil { err.Write(w); return }
rows, dbErr := store.MetricsIssuesList(r.Context(), q)
// ...
```

The store layer accepts `MetricsQuery` directly (or unpacks it into sqlc params), so date and sort
plumbing lives in one place. Endpoints with extra params (`/trend` adds `metric` and `bucket`,
`/subscribers` adds `bucket`) layer their own parsing on top of the shared parser.

### Other notes

- Vercel rewrite needed only for the parameterised route:
  `{ "source": "/api/metrics/issues/:slug", "destination": "/api/metrics/issues/slug?slug=:slug" }`.
  Query-param-only endpoints work natively without rewrites.
- The list-issues query groups by `issue_id` and excludes rows where `issue_id IS NULL`
  (events from non-digest sends such as confirmation emails).
- The list-items, tags and sources queries filter `event_type = 'clicked'` and require
  `item_id IS NOT NULL` so they can join through to `items`. Clicks that never resolved to a
  known item are silently excluded.
- Tag and source aggregations join `email_events` to `items` via `email_events.item_id =
  items.id`, then `GROUP BY items.tag` (or `items.source`).
- `period` math is **rolling** ("last N days ending now"), not aligned to calendar week/month
  boundaries. Use explicit `from`/`to` if you need calendar alignment.
- Rates (`open_rate`, `click_rate`) are computed in Go after the query to keep the aggregation
  queries simple. Sort-by-rate is implemented in SQL using the same expression to keep ordering
  consistent with the response values.
- The `/trend` endpoint zero-fills missing buckets in Go (SQL only returns buckets with events),
  so charts don't show gaps as missing data.
- Recommended index once event volume grows past ~1M rows:
  `CREATE INDEX idx_email_events_type_occurred ON email_events(event_type, occurred_at)`.
  Not needed at current scale.
