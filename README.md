# autopilot/state

State for GoDaily's nightly autopilot. **This branch is never merged**, and it
holds no code.

| File | What |
|---|---|
| `state.json` | Source of truth: the checks the autopilot is waiting on (`open_loops`), plus flags like `metrics_trusted`. |
| `log.md` | One entry per run: what it chose, did and deferred. Append only. |
| `vercel.json`, `dashboard/vercel.json` | Turn off Vercel deploys for this branch. |

The readable view is the **Autopilot journal** issue (#333), which is rewritten
from `state.json` every run. To change something (for example, drop a check),
comment on #333 and the next run applies it here.

How the autopilot uses this branch: `.claude/skills/autopilot/SKILL.md` and
`docs/features/autopilot.md` on `main`.
