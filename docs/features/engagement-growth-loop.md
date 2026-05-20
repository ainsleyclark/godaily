# Engagement-Driven Growth Loop

A closed feedback loop that lets GoDaily improve itself: monitor real
engagement, have an AI agent propose one data-backed change a week, open
a pull request for it, and surface that as a Slack suggestion with a PR
link. A human reviews, refines via chat, and merges. **The agent never
merges code** — the human is always the gate.

## Why this exists

GoDaily already ships a daily digest and auto-posts to social. What it
lacks is a measurement-to-improvement loop: today there is no first-party
data on what content lands, and no mechanism to turn that data into
concrete, reviewable changes. This plan builds that loop.

## Principles

- **Human merges, agent proposes.** Every change lands as a PR a person
  reviews. Nothing auto-merges.
- **Decisions are grounded in first-party data**, not model intuition.
- **Proposals reference prior outcomes.** Each suggestion cites the
  measured result of an earlier experiment, so advice compounds instead
  of looping back to generic tips.
- **Reuse existing infrastructure** — the Slack gateway
  (`pkg/gateway/slack`), social adapters (`pkg/gateway/social/*`), Vercel
  crons, Resend, and the sqlc + goose migration stack.

## Architecture at a glance

```
 ┌─ email events (Resend webhooks) ─┐
 ├─ social post stats (LinkedIn/    │
 │  Bluesky/Mastodon read-back)     ├─► metrics in DB ─► `godaily growth-digest`
 └─ experiments ledger ─────────────┘                         │
                                                              ▼
                                              weekly Claude Code Routine
                                                              │
                            drafts one change on a `claude/*` branch
                                                              │
                                      opens a draft PR (never merges)
                                                              │
                              posts Slack message: suggestion + PR link
                                                              │
                       human reviews → refines via chat → merges the PR
                                                              │
                                    outcome recorded in experiments ledger
                                                              └──► loop
```

## Phase 1 — Email analytics (data foundation)

Implements `docs/features/email-analytics.md`.

- **New endpoint:** `POST /api/webhooks/resend` — a Vercel function at
  `api/webhooks/resend.go`. Public but signature-verified (Resend uses
  Svix-style signing headers; verification is new code — the existing
  `pkg/gateway/hook` package only does *outbound* heartbeats/deploy
  hooks).
- **Migration `0006_email_events.sql`:** an `email_events` table —
  `(issue_id, subscriber_id, event_type, url, occurred_at)`.
- **New store package** `pkg/store/emailevents` with sqlc queries for the
  per-issue and per-link aggregates.
- **Event handling:**
  - `email.clicked` — record the URL clicked. **This is the primary
    signal.**
  - `email.opened` — stored, but flagged unreliable: Apple Mail Privacy
    Protection pre-fetches images and inflates opens for a large share of
    subscribers. Never make a decision on opens alone.
  - `email.bounced` / `email.complained` — mark the subscriber inactive
    or unsubscribe, reusing `pkg/services/subscriber`.

**Deliverable:** per-issue and per-link engagement is queryable.
**Tradeoff:** opens are noise; the loop weights click-through rate (CTR),
unsubscribes, and complaints.

## Phase 2 — Social engagement ingestion

GoDaily already posts via `pkg/gateway/social/{linkedin,bluesky,mastodon}`
and records each post in `social_posts` (migration `0005`). This phase
adds a **read-back** step.

- For each recent `social_posts` row, fetch engagement stats for *our own
  post* from the platform and store them in a new `social_post_stats`
  table (**migration `0007`**), keyed by `social_posts.id`, capturing
  likes/reposts/comments/impressions and a fetched-at timestamp.
- Run it on a new Vercel cron (e.g. daily), re-fetching posts from the
  last N days so late engagement is captured.

**Tradeoff:** platform parity is uneven. Bluesky (`getPostThread`) and
Mastodon (status endpoint) expose like/repost counts on open APIs —
easy. LinkedIn only exposes share statistics for **organization-page**
posts, not personal-profile posts. Recommendation: ship Bluesky +
Mastodon first; treat LinkedIn as best-effort and only viable if GoDaily
posts from a company page.

## Phase 3 — Experiments ledger

The piece that makes the loop compound rather than repeat itself.

- **Migration `0008_experiments.sql`:** an `experiments` table —
  `(id, hypothesis, change_pr_url, shipped_at, metric_name,
  baseline_value, result_value, status)` where `status` moves through
  `proposed → shipped → measured → rejected`.
- When a growth PR merges, it is linked to its experiment row. After
  ~2 weeks of post-merge data, the loop fills in `result_value`.
- Without this ledger the agent re-derives generic advice every week.
  With it, a proposal reads: *"subject-line teasers (shipped wk 12)
  lifted CTR 1.2pp; next, try X."*

## Phase 4 — `growth-digest` CLI

- A new `godaily growth-digest` command (`cmd/godaily`) aggregates the
  last 4 weeks of `email_events` + `social_post_stats` + open/measured
  `experiments` into a single Markdown/JSON report.
- The agent reads this report instead of touching the database directly
  — so the Routine's environment never needs DB credentials.
- The same command can post a plain weekly metrics snapshot to Slack via
  the existing `pkg/gateway/slack` client — a no-AI sanity baseline,
  useful even before Phase 5 exists.

## Phase 5 — The growth Routine (the loop itself)

A **Claude Code Routine** — a saved, cloud-run configuration that fires
on a schedule without anyone's machine being on. Scheduled weekly.

**Routine prompt, in essence:**

1. Run `godaily growth-digest` and read the report.
2. Pick the single highest-ROI experiment justified by the data and the
   experiments ledger. Candidate levers already exist in
   `docs/features/synth-ideas.md` (subject lines, TL;DR intro, semantic
   dedup) and `docs/features/weekly-roundup.md`.
3. Implement it on a `claude/*` branch.
4. Open a **draft** PR. **Do not merge.**
5. Post a Slack message: a one-paragraph suggestion + the PR link.

**The Slack step — two options:**

- **Option A (recommended): reuse the existing Slack gateway.** Add a
  small `godaily growth-notify --pr <url> --summary "..."` subcommand
  that posts through `pkg/gateway/slack`. The Routine knows the PR URL
  once it creates the PR, so it just calls this command. No new
  integration to configure; consistent with how GoDaily already Slacks.
- **Option B: the Slack MCP connector.** The Routine session posts to
  Slack via an MCP connector. More flexible (threaded replies) but
  requires linking a Slack account and adding the connector to the
  Routine config.

**Refinement via chat:** a Routine run *is* a Claude Code session, and
the session is resumable. Replying continues that same session — you
iterate on the PR conversationally ("make the copy punchier", "also
update the txt template") and the agent pushes follow-up commits. The PR
stays a draft until you mark it ready and merge.

## Why Routines (and where not to use them)

| | Routine | `/loop` skill | GitHub Actions / Vercel cron |
|---|---|---|---|
| Runs in cloud, machine off | yes | no | yes |
| Survives restarts, durable | yes | no (session-scoped) | yes |
| Can do agentic PR drafting | yes | yes (if session open) | no |
| Opens PRs, never merges | yes (`claude/*` by default) | n/a | n/a |
| Min interval | 1 hour | 1 minute | cron |

**Recommendation:** use deterministic schedulers (Vercel cron / GitHub
Actions) for the *data jobs* — Phases 1–4 ingestion and aggregation are
plain code and don't need an AI session. Use a **Routine only for
Phase 5**, the agentic step that reads, reasons, drafts, and proposes.
`/loop` is the wrong tool here — it needs an open interactive session.

## Guardrails

- The Routine pushes only to `claude/*` branches and never merges; PRs
  open as **draft**.
- The prompt scopes each run to **one experiment**, forbids schema
  migrations and infra/CI changes without explicitly flagging them in
  the PR description, and caps the diff size.
- Routines have a per-account daily run cap — a weekly cadence is well
  within it.

## Suggested sequencing

The minimum viable loop is **email-only**: Phases 1 → 3 → 4 → 5. Phase 2
plugs in afterward as an additional data source feeding the same
`growth-digest` report. Ship 1, 3, 4, 5 first; add 2 once the loop has
proven it produces useful PRs.

## Open questions to resolve before building

1. **Autonomy:** always human-merge (the default in this plan), or
   eventually let very low-risk changes — pure copy tweaks — become
   auto-merge-eligible once the loop has a track record?
2. **Slack delivery:** Option A (existing gateway via CLI) or Option B
   (MCP connector)?
3. **LinkedIn:** does GoDaily post from a personal profile or a company
   page? This decides whether LinkedIn stats are reachable at all.
4. **Cadence:** weekly, or fortnightly to give each experiment more time
   to accumulate measurable data?
