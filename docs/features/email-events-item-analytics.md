# Email Events: Item-Level Click Analytics

## Problem

The current `email_events` table (migration `0006`) embeds `issue_id` as a FK with `ON DELETE CASCADE`. This has two consequences:

1. **Silent data loss**: deleting an issue destroys all its engagement history.
2. **No item-level analytics**: clicks are stored with the clicked URL, but there is no explicit link to the `items` table. To see which article drove a click you must JOIN on URL strings at query time — fragile if URLs change.

There is also no API endpoint that surfaces this data.

---

## Options Considered

### Option A — Join at query time (no schema change)

Keep `email_events` as-is. Derive item clicks via SQL:

```sql
SELECT i.title, i.source, i.tag, COUNT(*) AS clicks
FROM email_events ee
JOIN items i ON i.issue_id = ee.issue_id
  AND (i.url = ee.url OR i.original_url = ee.url)
WHERE ee.event_type = 'clicked'
GROUP BY i.id ORDER BY clicks DESC;
```

- **Pro**: zero schema changes, works today against the live database.
- **Con**: URL-based join is fragile; CASCADE concern remains; item relationship is never stored explicitly.

### Option B — Two pivot tables

Make `email_events` fully agnostic (remove `issue_id`/`subscriber_id` columns). Add separate `issue_email_events` and `item_email_events` tables.

- **Pro**: clean domain separation.
- **Con**: every event write becomes 2–3 inserts; every read requires two extra JOINs. For 1:1 relationships (one event → one issue, one item) this is overhead without benefit.

### Option C — One context table (recommended)

Make `email_events` agnostic. Add a single `email_event_contexts` table that holds `issue_id`, `subscriber_id`, and `item_id` — all nullable, all with `ON DELETE SET NULL`.

```
email_events           → raw event facts (email, type, url, provider ids, timestamps)
email_event_contexts   → nullable FKs: issue_id, subscriber_id, item_id
```

- **Pro**: decouples `email_events` from domain entities; solves CASCADE concern; item_id is stored explicitly at write time; single join on reads.
- **Con**: one schema migration + updated write path.

---

## Decision: Option C

Option C is recommended because:

- The relationships (event → issue, event → item) are always 1:1, so two pivot tables add joins with no modelling benefit.
- Storing `item_id` explicitly at click-time is more reliable than re-deriving it from URL strings later.
- `ON DELETE SET NULL` on all FKs means analytics survive any entity deletion.
- The write overhead is minimal: one extra row per event in a tiny table.

---

## Implementation Plan

### Migration 0009

Recreate `email_events` without embedded domain FKs. Migrate existing `issue_id` and `subscriber_id` data into a new `email_event_contexts` table.

```sql
-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys = OFF;

CREATE TABLE email_events_new (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    email       TEXT NOT NULL,
    event_type  TEXT NOT NULL,
    url         TEXT,
    provider_id TEXT,
    event_id    TEXT NOT NULL,
    occurred_at TIMESTAMP NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO email_events_new (id, email, event_type, url, provider_id, event_id, occurred_at, created_at)
    SELECT id, email, event_type, url, provider_id, event_id, occurred_at, created_at
    FROM email_events;

CREATE TABLE email_event_contexts (
    email_event_id INTEGER PRIMARY KEY REFERENCES email_events_new(id) ON DELETE CASCADE,
    issue_id       INTEGER REFERENCES issues(id) ON DELETE SET NULL,
    subscriber_id  INTEGER REFERENCES subscribers(id) ON DELETE SET NULL,
    item_id        INTEGER REFERENCES items(id) ON DELETE SET NULL
);

INSERT INTO email_event_contexts (email_event_id, issue_id, subscriber_id)
    SELECT id, issue_id, subscriber_id FROM email_events
    WHERE issue_id IS NOT NULL OR subscriber_id IS NOT NULL;

DROP TABLE email_events;
ALTER TABLE email_events_new RENAME TO email_events;

CREATE UNIQUE INDEX idx_email_events_event_id       ON email_events (event_id);
CREATE INDEX        idx_email_events_type           ON email_events (event_type);
CREATE INDEX        idx_email_event_contexts_issue  ON email_event_contexts (issue_id);
CREATE INDEX        idx_email_event_contexts_item   ON email_event_contexts (item_id);

PRAGMA foreign_keys = ON;
-- +goose StatementEnd
```

