# Slack notifications — what's left to do

PR #250 enriched the highest-signal Slack messages with Block Kit
sections: social posts published (one section per platform with the live
copy and a "View" button, collapsing to a single quote + button row when
every platform shares the same text), social drafts published, new
subscriber, source-collection errors, and a global code-block treatment
for every `slack.Error`. The digest build summary
(`pkg/services/digest/buildsummary.go`) and the weekly engagement roundup
(`pkg/services/engagement/roundup.go`) were already rich and left as-is.
Note that the rotation kinds — recap, community, spotlight, cta,
new_source — need no separate work: they share the publish path
(`notifyPublishSummary` → the card builder) and so already render the new
card.

Two follow-ups remain, captured here rather than implemented:

## 1. Extract the card builder into an internal `slackcard` package

The social card builder currently lives in
`pkg/services/social/card.go` alongside the service. It should move to its
own internal package, `pkg/services/social/internal/slackcard` (package
`slackcard`), so the pure Block Kit assembly is cleanly separated from the
service logic. The name deliberately avoids `slack` to prevent a clash
with the `pkg/gateway/slack` import.

- **Scope:** move the Block Kit assembly and text helpers (`socialCard`,
  `sectionRow`, `contextBlock`, `linkButton`, `blockquote`, `truncate`,
  `cardRow`, `maxCardText`) into `slackcard`, exporting the entry point
  (e.g. `slackcard.Build`) and the `Row` type.
- **Tradeoff:** keep the domain mapping (`platformLabel`, `kindLabel`,
  `plural`) in the `social` service — `platformLabel` is also used by the
  per-platform error notifications, so pulling it into `slackcard` would
  couple the error path to the card package. The service builds the rows
  and titles; `slackcard` only assembles blocks.

## 2. Enrich the remaining thin messages

A few notifications are still plain title/body and could carry more of
the context already in scope:

- **AI synthesis review** (`pkg/services/digest/build.go:236`) — a plain
  `slack.Info` showing only subject + intro. Promote to a card with the
  item count and a "View issue" button.
- **Error provenance** — every error now renders as a code block, but
  none use `ErrorWithContext` yet. The clearest win is the
  "Recording {platform} publish failed" error in
  `pkg/services/social/publish_drafts.go`: the platform post already
  succeeded, so the context line should include the live `PostURL` for
  manual backfill. Build/send failures could similarly attach the issue
  date or slug.
- **Reddit ingest** (`pkg/api/handlers/ingest/reddit.go`) — currently
  error-only. An optional success card could report received / persisted
  / duplicate counts, though this risks being noisy on a frequent cron.
