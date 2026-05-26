# Preview E2E Tests

End-to-end tests that run the full production pipeline against the Vercel preview URL on every PR.

## What it tests

A single serial suite covering the complete journey: subscribe → confirm → collect → build → send →
assert digest received.

Unlike the local e2e suite (which uses an in-memory SQLite DB and a spy email sender), these tests
run against a real deployed Vercel preview, a real Turso preview database, and real Resend email
delivery — verified via Testmail.app.

## Pre-requisites (one-time setup)

### 1. Separate preview database in Turso

Create a `godaily-preview` database. In the Vercel dashboard add `TURSO_URL` and `TURSO_AUTH_TOKEN`
as **Preview-environment-only** env vars pointing at it. Without this the test would write to the
production database.

### 2. Testmail.app account

Sign up at [testmail.app](https://testmail.app) (free tier: 100 emails/month). From the dashboard,
note:

- `TESTMAIL_API_KEY`
- `TESTMAIL_NAMESPACE` (a short slug like `abc12`)

Incoming emails arrive at `<namespace>.<tag>@inbox.testmail.app`. Each test run generates a unique
tag, so parallel runs never collide.

### 3. GitHub repository secrets

Add the following secrets in **Settings → Secrets and variables → Actions**:

| Secret               | Value                            |
|----------------------|----------------------------------|
| `TESTMAIL_API_KEY`   | From Testmail dashboard          |
| `TESTMAIL_NAMESPACE` | From Testmail dashboard          |
| `API_SECRET`         | Same value as in Vercel env vars |

## How it works

The workflow file is `.github/workflows/preview-e2e.yaml`. It triggers on the `deployment_status`
event — Vercel fires this automatically when a preview build succeeds, so there is no polling.

The `BASE_URL` is set to `github.event.deployment_status.environment_url`, which is the live Vercel
preview URL for that PR.

## The `?force=true` flag

The three pipeline endpoints (`/api/collect`, `/api/build`, `/api/send`) now accept `?force=true`
when called with a valid `Authorization: Bearer <API_SECRET>` header. This bypasses:

- **Weekend guard** — normally returns 200 early on Sat/Sun
- **Already-sent guard** on `/api/send` — forwards `force=true` to `SendDigest`, which skips the "
  already sent today" check

This flag is only meaningful for automated testing; normal cron-triggered calls never pass it.

## The `CollectResponse` type

`Aggregator.Collect()` now returns a `CollectResponse` instead of `[]news.SourceItems`:

```go
type CollectResponse struct {
Sources []news.SourceItems
Errors  map[news.Source]error
}
```

A source that returns zero items (quiet news day) has no entry in `Errors`. A source that actually
failed has a non-nil error. The `/api/collect` endpoint serialises this as:

```json
{
	"ok": true,
	"sources": {
		"hn": {
			"count": 12,
			"error": null
		},
		"devto": {
			"count": 7,
			"error": null
		},
		"github": {
			"count": 0,
			"error": "rate limit exceeded"
		}
	}
}
```

The preview test asserts that every source has `error: null` and the total item count is > 0.

## Running locally against a preview URL

```bash
BASE_URL=https://your-preview.vercel.app \
TESTMAIL_API_KEY=... \
TESTMAIL_NAMESPACE=... \
API_SECRET=... \
  pnpm exec playwright test --config=playwright.preview.config.ts
```
