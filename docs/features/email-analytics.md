# Email Analytics via Resend Webhooks

Track open rates, link clicks, bounces, and spam complaints to understand what content resonates and keep the subscriber list healthy.

## Overview

Resend supports outbound webhooks for email lifecycle events. Consuming these events gives GoDaily actionable data without any third-party tracking scripts in the email itself.

## Events

| Event | Action |
|-------|--------|
| `email.opened` | Increment open count on the issue |
| `email.clicked` | Record which link was clicked and how often |
| `email.bounced` | Mark subscriber as inactive, stop sending |
| `email.complained` | Immediately unsubscribe the address |

## New Endpoint

`POST /api/webhooks/resend` — validates the Resend signature header and stores the event. This endpoint must be public but signature-verified.

## Data Model

A new `email_events` table records `(issue_id, subscriber_id, event_type, url, occurred_at)`. Aggregate queries can then answer: which issues had the highest open rate, which links were clicked most, how many unique openers per issue.

## Surfaces

Metrics can be displayed on a simple internal admin page or exported as structured logs for external dashboards.
