# Engagement-Driven Growth Loop

GoDaily ships a daily digest and auto-posts to social, but it does so
blind: nothing measures what content lands, and nothing turns that
measurement back into better content. This plan closes that gap.

The mental model matters more than any single feature. GoDaily is **not**
becoming an autonomous AI employee. It is becoming a **continuously
learning experimentation system**: a loop that records what happens,
notices patterns, proposes experiments, and applies the winners. The
database is the memory; the LLM only reasons over it. Every part of the
design follows from that distinction.

## Why this exists

There is no first-party data on what resonates and no mechanism to act on
it. Without stored memory of past experiments, any agent asked to "grow
GoDaily" re-derives the same generic advice every week. This plan builds
the data foundation first, then a small loop on top of it.

## The learning loop

```
 new content ──► raw events ──► content_metrics ──► insights
      ▲          (opens,        (aggregated)        (patterns)
      │           clicks,                               │
      │           social stats)                         ▼
      └── prompt / content changes ◄── experiments ◄─────┘
             (Tier 1 PRs)           (hypotheses → results)
```

Raw events become metrics, metrics become insights, insights become
experiments, winning experiments become prompt and content changes — and
those produce new content, which generates new events. The loop
compounds because each stage is **persisted in a table**, not held in an
agent's head.

## Principles

- **The database is the memory.** Institutional knowledge — what worked,
  what failed, which prompt produced which result — lives in tables. The
  LLM is stateless reasoning over that memory; it never "learns" on its
  own.
- **Deterministic jobs vs agentic steps.** Ingestion, aggregation, and
  scheduling stay plain Go. The LLM is used only where judgement is
  needed: generating insights, proposing experiments, drafting copy and
  code.
- **The gate matches the blast radius.** Code and prompt changes are
  always reviewed and merged by a person — nothing auto-merges. Public
  content publishes autonomously only within hard limits that live in
  code, not in a prompt.
- **Proposals cite prior outcomes.** Each suggestion references a
  measured experiment, so advice compounds instead of looping back to
  generic tips.
- **Reuse existing infrastructure** — the Slack gateway
  (`pkg/gateway/slack`), the AI client (`pkg/ai`), social adapters
  (`pkg/services/social/platform/*`), Resend, Vercel crons, and the sqlc + goose
  migration stack.

## Action tiers

A weekly run can produce one or more **actions**. They are not
interchangeable — each gets the lightest gate that is still safe.

| Tier | Action                | Gate                                        | Mechanism                                                          |
|------|-----------------------|---------------------------------------------|--------------------------------------------------------------------|
| 1    | Code or prompt change | **Human reviews & merges** a draft PR       | `claude/*` branch → draft PR                                       |
| 2    | Extra social post     | **Autonomous, within code-enforced limits** | capped, non-colliding `social_posts` slots — *deferred to Phase 6* |
| 3    | Metrics report        | None — information only                     | Slack snapshot                                                     |

Tier 1 is the deliberate lever — anything that changes how GoDaily
*works*, including prompt edits. Tier 2 is the fast lever (more reach,
no code review) and its safety is entirely structural; it is deliberately
deferred until the core loop is proven. Tier 3 is just visibility.

## Data model — five tables

The loop is built on five tables. All ship as goose migrations after the
current latest (`pkg/db/migrations/0005_social_posts.sql`), each with an
sqlc query file under `pkg/store/`.

### 1. `email_events` — raw telemetry

`(id, issue_id, subscriber_id, event_type, url, occurred_at)`

Append-only raw record of Resend webhook events. `email.clicked` is the
**primary signal** (records which URL was clicked); `email.opened` is
stored but flagged unreliable — Apple Mail Privacy Protection inflates
opens; `email.bounced` / `email.complained` mark the subscriber inactive
via `pkg/services/subscriber`.

**Populated by** the `POST /api/webhooks/resend` handler — one row per
event, never aggregated in place.

### 2. `content_metrics` — aggregated layer

`(content_id, platform, impressions, clicks, ctr, opens, open_rate,
likes, reposts, comments, unsubscribes, updated_at)`

The clean summary layer the loop reads — agents never touch raw events.
`content_id` is an issue id (email) or a `social_posts` id; `platform`
records the channel.

**Populated by** a deterministic `godaily aggregate-metrics` job (daily
cron) that rolls up `email_events` plus social-platform stats into one
row per piece of content.

### 3. `prompt_versions` — prompt registry

`(id, prompt_type, version, prompt_text, content_hash, created_at)`

Prompts stay in **Go code** — git is the source of truth (see
*Decisions*). This table is an auto-populated *registry*: on each
generation run the app hashes the active prompt; if the `content_hash`
is new, it inserts a version row that **snapshots** `prompt_text` for
historical and debugging reference.

**Populated automatically** at content-generation time. Each generated
row links back via a new `prompt_version_id` column on `issues` and
`social_posts` — so every digest and post remembers which prompt made
it, enabling a prompt-version performance leaderboard.

### 4. `experiments` — institutional memory

`(id, tier, hypothesis, variable, control_value, test_value, change_ref,
metric_name, baseline_value, result_value, uplift, confidence, status,
created_at, completed_at)`

The piece that makes the loop compound rather than repeat itself.
`status` moves through `proposed → shipped → measured → kept | rejected`.
`change_ref` points at a PR URL, a `prompt_versions` id, or a batch of
`social_posts`. Without this ledger the agent re-derives generic advice
every week; with it, a proposal reads: *"subject-line teasers (shipped
wk 12) lifted CTR 1.2pp — next, try X."*

**Populated by** the Phase 3 analyst (proposes new rows); generation and
evaluation jobs fill in `result_value`, `uplift`, and `confidence` after
~2 weeks of follow-on data.

