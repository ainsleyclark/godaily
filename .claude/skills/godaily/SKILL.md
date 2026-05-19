---
name: godaily
description: >
  Interact with the GoDaily production API (https://godaily.dev).
  Use when the user wants to: check API health, list or fetch digest issues,
  trigger news collection, send the digest, subscribe or unsubscribe emails,
  or inspect individual news items. Trigger on phrases like "show me the latest
  issue", "trigger collection", "send the digest", "subscribe X to GoDaily",
  "check if the API is up", or "list recent issues".
---

# GoDaily API Skill

GoDaily (`https://godaily.dev`) is a daily Go newsletter that aggregates news from Hacker News,
Reddit, Lobsters, Dev.to, Medium, GitHub Trending, YouTube, and more — synthesised by Claude and
delivered by email. This skill lets you interact with its HTTP API directly.

## Environment

| Variable | Purpose | Default |
|---|---|---|
| `GODAILY_API_URL` | Base URL for all requests | `https://godaily.dev` |
| `GODAILY_API_KEY` | Bearer token for protected endpoints | *(must be set)* |

Before any authenticated request, check that `GODAILY_API_KEY` is set. If it is empty or unset,
stop and tell the user:

> `GODAILY_API_KEY` is not set. Add it to your `.env` file or set it in your Claude Code
> environment configuration, then retry.

Use the env vars in every curl command:

```bash
BASE="${GODAILY_API_URL:-https://godaily.dev}"
```

## Endpoints

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/healthz` | No | API health check |
| GET | `/api/issues` | **Yes** | List paginated digest issues |
| GET | `/api/issues/{slug}` | No | Fetch a single issue by date slug (e.g. `2026-05-15`) |
| GET | `/api/items/{id}` | No | Fetch a single news item by numeric ID |
| GET | `/api/collect` | **Yes** | Trigger the news collection pipeline |
| GET | `/api/send` | **Yes** | Send the current draft digest by email |
| POST | `/api/subscribe` | No | Subscribe an email address |
| GET | `/api/confirm` | No | Confirm a subscription via token |
| GET / POST | `/api/unsubscribe/` | No | Unsubscribe via token (POST is RFC 8058 one-click) |

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

List recent issues. Supports pagination via `?page=` and `?per_page=` (max 100).

```bash
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/issues?page=1&per_page=10" | jq .
```

Response shape:
```json
{
  "data": [
    {"id": 1, "slug": "2026-05-15", "subject": "...", "status": "sent", "summary": "...", "sent_at": "..."}
  ],
  "page": 1,
  "per_page": 10,
  "total": 42
}
```

When presenting: show total count, then for each issue: slug, subject, status, and sent_at.

---

### Get issue by slug

Fetch a single digest issue by its date slug (format: `YYYY-MM-DD`). No auth required.

```bash
BASE="${GODAILY_API_URL:-https://godaily.dev}"
SLUG="2026-05-15"
curl -sf "$BASE/api/issues/$SLUG" | jq .
```

The response includes the issue metadata plus an `items` array of news items. When presenting:
show the subject, summary, sent_at, status, and a bulleted list of item titles with their tags and
scores.

---

### Get item by ID

Fetch a single news item by its numeric ID. No auth required.

```bash
BASE="${GODAILY_API_URL:-https://godaily.dev}"
ITEM_ID=123
curl -sf "$BASE/api/items/$ITEM_ID" | jq .
```

Key fields to highlight: `title`, `url`, `tag`, `source`, `score`, `summary`, `author`.

---

### Trigger collection

Run the news collection pipeline. This fetches items from all sources, scores them, synthesises
summaries via Claude, and saves a draft issue to the database. Takes up to ~5 minutes; the request
will block until complete.

```bash
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/collect" | jq .
```

Expected success: `{"ok": true}`

---

### Send the digest

Send the current draft digest by email to all active subscribers. The issue status changes from
`draft` to `sent` (or `error`).

```bash
BASE="${GODAILY_API_URL:-https://godaily.dev}"
curl -sf \
  -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/send" | jq .
```

Expected success: `{"ok": true}`

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
curl -sfL "$BASE/api/unsubscribe/?token=$TOKEN"
```

---

## Presenting Results

- Use `jq` to pretty-print all JSON responses.
- For list responses, lead with the total count: *"Found 42 issues. Showing page 1 of 5."*
- For issue details, present: slug, subject, status, sent_at, and item count; then list items grouped by tag.
- For errors, show the HTTP status code and the `"error"` field from the JSON body.
- For `{"ok": true}` responses, confirm the action succeeded in plain English.
- If `curl` exits non-zero (network failure, 4xx/5xx), report the status code and suggest checking `GODAILY_API_KEY` for auth failures.

## Workflow

1. **Identify the intent** — determine which endpoint(s) satisfy the user's request.
2. **Check auth** — if the endpoint requires auth, verify `GODAILY_API_KEY` is non-empty; halt with a helpful message if not.
3. **Build the command** — construct the `curl` command using `$GODAILY_API_URL` and `$GODAILY_API_KEY`.
4. **Execute** — run the command via Bash.
5. **Present** — parse the response with `jq` and summarise the key fields in plain English.
