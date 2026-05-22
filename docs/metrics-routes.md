# Metrics API

GoDaily exposes a flat `/api/metrics/` namespace for reading engagement analytics. All routes
require API key authentication.

> **Note:** `/api/social/metrics` is a separate cron *writer* that refreshes platform stats.
> The `/api/metrics/...` routes here are *readers* — they query what's already stored.

---

## Routes

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/metrics/issues` | Yes | List email engagement stats for all issues |
| GET | `/api/metrics/issues/:slug` | Yes | Single issue stats + top-clicked links |
| GET | `/api/metrics/items` | Yes | List click counts per news item (all-time) |

---

## GET /api/metrics/issues

Lists aggregate email engagement for every issue that has recorded events, newest first.

**Response**

```json
{
  "data": [
    {
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
    }
  ]
}
```

`open_rate` = `unique_opens / delivered`. Treat as directional only — Apple Mail Privacy
Protection pre-fetches images and inflates open counts.

`click_rate` = `unique_clicks / delivered`. This is the primary engagement signal.

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

Lists click counts for every news item that has been clicked at least once across all issues,
ordered by most-clicked first.

`item_id` is populated on `email_events` on a best-effort basis — it's set only when a click
resolves to a known item record. Rows without a resolved item are excluded.

**Response**

```json
{
  "data": [
    { "item_id": 42, "clicks": 18 },
    { "item_id": 7,  "clicks": 11 }
  ]
}
```

---

## Future routes

The `/api/metrics/` namespace is intentionally open-ended. Likely additions:

| Route | Description |
|-------|-------------|
| `GET /api/metrics/items/:id` | Per-item detail once `item_id` tracking is more reliable |
| `GET /api/metrics/social` | Read stored social engagement stats (followers, impressions, etc.) |
| `GET /api/metrics/subscribers` | Subscriber growth and churn over time |

---

## Implementation notes

- Vercel rewrite needed only for the parameterised route:
  `{ "source": "/api/metrics/issues/:slug", "destination": "/api/metrics/issues/slug?slug=:slug" }`
- The list-issues query groups by `issue_id` and excludes rows where `issue_id IS NULL`
  (events from non-digest sends such as confirmation emails).
- The list-items query groups by `item_id` and filters `event_type = 'clicked'` only.
- Rates are computed in Go after the query, not in SQL, to keep the aggregation queries simple.
