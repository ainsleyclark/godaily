---
name: godaily
description: >
  Interact with the GoDaily production API (https://godaily.dev).
  Use when the user wants to: check API health, list or fetch digest issues,
  trigger news collection or building, send the digest or preview, subscribe or
  unsubscribe emails, send confirmation nudges, post social updates, refresh
  social metrics, manually ingest Reddit items, inspect individual news items,
  or query engagement metrics (top links, best issues, most popular tags or
  sources over a time window). Trigger on phrases like "show me the latest
  issue", "trigger collection", "build the digest", "send the digest",
  "send the preview", "post featured", "post rotation", "subscribe X to
  GoDaily", "check if the API is up", "list recent issues", "top links last
  month", "best issue last week", "most popular tag", "which source drives the
  most clicks", "send the weekly roundup", or "refresh social metrics".
---

# GoDaily API Skill

GoDaily (`https://godaily.dev`) is a daily Go newsletter that aggregates news from Hacker News,
Reddit, Lobsters, Dev.to, Medium, GitHub Trending, YouTube, and more — synthesised by Claude and
delivered by email. This skill lets you interact with its HTTP API directly.

## API contract (source of truth)

The canonical, machine-readable API contract is the generated OpenAPI 3.1 spec at
**`docs/openapi/swagger.yaml`** (also `swagger.json`) in the repository. It is generated from the
Go handler annotations via `make openapi`, so it tracks production closely.

When you need exact paths, query/path parameters, request bodies, or response shapes — **read
`docs/openapi/swagger.yaml` and treat it as authoritative.** The tables and examples below are a
human-friendly quick reference; if they ever disagree with the spec, the spec wins. After the API
changes, regenerate with `make openapi` rather than hand-editing this skill.

## Environment

| Variable | Purpose | Default |
|---|---|---|
| `GODAILY_API_URL` | Base URL for all requests | `https://godaily.dev` |
| `GODAILY_API_KEY` | Bearer token for protected endpoints | *(must be set)* |

Before any authenticated request, check that `GODAILY_API_KEY` is set. If it is empty or unset,
**do not stop** — first attempt to source `~/.zshrc` silently, then re-check:

```bash
source ~/.zshrc 2>/dev/null
```

If the key is still unset after sourcing, tell the user:

> `GODAILY_API_KEY` is not set. It may be defined in `~/.zshrc` but Claude Code's shell doesn't
> source it automatically. Run `! export GODAILY_API_KEY=<your-key>` in the prompt, or add it
> via `/env`, then retry.

Always include `source ~/.zshrc 2>/dev/null` at the top of every Bash command block that needs
the key, so it is available even when Claude Code's shell hasn't inherited it.

Use the env vars in every curl command:

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
```

## Endpoints

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/healthz` | No | API health check |
| GET | `/api/digest/issues` | **Yes** | List paginated digest issues (filter by `?status=`) |
| GET | `/api/issues/{slug}` | **Yes** | Fetch a single issue by date slug (e.g. `2026-05-15`) |
| GET | `/api/items/{id}` | **Yes** | Fetch a single news item by numeric ID |
| GET | `/api/digest/collect` | **Yes** | Trigger the news collection pipeline |
| GET | `/api/digest/build` | **Yes** | Build today's collected items into a draft (`?force=true` to bypass weekend skip) |
| GET | `/api/digest/preview` | **Yes** | Send the draft digest preview to the owner (skipped at weekends) |
| GET | `/api/digest/send` | **Yes** | Send the current draft digest to all subscribers |
| GET | `/api/digest/nudge` | **Yes** | Send confirmation reminder emails to unconfirmed subscribers |
| GET | `/api/digest/subscribers` | **Yes** | List paginated subscribers |
| POST | `/api/subscribe` | No | Subscribe an email address |
| GET | `/api/confirm` | No | Confirm a subscription via token |
| GET / POST | `/api/unsubscribe` | No | Unsubscribe via token (POST is RFC 8058 one-click) |
| GET | `/api/social/featured` | **Yes** | Post today's featured social update across platforms |
| GET | `/api/social/rotation` | **Yes** | Post the next rotating social update across platforms |
| GET | `/api/social/metrics` | **Yes** | Refresh engagement stats for social posts (last 30 days) |
| POST | `/api/ingest/reddit` | **Yes** | Manually ingest raw Reddit listing JSON (fallback when live fetch is blocked) |
| POST | `/api/webhooks/resend` | No* | Receive Resend email webhook events (signature-verified) |
| GET | `/api/metrics/summary` | **Yes** | Headline rollup for a period |
| GET | `/api/metrics/issues` | **Yes** | Per-issue engagement stats with date/sort filters |
| GET | `/api/metrics/issues/{slug}` | **Yes** | Single-issue stats + top clicked links |
| GET | `/api/metrics/items` | **Yes** | Top clicked news items, enriched with title/tag/source |
| GET | `/api/metrics/tags` | **Yes** | Clicks aggregated by item tag |
| GET | `/api/metrics/sources` | **Yes** | Clicks aggregated by item source |
| GET | `/api/metrics/trend` | **Yes** | Time series for a chosen metric, bucketed daily/weekly |
| GET | `/api/metrics/subscribers` | **Yes** | Subscriber growth and churn over time |
| GET | `/api/metrics/roundup` | **Yes** | Trigger the weekly engagement roundup Slack post |
| GET | `/api/metrics/social` | **Yes** | Social post engagement metrics (likes, reposts, comments, impressions) |

*`/api/webhooks/resend` is public but every request is verified via Svix-style signature.

## Operations

### Health check

Verify the API is reachable and healthy.

```bash
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf "$BASE/api/healthz" | jq .
```

Expected: `{"ok": true}`

---

### List digest issues

List all issues. Supports pagination via `?page=` and `?per_page=` (max 100). Filter by
`?status=draft` or `?status=sent`.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/digest/issues?page=1&per_page=10" | jq .
```

Response shape:
```json
{
  "data": {
    "data": [
      {"id": 1, "slug": "2026-05-15", "subject": "...", "status": "sent", "summary": "...", "sent_at": "..."}
    ],
    "page": 1,
    "per_page": 10,
    "total": 42
  }
}
```

When presenting: show total count, then for each issue: slug, subject, status, and sent_at.

---

### Get issue by slug

Fetch a single digest issue by its date slug (format: `YYYY-MM-DD`). Auth required.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
SLUG="2026-05-15"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/issues/$SLUG" | jq .
```

The response includes the issue metadata plus an `items` array of news items. When presenting:
show the subject, summary, sent_at, status, and a bulleted list of item titles with their tags and
scores.

---

### Get item by ID

Fetch a single news item by its numeric ID. Auth required.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
ITEM_ID=123
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/items/$ITEM_ID" | jq .
```

Key fields to highlight: `title`, `url`, `tag`, `source`, `score`, `summary`, `author`.

---

### Trigger collection

Run the news collection pipeline. Fetches and ranks items from all registered sources; runs every
day including weekends. Takes up to ~5 minutes; the request will block until complete.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/digest/collect" | jq .
```

Response includes a `sources` map showing per-source item counts and any errors.

---

### Build the draft digest

Assembles today's collected items into a draft digest issue. Skipped on weekends unless
`?force=true` is passed.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/digest/build" | jq .
# Force a weekend build:
# curl -sf -H "Authorization: Bearer $GODAILY_API_KEY" "$BASE/api/digest/build?force=true" | jq .
```

Expected success: `{"ok": true, "message": "Successfully built digest"}`

---

### Send the preview

Sends the draft digest and AI synthesis suggestion to the owner ahead of the full subscriber send.
Skipped at weekends.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/digest/preview" | jq .
```

Expected success: `{"ok": true, "message": "Successfully sent preview"}`

---

### Send the digest

Send the current draft digest by email to all active subscribers. The issue status changes from
`draft` to `sent` (or `error`).

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/digest/send" | jq .
```

Expected success: `{"ok": true, "message": "Successfully sent digest"}`

---

### Send confirmation nudges

Sends a one-time reminder email to subscribers who signed up but never confirmed their address.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/digest/nudge" | jq .
```

Response: `{"data": {"sent": 5, "failed": 0}, "message": "..."}`

---

### List subscribers

List all subscribers, paginated.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/digest/subscribers?page=1&per_page=20" | jq .
```

---

### Subscribe an email address

Subscribe an email to the GoDaily newsletter. The subscriber will receive a confirmation email.

```bash
BASE="${GODAILY_API_URL:-https://godaily.dev}"
EMAIL="user@example.com"
curl -sf \
  -X POST \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL\"}" \
  "$BASE/api/subscribe" | jq .
```

Error cases: `400` invalid email, `409` already subscribed.

---

### Confirm subscription

Confirm a subscription using the token sent in the confirmation email.

```bash
BASE="${GODAILY_API_URL:-https://godaily.dev}"
TOKEN="the-confirm-token"
curl -sf "$BASE/api/confirm?token=$TOKEN" | jq .
```

---

### Unsubscribe

Unsubscribe using the token from the newsletter footer.

```bash
BASE="${GODAILY_API_URL:-https://godaily.dev}"
TOKEN="the-unsubscribe-token"
curl -sfL "$BASE/api/unsubscribe?token=$TOKEN"
```

---

## Social

All `/api/social/...` endpoints **require auth** and skip on weekends.

### Post featured social update

Posts today's featured item as a social update across platforms. Idempotent via the `social_posts`
table. Skipped at weekends and outside the chosen posting window.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/social/featured" | jq .
```

---

### Post rotation social update

Posts the next rotating social update (e.g. evergreen content) across platforms.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/social/rotation" | jq .
```

---

### Refresh social metrics

For every social post published in the last 30 days, fetches current engagement counts
(likes, reposts, comments, impressions) from each platform and upserts them into `social_metrics`.
Scheduled daily at 03:00 UTC.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/social/metrics" | jq .
```

---

## Ingest

### Manually ingest Reddit listing JSON

Fallback for when the live Reddit fetch is blocked (e.g. ScraperAPI 403). POST the raw JSON body
returned by `https://www.reddit.com/r/golang/new.json`. De-duplicates on `(url, tag)` so it is
safe to run repeatedly.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -X POST \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  -H "Content-Type: application/json" \
  --data-binary @reddit_new.json \
  "$BASE/api/ingest/reddit" | jq .
```

Response includes `received`, `persisted`, and `duplicates` counts.

---

## Metrics

All `/api/metrics/...` endpoints **require auth**. Full reference: `docs/metrics-routes.md`.

### Common query parameters

Every list endpoint accepts:

| Param | Values | Notes |
|---|---|---|
| `period` | `day`, `week`, `month`, `year`, `all` | Rolling window from now (`week`=last 7 days, `month`=last 30, `year`=last 365). Default `all`. |
| `from`, `to` | `YYYY-MM-DD` | Explicit range, overrides `period`. `from` inclusive, `to` exclusive. |
| `limit` | int, max `100` | Default `10`. |
| `sort` | per-endpoint allowlist | Only `/api/metrics/issues` accepts `sort`. Always descending. |

### Mapping natural-language time phrases

Translate consistently before building the curl:

| User says... | Use |
|---|---|
| "today" / "yesterday" / "in the last day" | `period=day` |
| "this week" / "last week" / "past week" | `period=week` |
| "this month" / "last month" / "past month" | `period=month` |
| "this year" / "last year" / "past year" | `period=year` |
| "of all time" / "ever" / no time mentioned | `period=all` (or omit) |
| "between 2026-05-01 and 2026-05-15" / "in May" | `from=YYYY-MM-DD&to=YYYY-MM-DD` |

If the user gives a vague phrase you can't map (e.g. "recently"), default to `period=week` and
mention the assumption in your reply.

### Period summary

Use this for one-glance "how are we doing?" requests.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/summary?period=month" | jq .
```

Returns a single object with `issues_sent`, `delivered`, `unique_opens`, `total_opens`,
`unique_clicks`, `total_clicks`, `bounced`, `complained`, `open_rate`, `click_rate`, and
`unique_subscribers_engaged`.

### Per-issue engagement stats

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/issues?period=week&sort=click_rate&limit=10" | jq .
```

Sort allowlist: `click_rate` (default for "best"), `open_rate`, `total_clicks`, `unique_clicks`,
`total_opens`, `unique_opens`, `delivered`, `sent_at` (default if omitted).

Each row includes `issue_id`, `slug`, `sent_at`, raw counters (`delivered`, `unique_opens`,
`total_opens`, `unique_clicks`, `total_clicks`, `bounced`, `complained`, `delayed`, `failed`,
`suppressed`), and computed `open_rate` / `click_rate`.

### Single issue stats + top links

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
SLUG="2026-05-22"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/issues/$SLUG" | jq .
```

Returns `{ "stats": {...}, "links": [{ "url": "...", "clicks": N }] }`. `links` is top 10.

### Top clicked items

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/items?period=month&limit=10" | jq .
```

Each row has `item_id`, `title`, `url`, `tag`, `source`, `clicks` — directly human-presentable.

### Clicks by tag

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/tags?period=week&limit=10" | jq .
```

Returns `[{ "tag": "release", "clicks": 142 }, ...]`.

### Clicks by source

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/sources?period=month&limit=10" | jq .
```

Returns `[{ "source": "hn", "clicks": 220 }, ...]`.

### Engagement trend (time series)

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/trend?period=month&metric=click_rate&bucket=day" | jq .
```

`metric` allowlist: `delivered`, `unique_opens`, `total_opens`, `unique_clicks`, `total_clicks`,
`open_rate`, `click_rate` (default `click_rate`). `bucket` is `day` or `week` (default `day`).
Response includes every bucket in the window even if zero, so charts don't break.

### Subscriber growth and churn

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/subscribers?period=month&bucket=day" | jq .
```

`bucket` is `day`, `week`, or `month` (default `day`). Each point has `new`, `confirmed`,
`unsubscribed`, `lost`, `net_change`, `active_at_end`.

### Weekly engagement roundup

Gathers the last seven days of engagement data with a week-over-week comparison and posts a
formatted summary to Slack. Normally scheduled every Friday at 15:00 UTC.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/roundup" | jq .
```

### Social post engagement metrics

Returns social posts joined with their latest engagement counts (likes, reposts, comments,
impressions), optionally filtered by date range.

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/social?period=month" | jq .
```

Accepts `period`, `from`, and `to` parameters (same as other metrics endpoints).

### Worked examples

| Question | Request |
|---|---|
| "How are we doing this month?" | `GET /api/metrics/summary?period=month` |
| "Top performing links in the last month" | `GET /api/metrics/items?period=month&limit=10` |
| "Best issue in the last week" | `GET /api/metrics/issues?period=week&sort=click_rate&limit=1` |
| "Most popular tag last week" | `GET /api/metrics/tags?period=week&limit=1` |
| "Which source drove the most clicks this year" | `GET /api/metrics/sources?period=year&limit=1` |
| "How did the 2026-05-22 issue perform" | `GET /api/metrics/issues/2026-05-22` |
| "Top 5 links between May 1 and May 15" | `GET /api/metrics/items?from=2026-05-01&to=2026-05-15&limit=5` |
| "Show me click-rate trend over the past month" | `GET /api/metrics/trend?period=month&metric=click_rate&bucket=day` |
| "How many subscribers did we gain last week" | `GET /api/metrics/subscribers?period=week&bucket=day` |
| "Send the weekly roundup" | `GET /api/metrics/roundup` |
| "Show social post engagement this month" | `GET /api/metrics/social?period=month` |

---

## Presenting Results

- Use `jq` to pretty-print all JSON responses.
- For list responses, lead with the total count: *"Found 42 issues. Showing page 1 of 5."*
- For issue details, present: slug, subject, status, sent_at, and item count; then list items grouped by tag.
- For errors, show the HTTP status code and the `"error"` field from the JSON body.
- For `{"ok": true}` responses, confirm the action succeeded in plain English.
- If `curl` exits non-zero (network failure, 4xx/5xx), report the status code and suggest checking `GODAILY_API_KEY` for auth failures.

## Workflow

1. **Identify the intent** — determine which endpoint(s) satisfy the user's request. When in doubt about a path, parameter, or response shape, consult `docs/openapi/swagger.yaml` (the authoritative contract).
2. **Check auth** — if the endpoint requires auth, verify `GODAILY_API_KEY` is non-empty; halt with a helpful message if not.
3. **Build the command** — construct the `curl` command using `$GODAILY_API_URL` and `$GODAILY_API_KEY`.
4. **Execute** — run the command via Bash.
5. **Present** — parse the response with `jq` and summarise the key fields in plain English.