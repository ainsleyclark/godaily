# Stage 2: API

## Context

Builds on Stage 1 ([stage-1-database.md](stage-1-database.md)). The DB
and cron persistence are in place. This stage adds an HTTP API so the
subscriber lifecycle (signup, confirm, unsubscribe) and read endpoints
for issues and items become available. The cron fan-out switches from
the single `EMAIL_SEND_ADDRESS` to the subscribers table.

- Stage 1: Database + migrations ([stage-1-database.md](stage-1-database.md))
- Stage 2: API (this file)
- Stage 3: Web ([stage-3-web.md](stage-3-web.md))

## Goal

A `cmd/api` binary that serves the surface below. No UI yet.

## Tech

- HTTP: `github.com/ainsleydev/webkit/pkg/webkit` (already a dep at v0.13.2)
- Handlers use the `Handler func(c *webkit.Context) error` signature
- Routes registered via `kit.Get/Post(...)` and grouped via `kit.Group("/api", ...)`
- Middleware via `kit.Plug(...)` (global) or per-route plug args

## Endpoints

| Method | Path                  | Purpose                                              |
|--------|-----------------------|------------------------------------------------------|
| GET    | `/healthz`            | Liveness                                             |
| GET    | `/api/issues`         | Paginated list, JSON                                 |
| GET    | `/api/issues/{slug}`  | Single issue with its items                          |
| GET    | `/api/items/{id}`     | Single news item                                     |
| POST   | `/api/subscribe`      | `{email}` body, rate-limited, sends confirm email    |
| GET    | `/api/confirm`        | `?token=...`, marks `confirmed_at`                   |
| GET    | `/api/unsubscribe`    | `?token=...`, marks `unsubscribed_at` (one-click)    |
| POST   | `/api/unsubscribe`    | `{email}` body, requests an unsubscribe email if `confirmed_at` is set (covers users who lost the link) |

Per-route plugs: rate limiter on `POST /api/subscribe` and
`POST /api/unsubscribe`.

## Subscriber flow

1. `POST /api/subscribe` validates the email, generates `confirm_token`
   and `unsubscribe_token`, inserts the row with both tokens, mails the
   confirm link. On duplicate email, idempotently re-send the confirm
   if not yet confirmed.
2. `GET /api/confirm?token=...` flips `confirmed_at`. The confirm token
   is then inert.
3. `GET /api/unsubscribe?token=...` flips `unsubscribed_at`. The
   unsubscribe token is stable so it works from any past digest.
4. `POST /api/unsubscribe` is the manual fallback: look up by email,
   and if the row is confirmed, re-send the unsubscribe link via Resend.
   Never confirm or deny existence in the response (avoid email
   enumeration).

## Files

To create:

- `cmd/api/main.go`: wires `webkit.New()`, DB, handlers, plugs, calls `kit.Start(":8080")`
- `internal/api/handler.go`: handlers grouped by resource
- `internal/api/middleware.go`: rate limiter plug, request logger
- `internal/subscribers/service.go`: signup, confirm, unsubscribe logic, token generation, dedupe rules
- `internal/email/confirm.go`, `internal/email/unsubscribe.go`: Resend templates and senders

To modify:

- `internal/cron/run.go`: read confirmed, non-unsubscribed subscribers
  via the subscribers query, fan out via Resend (batched). Preserve
  `EMAIL_SEND_ADDRESS` single-recipient mode as a dev fallback when
  `TURSO_URL` is unset.
- `internal/email/email.go`: extract a small helper for token-link URLs
- `Makefile`: `make api`
- `.env.example`: `WEB_BASE_URL` (used to build links in emails)

## Verification

1. `make api`, hit `/healthz`
2. `curl -X POST /api/subscribe` with an email, confirm email lands,
   click confirm, row updated
3. `make run`, the digest now goes to confirmed subscribers and contains
   a working unsubscribe link
4. Click unsubscribe, row updated, next `make run` skips that email
5. Handler tests via `httptest.NewRecorder` and webkit's own test
   helpers, table-driven per AGENTS.md
6. Subscribe and unsubscribe rate-limited (verify 429 on burst)
