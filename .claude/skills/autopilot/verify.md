# Verification recipes

How to prove a fix worked in production. Prefer checks a script or query can
answer over checks that need judgement. Record the **evidence** (ids, counts,
timestamps, quotes), not just "looks good".

## When code is actually running

A merge does not change production until it is deployed **and** the cron that
runs the changed code has fired.

1. **Deployed?** `mcp__Vercel__list_deployments` with `projectId:
   prj_cSwqZJAaVZyd65fs6tPGreEggATj`, `teamId: team_g5WW9l3cVUo50emtwDIA57oA`,
   `target: production`. Find the deployment whose `githubCommitSha` is the
   merge commit, and note its `created` time (ms since epoch). A newer
   deployment includes it too.
2. **Exercised?** Compare that time with the cron that runs the code
   (`vercel.json`, all UTC, every day; handlers skip weekends where noted):

| Cron (UTC) | Path | What it does |
|---|---|---|
| 01:00 | `/api/digest/collect/` | Fetch sources, store items |
| 02:00 | `/api/digest/build/` | Build draft issue **and draft social posts** (featured + rotation) |
| 03:00 | `/api/social/metrics/` | Refresh social likes/impressions |
| 08:00 | `/api/digest/send/` | Send the email |
| 09:00 | `/api/digest/nudge/` | Confirmation nudges |
| 10:00 | `/api/social/publish/featured/` | Publish featured drafts (weekdays) |
| 14:00 | `/api/social/publish/rotation/` | Publish rotation: **recap Mon, community Wed, new_source/spotlight/cta Fri** |
| Fri 15:00 | `/api/metrics/roundup/` | Weekly Slack roundup |

Social copy is **drafted at 02:00** and only published later. So a fix to
drafting first shows up in posts drafted after the deploy, not in posts
published after it. `posted_at` currently stores the draft time (#317).

If the code hasn't been exercised yet: the result is **can't tell yet**. Set
`due` to the first date it will be, and say which cron you're waiting on.

## Data sources

| Need | Tool |
|---|---|
| Social post text and engagement | `mcp__GoDaily__social_post_metrics` (`from`/`to`, `limit` ≤ 100) |
| Today's draft before the 08:00 send | `mcp__GoDaily__get_issue` with `key: "draft"` |
| A sent issue and its items | `mcp__GoDaily__get_issue` (`latest`, id or `YYYY-MM-DD`) |
| Items, tags, sources | `search_items`, `get_item`, `list_item_facets` |
| Email engagement | `get_issue_engagement`, `list_issue_engagement`, `metrics_summary` (**untrusted until #317 is verified**) |
| Subscribers | `subscriber_growth` |
| Error clusters | `mcp__Vercel__get_runtime_errors` (`since: "24h"`, max 7d) |
| Log lines | `mcp__Vercel__get_runtime_logs`. Use `group_by: "level"` or `requestPath` for counts. For lines, narrow with `query`, `level` or `deploymentId`, because wide windows time out after 5s. |
| Health | `curl -sL https://godaily.dev/api/healthz` (public, no key). Expect `"message":"ok"`. The `-L` matters: it 308-redirects to the trailing-slash URL. |

Logs: before webkit v0.14.3 (deployed 26 Nov 2026), ERROR records were not
labelled as errors. "0 errors" from before that date proves nothing.

## Social copy checks

Save the `social_post_metrics` result to a file (the harness does this for
large results), then run:

```bash
python3 -I .claude/skills/autopilot/evals/social_copy.py <file> --since YYYY-MM-DD
python3 -I .claude/skills/autopilot/evals/social_copy.py <file> --since YYYY-MM-DD --json
```

`--json` lists each offending post id. Use `--since` with the first draft date
after the deploy, so old posts don't dilute the result.

## Worked example: the #330 and #331 loops

- **#330 featured footer**: posts with `kind: featured` drafted on or after
  27 Nov 2026. Pass: every platform's text has a `Subscribe:` line whose link
  carries `utm_campaign=featured`, and nothing is truncated with `…`.
- **#330 recap**: the Monday 30 Nov recap. Pass: no `…`, ends with the
  Subscribe line (`utm_campaign=recap`), no `jobs`-tagged item (check the
  linked items' tags), and within the limits (the eval's `over_limit` is empty).
- **#331 community**: the Wednesday 2 Dec rotation. Pass: the community post
  names a specific event with a date, time and event link, **or** no community
  post exists that day (skipped because nothing was timely). Fail: a
  directory-style entry like post 628 ("Tokyo Go (Tokyo, Japan) - …").
