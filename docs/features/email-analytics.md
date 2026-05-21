# Email Analytics via Resend Webhooks

Track open rates, link clicks, bounces, and spam complaints to understand what content resonates and keep the subscriber list healthy.

## Overview

Resend supports outbound webhooks for email lifecycle events. Consuming these events gives GoDaily actionable data without any third-party tracking scripts in the email itself.

## Events

| Event | Action |
|-------|--------|
| `email.delivered` | Recorded — the denominator for open and click rates |
| `email.opened` | Increment open count on the issue (unreliable — see below) |
| `email.clicked` | Record which link was clicked and how often |
| `email.bounced` | Mark subscriber as bounced, stop sending |
| `email.complained` | Immediately unsubscribe the address |

Opens are stored but treated as noise: Apple Mail Privacy Protection pre-fetches images and inflates opens for a large share of subscribers. The loop weights click-through rate, unsubscribes, and complaints — never opens alone.

## Endpoint

`POST /api/webhooks/resend` (`api/webhooks/resend.go`) — public but signature-verified. Resend signs every request with Svix-style headers (`svix-id`, `svix-timestamp`, `svix-signature`); the handler verifies them against `RESEND_WEBHOOK_SECRET` before doing anything. HTTP status codes are chosen for Resend's retry behaviour: `2xx` acknowledges, `5xx` asks Resend to retry, `4xx` reports a permanent rejection.

## Data Model

The `email_events` table (migration `0006`) records `(id, issue_id, subscriber_id, email, event_type, url, provider_id, event_id, occurred_at, created_at)`. `issue_id` and `subscriber_id` are nullable — events for non-digest mail still record. `event_id` (the Svix message ID) is uniquely indexed, so duplicate webhook deliveries are no-ops.

Each outbound digest send carries `issue_id` and `subscriber_id` as Resend **tags**; Resend echoes the tags back on every webhook event, which is how an event is correlated to its issue and subscriber without a send-time database write.

Aggregate queries (`pkg/store/emailevents`) answer: per-issue delivered / unique opens / clicks / bounces / complaints with open and click rates, and the most-clicked links per issue.

## Surfaces

This phase ships a queryable data foundation — the aggregates above plus structured logs. Metrics are consumed by the growth loop's `growth-digest` report; no admin UI is built.

## Setup on Resend

1. In the [Resend dashboard](https://resend.com), open **Webhooks → Add Webhook**.
2. Set the endpoint URL to `https://godaily.dev/api/webhooks/resend`.
3. Subscribe to these events: `email.delivered`, `email.opened`, `email.clicked`, `email.bounced`, `email.complained`.
4. After creating the webhook, copy its **Signing Secret** (it starts with `whsec_`).
5. Set it as the `RESEND_WEBHOOK_SECRET` environment variable (in Vercel for production, or `.env` locally). The endpoint returns `500` until this is configured.
6. Use Resend's **Send test event** button to confirm a `2xx` response.

Sample webhook payloads for each event type live in `examples/webhooks/resend/` and double as test fixtures.
