# Slack notifications — what's left to do

The Slack notifications were reworked across PR #250 to use richer Block Kit
layouts. Done so far:

- **Social posts published / drafts published** — one section per platform with
  the live copy as a blockquote and a "View" button, collapsing to a single
  quote + button row when every platform shares the same text. The rotation
  kinds (recap, community, spotlight, cta, new_source) share this path, so they
  get the same card.
- **New subscriber**, **source-collection errors**, and a global code-block
  treatment for every `slack.Error`.
- **Card builder extracted** to `pkg/services/social/internal/slackcard`
  (package `slackcard`) — pure Block Kit assembly, separated from the service.
  Domain mapping (`platformLabel`, `kindLabel`) stays in the `social` service.
- **02:00 build summary** (`pkg/services/digest/buildsummary.go`) — leads with
  the subject + "View issue" button, the AI intro as a blockquote, a one-line
  meta context (date · stories · drafts across platforms), and each draft with
  an "Edit" button.
- **Friday weekly roundup** (`pkg/services/engagement/roundup.go`) — headline
  KPIs and subscriber stats as scannable two-column fields with deltas, ranked
  top links, and a "Best issue" row with a "View analytics" button plus a
  dashboard footer.

## Still to do

A few notifications are still plain title/body and could carry more of the
context already in scope:

- **AI synthesis review** (`pkg/services/digest/build.go:236`) — a plain
  `slack.Info` showing only subject + intro. It fires during the same 02:00
  build as the (now rich) build summary, so it is partly redundant; either
  promote it to a small card or fold it into the summary.
- **Error provenance** — every error now renders as a code block, but none use
  `ErrorWithContext` yet. The clearest win is the "Recording {platform} publish
  failed" error in `pkg/services/social/publish_drafts.go`: the platform post
  already succeeded, so the context line should include the live `PostURL` for
  manual backfill. Build/send failures could similarly attach the issue date or
  slug.
- **Reddit ingest** (`pkg/api/handlers/ingest/reddit.go`) — currently
  error-only. An optional success card could report received / persisted /
  duplicate counts, though this risks being noisy on a frequent cron.
