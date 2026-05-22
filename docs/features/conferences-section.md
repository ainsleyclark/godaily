# Conferences Section

Surfaces major Go conferences in a dedicated digest section placed just below Proposals — reflecting their importance as key news in the Go community.

## Why a Separate Section

Local meetups (surfaced via the Meetup source with `TagEvent`) are a different type of content from major multi-day conferences like GopherCon. Conferences get their own section so they stand out at the top of the digest, not buried below articles and discussions.

## Data Source

Conferences are driven by a **curated static YAML file** at `pkg/source/conferences.yaml`.

This is intentional. External sources like `go.dev/wiki/Conferences` are maintained by the community but can lag or omit conferences (the wiki has historically been missing GopherCon UK). A committed YAML file gives:

- 100% accuracy for the conferences we choose to cover
- A single PR as the audit trail for every change
- Zero scraping fragility

### Updating the YAML

The file is manually maintained by the repo maintainer. The expected cadence is:

- **Yearly**: Update dates for recurring conferences (typically 5–10 line changes)
- **As needed**: Add a new entry when a new conference is announced

Use `make conferences-check` to discover conferences listed in the Go wiki that are not yet in the YAML. The command fetches `go.dev/wiki/Conferences` and prints any URLs absent from the local file.

```sh
make conferences-check
```

The wiki is used as a discovery signal only — the YAML is the source of truth.

## Notification Phases

Each conference emits up to three notifications at explicit dates configured per entry:

| Index | Tag                    | Timing                  |
|-------|------------------------|-------------------------|
| 0     | `TagConference`        | Announcement            |
| 1     | `TagConferenceReminder`| ~3 months before        |
| 2     | `TagConferenceAlert`   | ~1 week before          |

All three fold into the `Conferences` display section via `Tag.Section()`.

Notify dates are **the day the item should appear in the digest** — set `notify_date` to the date you want readers to see the item, not the day before.

### How Timing Works

The collect pipeline runs daily and captures items published in yesterday's window `[yesterday midnight, today midnight)`. The conference source emits items with `Published = yesterday noon` when `today == notify_date`. This ensures the item falls exactly in the current collect window.

### Deduplication

Each phase uses a distinct (url, tag) pair in the database:

- Phase 0: `https://gophercon.co.uk/#announce` + `TagConference`
- Phase 1: `https://gophercon.co.uk/#reminder` + `TagConferenceReminder`
- Phase 2: `https://gophercon.co.uk/#alert` + `TagConferenceAlert`

The URL fragment is stripped in email rendering; click targets land on the correct conference page. The `items_url_tag_unique` index (migration `0007`) prevents the same phase from being stored twice if the cron runs multiple times in a day.

## YAML Entry Format

```yaml
- slug: gophercon-uk-2026          # unique identifier (year-scoped)
  name: GopherCon UK 2026          # display name in digest
  url: https://www.gophercon.co.uk/
  location: London, UK
  start_date: 2026-08-12           # YYYY-MM-DD
  end_date: 2026-08-13
  description: "The UK's annual Go conference."  # digest snippet; falls back to "Location · Date" if empty
  image_url: ""                    # optional header image for the digest card
  notify_dates:
    - 2026-03-02                   # announcement
    - 2026-05-12                   # reminder (~3 months before)
    - 2026-08-05                   # alert (1 week before)
```

**Notes:**
- `slug` must be unique across all entries. Use `<name>-<year>` convention.
- Duplicate the entry with a new year's slug when updating recurring conferences; remove the old entry once all its notify_dates have passed.
- Notify dates that are in the past are silently skipped — the source only emits items for today's date.

## Conference Videos / Recordings

Recordings of talks from GopherCon and other conferences belong in the **Videos section**, not the Conferences section. Mixing "Register now" and "Watch the talk" in one section hurts clarity.

**Planned approach (separate PR):**
- Extend `pkg/source/youtube.go` to monitor known GopherCon YouTube channels/playlists in addition to the Golang channel it already fetches
- Tag those items `TagVideo` (no new tag needed)
- Channels to monitor: GopherCon UK, GolangChannel (@GolangChannel), GopherCon EU

No code changes are needed in this PR. The YouTube source already handles `TagVideo` rendering correctly.

## Implementation Files

| File | Purpose |
|------|---------|
| `pkg/source/conferences.yaml` | Curated conference list (edit this to add/update conferences) |
| `pkg/source/conferences.go` | Source implementation — reads YAML, emits items on notify days |
| `pkg/source/conferences_test.go` | Unit tests |
| `pkg/domain/news/item.go` | `TagConference`, `TagConferenceReminder`, `TagConferenceAlert` tags |
| `pkg/db/migrations/0007_items_url_tag_unique.sql` | Unique index on `(url, tag)` — prevents duplicate collection |
| `pkg/store/items/query.sql` | `INSERT OR IGNORE` upsert for dedup |
| `cmd/conferences-check/main.go` | Discovery helper: compares wiki vs local YAML |