### 5. `insights` — machine-generated conclusions

`(id, insight, confidence, supporting_data, created_at)`

Durable growth notes — e.g. *"subject lines over 70 chars reduce opens
(confidence 0.79)"*. Distinct from experiments: an insight is an
observation, an experiment is a test.

**Populated by** the Phase 3 weekly analyst from the LLM's structured
output.

## Cadence — daily vs weekly

The loop runs at two speeds so feedback is not needlessly slow:

- **Daily** — ingestion (`email_events`), aggregation
  (`content_metrics`), and lightweight metric refresh. Plain deterministic
  jobs.
- **Weekly** — the analyst run: insights, experiment proposals, and any
  Tier-1 PRs.

## Phases

### Phase 1 — Data foundation

Build the five tables above and the ingestion endpoint. **No agents
yet.** A new `POST /api/webhooks/resend` Vercel function
(`api/webhooks/resend.go`) verifies the Resend signature header and
writes `email_events` — verification is new code; the existing
`pkg/gateway/hook` package only does *outbound* heartbeats and deploy
hooks. This phase implements and extends `docs/features/email-analytics.md`.

### Phase 2 — `growth-digest` CLI

A new `godaily growth-digest` command (`pkg/cmd/`, registered in
`pkg/cmd/cmd.go`) aggregates ~4 weeks of `content_metrics`,
`experiments`, and `insights` into a single Markdown/JSON report: top
and worst content, top hooks and topics, the prompt-version leaderboard,
CTR trends, unsubscribe anomalies, and experiment outcomes. The agent
reads this report instead of touching the database — so its environment
never needs DB credentials. The deterministic `godaily aggregate-metrics`
job (daily Vercel cron) also lands here, and the command can post a
plain Slack metrics snapshot via `pkg/gateway/slack` — a no-AI baseline
useful even before Phase 3.

### Phase 3 — Weekly AI analyst

A **simple weekly cron** — no Claude Routine, no MCP connector. The flow:
run `growth-digest` → send the report to the LLM (`pkg/ai`) → the LLM
returns insights, hypotheses, and experiment ideas as JSON → store them
in `insights` and `experiments` (`status=proposed`) → post a summary to
Slack. One cron, one LLM call. This alone is already valuable.

### Phase 4 — Controlled experiments

Run real A/B tests: subject-line, hook, intro, and CTA variants. Each
variant is a registered `prompt_versions` row; a selector assigns issues
to control or test; `prompt_version_id` records the assignment; and
`content_metrics` + `experiments` measure the `uplift` and `confidence`.
This is where compounding begins.

### Phase 5 — Prompt & content evolution

The AI now proposes real changes — prompt edits, scoring tweaks, template
changes — as **draft PRs** on `claude/*` branches. Human reviews and
merges; never auto-merge. This is the first phase that needs an *agentic*
step able to write code and open PRs; a Claude Code Routine is one option,
deliberately deferred to here rather than introduced earlier. Candidate
levers already exist in `docs/features/synth-ideas.md` and
`docs/features/weekly-roundup.md`.

### Phase 6 — Optional autonomous social posting (Tier 2)

Only once experiments and metrics are trustworthy. GoDaily posts **once
per platform per weekday** (the `/api/social` cron, 11:00–11:50 UTC,
weekdays — `pkg/services/social`). The rule for extra growth posts:
**never a second post on a day that already carries the digest post** —
which today resolves to **weekends only**. Enabling this needs a one-time
reviewed PR adding a small, fixed number of weekend-only *growth slots*
to the scheduler, hard-capped (start at ≤2 posts/week). The cap and the
no-collision rule live in scheduler code — a misbehaving prompt cannot
over-post. Whether weekend posts earn engagement from a weekday audience
is itself a measured experiment.

## Guardrails

- **Tier 1** — the agent pushes only to `claude/*` branches; PRs open as
  **draft** and never auto-merge. The prompt forbids schema migrations
  and infra/CI changes unless explicitly flagged, and caps diff size.
- **Tier 2** — the per-week post cap and no-collision rule are enforced
  in scheduler code, not the prompt. Every queued post is reported in
  Slack so a human can delete it.
- **Tier 3** changes nothing and needs no guardrail.
- Deterministic jobs (Phases 1–2) involve no LLM and need no AI gate.

## Minimum viable loop

The four highest-ROI pieces, in order: `email_events`, `experiments`,
`prompt_versions`, and `growth-digest` — i.e. Phases 1 → 2 → 3. That
delivers a measured, memory-backed weekly analyst before any autonomous
behaviour exists. Phases 4–6 layer on afterward; Phase 6 is optional.

## Decisions

1. **Prompts stay in Go code.** Git is the source of truth;
   `prompt_versions` is an auto-populated registry that snapshots prompt
   text. Prompt edits flow through reviewed PRs (Phase 5) — keeping the
   human-merge gate that a DB-stored prompt would bypass.
2. **No auto-merge, ever.** Every code and prompt change is reviewed and
   merged by a person.
3. **The weekly analyst is a plain cron + one LLM call** — no Routine, no
   MCP — until Phase 5 needs agentic PR drafting.
4. **Cadence is split:** daily ingestion and aggregation, weekly
   analysis.
5. **Stats: email + Bluesky + Mastodon first.** Bluesky and Mastodon
   expose engagement on open APIs. LinkedIn analytics
   (`organizationalEntityShareStatistics`) are reachable for the org page
   but carry API-versioning overhead — deferred to v2.
6. **Autonomous social posting is deferred to Phase 6**, behind
   structural caps and the no-collision rule.
7. **Triggers are scheduled only** — no event-driven runs. Simpler to
   reason about and to rate-limit.
