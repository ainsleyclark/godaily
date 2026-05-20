# Engagement-Driven Growth Loop

A closed feedback loop that lets GoDaily improve itself: monitor real
engagement, and once a week have an AI agent take **data-backed actions**
to grow it. Not every action is the same — a code change is reviewed as a
draft pull request before a human merges it, while extra social posts
publish autonomously but only inside limits enforced by code. The gate
matches the blast radius of the action; see *Action tiers* below.

## Why this exists

GoDaily already ships a daily digest and auto-posts to social. What it
lacks is a measurement-to-improvement loop: today there is no first-party
data on what content lands, and no mechanism to turn that data into
concrete, data-backed actions. This plan builds that loop.

## Principles

- **The gate matches the blast radius.** Code changes are always reviewed
  and merged by a person — nothing auto-merges. Public content (social
  posts) publishes autonomously, but only within hard limits that live in
  code, not in the agent's prompt.
- **Decisions are grounded in first-party data**, not model intuition.
- **Proposals reference prior outcomes.** Each suggestion cites the
  measured result of an earlier experiment, so advice compounds instead
  of looping back to generic tips.
- **Reuse existing infrastructure** — the Slack gateway
  (`pkg/gateway/slack`), social adapters (`pkg/gateway/social/*`), Vercel
  crons, Resend, and the sqlc + goose migration stack.

## Architecture at a glance

```
 ┌─ email events (Resend webhooks) ──┐
 ├─ social post stats (read-back) ───┤
 └─ experiments ledger ──────────────┴─► metrics in DB ─► `godaily growth-digest`
                                                                  │
                                                weekly Claude Code Routine
                                                                  │
                       ┌──────────────────────┼──────────────────────┐
                       ▼                      ▼                      ▼
              Tier 1: code change    Tier 2: social posts    Tier 3: report
              draft PR, never        queued into capped,     Slack metrics
              merged → human         non-colliding slots,    snapshot,
              reviews & merges       publish autonomously    no gate
                       └──────────────────────┼──────────────────────┘
                                                  │
                                  outcomes recorded in experiments ledger
                                                  └──► loop
```

## What the loop does — action tiers

Each weekly run can produce one or more **actions**. Actions are not
interchangeable: a code change and a published post carry very different
risk, so each tier gets the lightest gate that is still safe.

| Tier | Action | What it touches | Gate | Mechanism |
|------|--------|-----------------|------|-----------|
| 1 | Code change | digest logic, email/site templates, scoring, social copy generation | **Human reviews & merges** a draft PR | `claude/*` branch → draft PR |
| 2 | Extra social post | a net-new post on Bluesky / Mastodon / LinkedIn | **Autonomous, within code-enforced limits** | queued into capped, non-colliding `social_posts` slots |
| 3 | Metrics report | nothing — information only | None | Slack snapshot |

Tier 1 is the slow, deliberate lever — anything that changes how GoDaily
*works*. Tier 2 is the fast lever — more reach without code review — and
its safety comes entirely from the limits being structural (see *Social
posting: the collision constraint*). Tier 3 is just visibility.

A future Tier-2 action, noted but out of scope for v1: replying to
comments on GoDaily's own social posts — same tier, same gating model.

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

**All three platforms are in scope.** GoDaily posts daily to Bluesky,
Mastodon, and LinkedIn, and the LinkedIn client publishes to an
**organisation page** (`pkg/gateway/social/linkedin/linkedin.go` —
`urn:li:organization` author, `w_organization_social` scope). That
matters: org-page posts *do* expose engagement, so LinkedIn stats are
reachable — unlike personal-profile posts, which are not.

- **Bluesky** (`getPostThread`) and **Mastodon** (status endpoint)
  expose like/repost/reply counts on open APIs — straightforward.
- **LinkedIn** read-back needs the token to also carry
  `r_organization_social`, then uses `organizationalEntityShareStatistics`
  (impressions, clicks, engagement rate) plus the Social Actions API
  (likes/comments) for the org's shares.

**Tradeoff:** LinkedIn's API versions retire ~yearly (see the
`LINKEDIN_API_VERSION` note in the client), so the read-back code needs
the same version-pinning discipline as the posting code.

## Phase 3 — Experiments ledger

The piece that makes the loop compound rather than repeat itself.

- **Migration `0008_experiments.sql`:** an `experiments` table —
  `(id, tier, hypothesis, change_ref, shipped_at, metric_name,
  baseline_value, result_value, status)`. `tier` records which action
  tier this was; `change_ref` points at a PR URL (Tier 1) or a batch of
  `social_posts` rows (Tier 2). `status` moves through
  `proposed → shipped → measured → rejected`.
- A Tier-1 experiment is linked when its PR merges; a Tier-2 experiment
  the moment its posts are queued. After ~2 weeks of follow-on data the
  loop fills in `result_value`.
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
  useful even before Phase 5 exists. (This deterministic snapshot runs as
  a cron job, so it uses the Go Slack gateway directly; the *agentic*
  suggestion in Phase 5 posts differently — see below.)

## Phase 5 — The growth Routine (the loop itself)

A **Claude Code Routine** — a saved, cloud-run configuration that fires
on a schedule without anyone's machine being on. Scheduled weekly, and
triggered purely by that schedule — no event-driven runs (see
*Decisions*).

**Each weekly run:**

1. Run `godaily growth-digest` and read the report.
2. Decide this week's actions across the tiers — typically one Tier-1
   experiment, optionally a small batch of Tier-2 posts, always the
   Tier-3 snapshot. Every choice must be justified by the data and the
   experiments ledger. Candidate Tier-1 levers already exist in
   `docs/features/synth-ideas.md` (subject lines, TL;DR intro, semantic
   dedup) and `docs/features/weekly-roundup.md`.
