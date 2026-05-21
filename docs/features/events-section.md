# Events Section

Surfaces upcoming Go meetups and conferences in the newsletter digest under a dedicated Events section.

## Current State (PR #124)

- Source: `pkg/source/meetup.go` — fetches a curated list of Meetup.com group pages
- Parses the embedded `__NEXT_DATA__` Apollo GraphQL JSON blob (no paid API required)
- Only `ACTIVE` (upcoming) events are included; `PAST` and `[Outside Event]` prefixed events are dropped
- Tag: `news.TagEvent`
- Section appears after Discussions in the digest

## Event Lifecycle (Planned)

Each event has two meaningful moments worth surfacing:

| Moment | Tag | Trigger |
|--------|-----|---------|
| Announced | `TagEvent` | First collected with `status == "ACTIVE"` |
| It happened | `TagEventRecap` | First collected with `status == "PAST"` |

`TagEventRecap` does not exist yet. When added it should:
- Have its own section (or fold under Events)
- Be a natural fit for conferences specifically — a conference announcement comes in as `TagEvent`, the recap after it concludes as `TagEventRecap`

## Deduplication (Not Yet Implemented)

### Problem

The collect pipeline runs daily. Meetup events use `Published: time.Now().UTC()` so they
fall inside every day's build window. Without dedup, the same upcoming event appears in
every digest until the event date passes.

### Required DB Migration

Add a unique constraint on `(url, tag)` to the items table:

```sql
ALTER TABLE items ADD CONSTRAINT items_url_tag_unique UNIQUE (url, tag);
```

This allows at most one "announced" row and one "recap" row per event URL — satisfying
the two-moment lifecycle above without blocking the state transition.

### Required Store Change

`pkg/store/items/store.go` `Create` should upsert on conflict rather than always insert:

```sql
INSERT INTO items (...) VALUES (...)
ON CONFLICT (url, tag) DO NOTHING;
```

`DO NOTHING` is correct: if the same event is re-collected in the same state the existing
row is kept, scores/snippets are not overwritten. The row was already included in a past
digest; silently dropping the duplicate is the right behaviour.

### Why Not `UNIQUE (url)` Alone

A plain URL constraint prevents storing both the announcement and the recap for the same
event. The `(url, tag)` pair is the correct granularity.

## Curated Group List

Groups are defined in `var meetupGroups` at the top of `pkg/source/meetup.go`. Adding a
group is one line. The official Meetup Pro network for Go (`meetup.com/pro/go`) lists 81
verified Go groups and is a good source for expanding coverage.

## Filtering

`ShouldInclude()` currently drops:
- `status != "ACTIVE"` — past events
- Title prefix `[Outside Event]` — cross-posted non-Go content posted by group admins

`[Paid]` events are kept — paid events at Go groups (conferences, workshops) are still
Go content.
