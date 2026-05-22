# Email Events: Item-Level Click Analytics

## Problem

The `email_events` table (migration `0006`) has two shortcomings:

1. **Silent data loss.** `issue_id` is a FK with `ON DELETE CASCADE`, so deleting an issue destroys all of its engagement history.
2. **No item-level analytics.** Clicks store only the clicked URL. To learn which article drove a click you must JOIN `items` on URL strings at query time — fragile, and the relationship is never stored.

There is also no API endpoint that surfaces per-issue engagement.

## Approach

Keep a single `email_events` table. Add a nullable `item_id` column and change `issue_id` from `ON DELETE CASCADE` to `ON DELETE SET NULL`.

The event → issue/subscriber/item relationship is strictly 1:1, so a separate context table would only add a JOIN to every read and a second insert to every write with no modelling benefit. The simplest correct schema is the most durable one. The `EmailEventRepository` interface hides the physical layout, so the table can still be split later — at zero extra cost — if that ever becomes necessary.

One digest email carries one issue and one subscriber but **many items**, so `item_id` cannot be an email tag the way `issue_id` and `subscriber_id` are. It is resolved from the clicked URL at webhook time: the email renders item URLs verbatim and Resend reports the original link on the `clicked` event, so an exact match against `items.url` or `items.original_url` within the issue is reliable.

Clicks on GoDaily system links (footer unsubscribe, confirm) are stored normally — the item lookup simply misses and `item_id` stays null. These clicks still carry the digest's `issue_id`, so keeping them keeps the delivered/click counts honest.

## Implementation Plan

### Migration 0009

`pkg/db/migrations/0009_email_events_item_id.sql` recreates `email_events`. Nothing references `email_events` by FK, so a plain drop/recreate is safe with foreign keys on. Existing event data is intentionally discarded.

```sql
-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_email_events_type;
DROP INDEX IF EXISTS idx_email_events_issue_id;
DROP INDEX IF EXISTS idx_email_events_event_id;
DROP TABLE IF EXISTS email_events;

CREATE TABLE email_events (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    issue_id      INTEGER REFERENCES issues(id) ON DELETE SET NULL,
    subscriber_id INTEGER REFERENCES subscribers(id) ON DELETE SET NULL,
    item_id       INTEGER REFERENCES items(id) ON DELETE SET NULL,
    email         TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    url           TEXT,
    provider_id   TEXT,
    event_id      TEXT NOT NULL,
    occurred_at   TIMESTAMP NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_email_events_event_id ON email_events (event_id);
CREATE INDEX idx_email_events_issue_id ON email_events (issue_id);
CREATE INDEX idx_email_events_item_id  ON email_events (item_id);
CREATE INDEX idx_email_events_type     ON email_events (event_type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_email_events_type;
DROP INDEX IF EXISTS idx_email_events_item_id;
DROP INDEX IF EXISTS idx_email_events_issue_id;
DROP INDEX IF EXISTS idx_email_events_event_id;
DROP TABLE IF EXISTS email_events;

CREATE TABLE email_events (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    issue_id      INTEGER REFERENCES issues(id) ON DELETE CASCADE,
    subscriber_id INTEGER REFERENCES subscribers(id) ON DELETE SET NULL,
    email         TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    url           TEXT,
    provider_id   TEXT,
    event_id      TEXT NOT NULL,
    occurred_at   TIMESTAMP NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_email_events_event_id ON email_events (event_id);
CREATE INDEX idx_email_events_issue_id ON email_events (issue_id);
CREATE INDEX idx_email_events_type ON email_events (event_type);
-- +goose StatementEnd
```

The down migration does not touch `subscribers.bounced_at` — that column belongs to migration `0006`. Production is forward-only; goose applies `0009` on the next deploy via `db.Up`. The migration must not be pre-applied in the Turso dashboard, or goose's version table desyncs and the `CREATE TABLE` collides.

### SQL queries

`pkg/store/emailevents/query.sql` — add `item_id` to the insert and add a top-items query. `EmailEventIssueStats` and `EmailEventTopLinks` are unchanged.

```sql
-- name: EmailEventCreate :one
INSERT INTO email_events (
    issue_id, subscriber_id, item_id, email, event_type, url, provider_id, event_id, occurred_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: EmailEventTopItems :many
SELECT i.id AS item_id, i.title, i.url, i.source, i.tag, COUNT(*) AS clicks
FROM email_events ee
JOIN items i ON i.id = ee.item_id
WHERE ee.event_type = 'clicked'
  AND ee.issue_id = ?
GROUP BY i.id
ORDER BY clicks DESC
LIMIT ?;
```

`pkg/store/items/query.sql` — add a lookup that resolves a clicked URL to an item within an issue. The named `@url` param binds the click URL once for both columns.

```sql
-- name: ItemFindByURLInIssue :one
SELECT id FROM items
WHERE issue_id = @issue_id
  AND (url = @url OR (original_url IS NOT NULL AND original_url = @url))
LIMIT 1;
```