Historical click events keep their `issue_id` and `subscriber_id`; `item_id` starts null for historical rows (URL-based backfill is possible but not required).

### SQL queries (`pkg/store/emailevents/query.sql`)

Add two queries:

```sql
-- name: EmailEventContextCreate :exec
INSERT INTO email_event_contexts (email_event_id, issue_id, subscriber_id, item_id)
VALUES (?, ?, ?, ?);

-- name: EmailEventTopItems :many
SELECT i.id AS item_id, i.title, i.url, i.source, i.tag, COUNT(*) AS clicks
FROM email_events ee
JOIN email_event_contexts c ON c.email_event_id = ee.id
JOIN items i ON i.id = c.item_id
WHERE ee.event_type = 'clicked'
  AND (sqlc.narg(issue_id) IS NULL OR c.issue_id = sqlc.narg(issue_id))
GROUP BY i.id
ORDER BY clicks DESC
LIMIT ?;
```

Update `EmailEventIssueStats` and `EmailEventTopLinks` to join through `email_event_contexts`.

### Item lookup at write time

When processing a `clicked` event in `pkg/services/emailevent/service.go`, look up the item by URL within the issue before creating the context row. This requires an `ItemFinder` interface injected into the service:

```go
type ItemFinder interface {
    FindByURLInIssue(ctx context.Context, url string, issueID int64) (int64, bool, error)
}
```

Add `ItemFindByURLInIssue` to `pkg/store/items/query.sql`:

```sql
-- name: ItemFindByURLInIssue :one
SELECT id FROM items
WHERE issue_id = ?
  AND (url = ? OR (original_url IS NOT NULL AND original_url = ?))
LIMIT 1;
```

The lookup is best-effort: a miss (e.g. unsubscribe link click) leaves `item_id` null.

### Internal-link filter

Drop click events whose URL is a GoDaily system link (confirm/unsubscribe). These have no `issue_id` and should not be stored:

```go
func isInternalLink(u string) bool {
    return strings.Contains(u, "godaily.dev/api/confirm") ||
        strings.Contains(u, "godaily.dev/api/unsubscribe")
}
```

### New domain type (`pkg/domain/engagement/event.go`)

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

Extend `EmailEventRepository`:

```go
TopItems(ctx context.Context, issueID *int64, limit int64) ([]ItemClicks, error)
```

### New API endpoint

`GET /api/issues/{slug}/engagement` — auth-gated, returns:

```json
{
  "stats":     { "delivered": 109, "unique_clicks": 14, "click_rate": 0.128, ... },
  "top_items": [ { "item_id": 42, "title": "...", "source": "medium", "tag": "article", "clicks": 5 }, ... ]
}
```

Add rewrite in `vercel.json`:
```json
{ "source": "/api/issues/:slug/engagement", "destination": "/api/issues/slug/engagement?slug=:slug" }
```

---

## Files to Change

| File | Change |
|------|--------|
| `pkg/db/migrations/0009_email_events_refactor.sql` | New migration |
| `pkg/store/emailevents/query.sql` | Add context create + top items queries; update existing queries |
| `pkg/store/internal/sqlc/query.sql.go` | Regenerated |
| `pkg/store/internal/sqlc/models.go` | Updated `EmailEvent` model |
| `pkg/store/items/query.sql` | Add `ItemFindByURLInIssue` |
| `pkg/store/items/store.go` | Implement `FindByURLInIssue` |
| `pkg/domain/engagement/event.go` | Add `ItemClicks`, `TopItems` to interface |
| `pkg/services/emailevent/service.go` | Add `ItemFinder`, `isInternalLink`, context write |
| `pkg/app.go` | Wire `ItemFinder` into service |
| `api/issues/slug/engagement.go` | New handler |
| `vercel.json` | Add rewrite |
| `pkg/mocks/domain/engagement/EmailEventRepository.go` | Add `TopItems` mock |

---

## Verification

1. `go build ./...` — clean compile.
2. Apply migration 0009 against a local DB snapshot; verify `email_event_contexts` is populated from existing rows.
3. `go test ./...` — all existing tests pass.
4. POST a synthetic `email.clicked` webhook with a real item URL → context row created with `item_id` set.
5. POST a synthetic `email.clicked` with a confirm URL → event is dropped.
6. `GET /api/issues/2026-05-22/engagement` → `top_items` array is non-empty.
