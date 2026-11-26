# Autopilot log

One entry per run, newest at the bottom. Written by the nightly autopilot
(`.claude/skills/autopilot/SKILL.md` on `main`). Append only: never edit or
delete an earlier entry.

### Run 2026-11-26
Chose: This was the first run, so I created the journal. Production was healthy, no loops were due (L1 is due 27 Nov) and there were no steers, so I went to bugs. #317 ranks first, and part 2 (summary rates) was the one piece that was small and had a single correct answer.
Did:
- Production checks: Vercel had 0 runtime errors in the last 24h and 1201 info-level logs. `/api/healthz` returned ok. Issue 144 (26 Nov) sent at 08:00:35. There's no draft because this run started at 13:37 UTC, after the send.
- Created the journal issue (#333).
- Opened draft PR #334, `fix: Compute headline open and click rates per delivery (#317 part 2)`. A failing test came first. `go test ./...` passes and golangci-lint v2.15.0 reports 0 issues. Added loop L4.
- Added evidence to #319: 3 of the 4 Mastodon titles in issue 144 are cut mid-word, and a job-seeking post (item 29210) was tagged `discussion`. https://github.com/ainsleyclark/godaily/issues/319#issuecomment-7193757380
- CI on `main` (#332) is all green. There are no open Dependabot PRs.
Deferred:
- #319 truncation and job filtering: both need a product call, so no PR yet.
- #317 parts 1, 3 and 4: bot clicks need a judgement on how to filter them; `posted_at` and impressions are larger changes. I'll pick one up on a later night.
- Improvement review: skipped because metrics are still untrusted and the run budget was spent on the bug. `last_improve_review` stays null, so it can run on a later night.
- Skill nit for later: the preinstalled golangci-lint (v2.5.0, built with go1.25) can't load the config. I installed v2.15.0 with `go install`. sqlc also isn't preinstalled (I installed v1.31.1).
Next due: L1 2026-11-27, L2 2026-11-30, L4 2026-11-30 (if #334 is merged), L3 2026-12-02

### Migration 2026-11-26
State moved from the JSON block in issue #333 to `state.json` on this branch, at Ainsley's request. #333 stays as the readable view and is rewritten from `state.json` each run. Loops L1–L4 were copied unchanged.
