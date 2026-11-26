# Autopilot: nightly self-healing and improvement

Every weeknight, a Claude Code scheduled task runs the `autopilot` skill
(`.claude/skills/autopilot/SKILL.md`) against this repository. It:

1. **Heals**: finds production problems (Vercel logs, health check, tonight's
   draft before the 08:00 send), opens `bug` issues with evidence, and opens
   small **draft** PRs for bugs that are safe to fix without a decision.
2. **Verifies**: checks in production, in a later run, that each merged fix
   actually worked, and records the evidence on the PR.
3. **Improves** (about weekly): scores the AI copy with offline evals and
   proposes up to three ideas for reach, data or content quality.
4. **Reports**: DMs Ainsley on Slack when there's news, plus a Friday summary.

GitHub is the source of truth. State is the **Autopilot journal**
issue plus labels. Ainsley always merges.

## Why it's built this way

Each choice comes from published experience of running agents unattended:

| Choice | Why |
|---|---|
| State in a JSON checklist the agent may only tick (`open_loops`), plus a run log | Anthropic, [*Effective harnesses for long-running agents*](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents): a progress log and a JSON feature list let a fresh session pick up the work. The model is less likely to rewrite JSON than Markdown. |
| A fix is never verified in the run that wrote it | Anthropic, [*Harness design for long-running apps*](https://www.anthropic.com/engineering/harness-design-long-running-apps): agents "confidently praise" their own work, and separating generator from evaluator helps. A fresh session is the cheapest separation. |
| Every PR states how it will be verified **before** merge | The same post's "sprint contracts": agree what "done" looks like before writing code. |
| A fixed checklist and priority list, not an AI "manager" | Anthropic, [*Project Vend phase 2*](https://www.anthropic.com/research/project-vend-2): "bureaucracy matters". A same-model supervisor shared the worker's blind spots. |
| Deterministic checks first, and never weaken tests | Spotify Honk ([part 3](https://engineering.atspotify.com/2025/12/feedback-loops-background-coding-agents-part-3)) saw agents delete failing tests to pass CI. Anthropic's [reward-hacking research](https://www.anthropic.com/research/emergent-misalignment-reward-hacking). |
| Pause by labelling the issue and asking, then move on | Runs can't wait for a reply. Agent-initiated stops are a real form of oversight ([*Measuring agent autonomy*](https://www.anthropic.com/research/measuring-agent-autonomy)). |
| Human merges, CI gates | Stripe [Minions](https://stripe.dev/blog/minions-stripes-one-shot-end-to-end-coding-agents-part-2): agent-written, human-reviewed PRs. |
| Offline evals before live A/B tests | About 300 subscribers and about 23 LinkedIn impressions a post can only detect huge effects. Offline evals of the AI copy have as much data as we want. |

## Decisions (Nov 2026)

- **One scheduled task, one run per weeknight.** The skill decides what matters
  using a fixed priority list: broken production, then due verifications, then
  answered steers, then bugs, then (about weekly) improvement review.
- **Ainsley always merges.** No auto-merge.
  - **Revisit around late Dec 2026**: consider auto-merging tiny bug fixes
    once there's a month of track record (PRs merged unchanged, verification
    pass rate, steers that were worth asking). It would need branch protection
    with required checks on `main` first.
- **Always ask first** for changes to voice or tone (any prompt change),
  adding or removing sources, and anything that adds cost. Also schema changes,
  ambiguous fixes, and second failed verifications.
- **Fix the numbers first.** #317 (bot clicks, wrong summary rates,
  `posted_at`, 0 impressions) is the top bug. Until it's verified, the
  autopilot must not judge ideas on engagement numbers (`metrics_trusted:
  false` in the journal).
- **Goal is a scorecard of four**: confirmed subscribers, real reader clicks,
  social reach and content quality. Each experiment picks one primary metric.
  The other three are guardrails it must not hurt.
- **Slack**: a DM only when there's news, plus a Friday summary.

## Guardrails

- **Prompt**: the hard rules in the skill (never merge, deploy, send, post,
  touch secrets, weaken tests, or self-verify).
- **Structural**: `.claude/settings.json` puts GitHub merge and direct file
  writes, pushes to `main`, force pushes, and Vercel and Buffer write tools
  behind `ask` rules. In an unattended run, that stalls the action instead of
  doing it. In your own sessions, you just get a prompt. `GODAILY_API_KEY`
  is empty in cloud sessions, so the state-changing API can't be called.
- **Not yet structural**: `main` has no branch protection
  (`.github/settings.yml`), and runs act as Ainsley's GitHub identity. Turning
  on required checks plus `enforce_admins: true` would make "never push to
  `main`" a hard rule.
- **WIP limit**: no new PRs while 3 or more `autopilot` PRs are open.

## Phase 2: experiments infrastructure (after #317 is verified)

When the metrics can be trusted, the autopilot opens a `needs-steer` issue to
build this. It's a schema change, so it asks first.

- **Definitions** in `pkg/data/experiments.yaml`, embedded like
  `conferences.yaml` and validated in `pkg/data/data_test.go`: `slug`,
  `hypothesis`, `surface` (featured | recap | subject | intro | …),
  `variants`, `assignment: alternate-days`, `primary_metric`,
  `guardrail_metrics`, `start`, `min_days`, `decision_rule`, `issue`. Claude
  proposes experiments by PR, and **merging the PR is the approval**.
- **Assignment** is deterministic by date (alternating days), so every reader
  sees both variants. This gives more signal than splitting about 300
  subscribers.
- **Stamps**: a goose migration adds nullable `experiment_slug` and `variant`
  to `social_posts` and `issues`. The build and draft code writes them.
- **Results**: a read-only MCP tool `experiment_results(slug)`
  (`pkg/mcpserver`) plus an API route and a dashboard view: per-variant
  primary and guardrail metrics, n, and days.
- The autopilot never needs a write key: it defines experiments by PR and
  reads results through the MCP.

This supersedes Phases 3–5 of
[`engagement-growth-loop.md`](engagement-growth-loop.md). Its data model
(`experiments`, `insights`, `prompt_versions`) is replaced by the YAML, the
stamps, GitHub issues and git history.

## Setup

1. Merge the PR that adds the skill.
2. Create the scheduled task (Mon–Fri, about 04:17 UTC, fresh session each run,
   prompt "Run the autopilot skill."). Connectors: GoDaily, Slack, Vercel and
   GitHub only.
3. Optional: add a GitHub **Issue → labeled** trigger to the same task in
   the web UI, so adding `go-ahead` starts a run straight away instead of
   waiting for the night.
4. Run it once with **Run now** and check: a journal issue with loops for
   #330 and #331, a run-log comment, and (because the loops aren't due yet) no
   DM unless it found something.

## Files

| Path | What |
|---|---|
| `.claude/skills/autopilot/SKILL.md` | The procedure |
| `.claude/skills/autopilot/verify.md` | Proving fixes in production: cron table, tools, IDs |
| `.claude/skills/autopilot/evals/social_copy.py` | Deterministic copy checks (stdlib Python) |
| `.claude/skills/autopilot/evals/rubric.md` | LLM-judge rubric for grounding and repeated stories |
| `.github/pull_request_template.md` | Includes the required `## Verification` block |
| `.github/ISSUE_TEMPLATE/experiment.yml` | Pre-registered experiments |