3. **Tier 1:** implement the change on a `claude/*` branch and open a
   **draft** PR. **Never merge.**
4. **Tier 2:** draft the posts and queue them through the guarded social
   path, which rejects anything over the cap or in a colliding slot —
   the agent cannot override this.
5. Record each action in the experiments ledger.
6. Post one Slack message summarising the run: the Tier-1 PR link, the
   Tier-2 posts queued (for visibility, not approval), and the Tier-3
   metrics snapshot.

**The Slack step — via the Slack MCP connector.** The Routine session
posts the suggestion and PR link to Slack through the Slack MCP
connector. This keeps the message inside the agent's session, so a reply
in the Slack thread can feed straight back into the refinement loop.

Setup required once: link a Slack account to Claude Code and add the
Slack connector to the Routine's configuration. The connector's traffic
is routed through Anthropic's servers, so no network-allowlist change is
needed in the Routine environment.

(The existing `pkg/gateway/slack` Go client is still used for the
deterministic Phase 4 snapshot — that runs as a cron job outside any
agent session and so can't use an MCP connector.)

**Refinement via chat:** a Routine run *is* a Claude Code session, and
the session is resumable. Replying continues that same session — you
iterate on the Tier-1 PR conversationally ("make the copy punchier",
"also update the txt template") and the agent pushes follow-up commits.
The PR stays a draft until you mark it ready and merge. (Refinement
applies to Tier 1; a Tier-2 post that lands wrong is deleted rather than
refined — that is the cost of the autonomous tier, and the reason its
limits are strict.)

## Social posting: the collision constraint

GoDaily already posts **once per platform per weekday** — the
`/api/social` cron picks a jittered 10-minute slot between 11:00–11:50
UTC, Monday to Friday, and skips weekends entirely
(`api/social.go`, `pkg/services/social`). Tier-2 posts must not erode
that rhythm: a second post stacked onto a day that already carries the
digest post reads as spam and competes with GoDaily's own best content
of the day.

The rule: **never a second post on a day that already carries the daily
digest post.** Today that resolves to **weekends only** — the one slot
currently empty.

Two consequences:

1. **A one-time Tier-1 prerequisite.** The social handler hard-skips
   weekends today. Supporting Tier-2 posts means a code change first:
   extend the scheduler with a small, fixed number of *growth slots*
   that are weekend-only and capped (start at ≤1 per platform per
   weekend day — i.e. ≤2 posts/week). The cap and the no-collision rule
   live in this code. It ships as a normal reviewed PR.
2. **The agent fills a queue; it does not post freely.** Once the slots
   exist, the Routine drafts posts and inserts them as scheduled
   `social_posts` rows; the existing idempotent posting pipeline
   publishes them. If the Routine tries to exceed the cap or target a
   colliding day, the guarded path rejects the write. "Autonomous within
   limits" means the *limits are structural* — a misbehaving prompt
   cannot over-post.

Whether weekend posts actually earn engagement from a weekday-oriented
developer audience is itself unknown — so it is a measured experiment
like any other. Phase 2 stats compare growth-slot posts against daily
posts; if weekend posts underperform, the loop stops scheduling them.

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

- **Tier 1:** the Routine pushes only to `claude/*` branches and never
  merges; PRs open as **draft**. The prompt forbids schema migrations and
  infra/CI changes unless explicitly flagged in the PR description, and
  caps the diff size.
- **Tier 2:** the per-week post cap and the no-collision rule are
  enforced in the social-scheduler code, not the prompt. Every queued
  post is reported in the Slack summary so a human can delete it before
  or shortly after it publishes.
- **Tier 3** changes nothing and needs no guardrail.
- Routines have a per-account daily run cap — a weekly cadence is well
  within it.

## Suggested sequencing

The minimum viable loop is **email-only, Tier 1 + Tier 3**: Phases
1 → 3 → 4 → 5. Phase 2 (social ingestion) plugs in afterward as another
data source feeding the same `growth-digest` report.

Tier-2 social posting has two prerequisites before it can run: Phase 2
(so its posts can be measured) and the one-time social-scheduler change
described above (so the non-colliding slots exist). Recommended order:
ship the Tier-1/Tier-3 loop first; add Phase 2; then enable Tier 2 once
the loop has shown it produces useful PRs.

## Decisions

These were settled before the doc was finalised:

1. **Autonomy — always human-merge.** No auto-merge tier, ever. Every
   change, however small, is reviewed and merged by a person.
2. **Slack delivery — the Slack MCP connector.** Keeps the suggestion
   inside the agent's session so thread replies feed the refinement
   loop. Requires linking a Slack account and adding the connector to
   the Routine config (one-time setup).
3. **LinkedIn — in scope.** GoDaily posts to a LinkedIn organisation
   page, so share statistics are reachable; all three social platforms
   feed the loop.
4. **Cadence — weekly.** One run per week.
5. **Content gate — autonomous within code-enforced limits.** Tier-2
   social posts publish without per-post approval, but the volume cap
   and the no-collision-with-the-daily-post rule live in the
   social-scheduler code — the agent fills a constrained queue, it never
   posts ad hoc. Initial cap: ≤2 growth posts per week, weekend slots
   only.
6. **Triggers — scheduled only.** The loop runs on its weekly schedule;
   no event-driven runs. Simpler to reason about and to rate-limit.