Run `make sqlc` afterwards to regenerate `pkg/store/internal/sqlc`.

### Domain (`pkg/domain/engagement/event.go`)

- Add `ItemID *int64` to `EmailEvent` — best-effort, nil when the click resolves to no item.
- Add an `ItemClicks` type:

```go
type ItemClicks struct {
    ItemID int64  `json:"item_id"`
    Title  string `json:"title"`
    URL    string `json:"url"`
    Source string `json:"source"`
    Tag    string `json:"tag"`
    Clicks int64  `json:"clicks"`
}
```

- Extend `EmailEventRepository`:

```go
TopItems(ctx context.Context, issueID int64, limit int64) ([]ItemClicks, error)
```

### Stores

`pkg/store/emailevents/store.go` — `Create` passes `item_id`; `transform` populates `ItemID`; add the `TopItems` method.

`pkg/store/items/store.go` — add a best-effort lookup that returns `false` (not an error) when no row matches:

```go
func (s Store) FindByURLInIssue(ctx context.Context, issueID int64, url string) (int64, bool, error)
```

### Item lookup at write time (`pkg/services/emailevent/service.go`)

Add a consumer-side interface, following the existing `SubscriberHealth` pattern:

```go
type ItemFinder interface {
    FindByURLInIssue(ctx context.Context, issueID int64, url string) (int64, bool, error)
}
```

Inject it into the service. In `Process`, after the duplicate check and before `Create`, resolve the item for `clicked` events:

```go
if e.Type == engagement.EmailEventTypeClicked && e.IssueID != nil && e.URL != "" {
    if id, ok, err := s.items.FindByURLInIssue(ctx, *e.IssueID, e.URL); err != nil {
        slog.WarnContext(ctx, "Item lookup for click event failed", "url", e.URL, "err", err)
    } else if ok {
        e.ItemID = &id
    }
}
```

A lookup error must not fail the webhook — log and continue with `item_id` nil. A miss leaves `item_id` nil; the event is still stored.

### Wiring (`pkg/app.go`)

Keep a concrete `*items.Store` reference so it satisfies both `news.ItemRepository` and `ItemFinder`, and pass it to `emailevent.New`. `FindByURLInIssue` is not added to `news.ItemRepository` — it stays off the shared interface.

### API endpoint

`GET /api/issues/{slug}/engagement` — new handler `api/issues/slug/engagement.go`, auth-gated via `api.HandleAuth`. Resolves the slug, then returns:

```json
{
  "stats":     { "delivered": 109, "unique_clicks": 14, "click_rate": 0.128, "...": "..." },
  "top_items": [ { "item_id": 42, "title": "...", "url": "...", "source": "medium", "tag": "article", "clicks": 5 } ]
}
```

The result limit comes from a `limit` query parameter (default 10).

Add the rewrite to `vercel.json`, above the existing `/api/issues/:slug` rule so the more specific path matches first:

```json
{ "source": "/api/issues/:slug/engagement", "destination": "/api/issues/slug/engagement?slug=:slug" },
{ "source": "/api/issues/:slug",            "destination": "/api/issues/slug?slug=:slug" }
```

## Files to Change

| File | Change |
|------|--------|
| `pkg/db/migrations/0009_email_events_item_id.sql` | New migration |
| `pkg/store/emailevents/query.sql` | Add `item_id` to insert; add `EmailEventTopItems` |
| `pkg/store/emailevents/store.go` | Persist/return `item_id`; add `TopItems` |
| `pkg/store/items/query.sql` | Add `ItemFindByURLInIssue` |
| `pkg/store/items/store.go` | Add `FindByURLInIssue` |
| `pkg/store/internal/sqlc/*` | Regenerated (`make sqlc`) |
| `pkg/domain/engagement/event.go` | Add `ItemID`, `ItemClicks`, `TopItems` |
| `pkg/services/emailevent/service.go` | Add `ItemFinder`, resolve item on clicked events |
| `pkg/app.go` | Wire `ItemFinder` into the service |
| `api/issues/slug/engagement.go` | New handler |
| `vercel.json` | Add rewrite |
| `pkg/mocks/domain/engagement/EmailEventRepository.go` | Regenerated (`go generate ./...`) |

## Verification

1. `make sqlc` then `go build ./...` — clean compile.
2. `go generate ./...` — mocks regenerate and compile.
3. `go test ./...` and `golangci-lint run` — all green.
4. Apply migration `0009` to a local DB; confirm `email_events` has `item_id` and `issue_id` is `ON DELETE SET NULL`.
5. POST a synthetic `email.clicked` webhook with a real item URL → row stored with `item_id` set; with an unsubscribe URL → row stored with `item_id` null.
6. `GET /api/issues/{slug}/engagement` with the auth token → `stats` plus a non-empty `top_items` array.
