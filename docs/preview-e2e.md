# Preview E2E Testing

## Problem

The preview e2e workflow (`preview-e2e.yaml`) runs against a live Vercel preview deployment
backed by a real persistent database. Each run creates subscribers, collected news items, draft
issues, and sent issues. Because there is no cleanup step, subsequent runs hit stale state and
the pipeline tests fail (e.g. `build` can't create a new draft if one already exists, `send`
assertions land on the wrong issue).

Exposing delete/wipe HTTP endpoints was ruled out — they're a footgun regardless of how they're
gated.

## Chosen approach: per-PR Turso database branch

Each PR gets its own isolated Turso database branch. The branch is created fresh when the PR
opens, used by every preview deployment on that branch, and destroyed when the PR closes.

This means every e2e run starts with a clean, migrated database. No wipe logic, no shared
state between PRs.

### Why not a reset endpoint?

A reset/wipe handler — even behind a secret — is dangerous in production-adjacent code. It was
explicitly ruled out.

### Why not the Vercel build command?

The build command runs *after* Vercel has already snapshotted the environment variables for the
deployment. Writing credentials during `buildCommand` or `installCommand` won't affect the
current deployment's runtime env. This is a confirmed race condition:
<https://github.com/vercel/vercel/discussions/8801>

## Implementation plan

### 1. Disable Vercel auto-deploy for preview branches

In `vercel.json`, add an ignored build step that causes Vercel to skip automatic builds for
preview environments. GitHub Actions will own the deploy instead.

```json
"ignoreCommand": "[ \"$VERCEL_ENV\" != \"preview\" ]"
```

Production deployments are unaffected — Vercel still auto-deploys `main`.

### 2. GitHub Actions workflow: `preview-deploy.yaml`

Triggers on `pull_request` (opened, synchronize, reopened).

Steps:
1. Create a Turso branch named `godaily-pr-{PR_NUMBER}` (idempotent — `|| true` if it already
   exists)
2. Fetch the branch URL and generate an auth token via the Turso CLI
3. Run database migrations against the new branch using Goose
4. Set `TURSO_URL` and `TURSO_AUTH_TOKEN` as Vercel branch-scoped env vars via `vercel env add`
   (scoped to `preview` + the git branch name)
5. Trigger the Vercel deployment via `vercel deploy --prebuilt` or the REST deploy hook
6. The deployment picks up the branch-scoped env vars and connects to the clean DB

Required secrets: `TURSO_API_TOKEN`, `VERCEL_TOKEN`, `VERCEL_ORG_ID`, `VERCEL_PROJECT_ID`

### 3. GitHub Actions workflow: `preview-cleanup.yaml`

Triggers on `pull_request` closed.

Steps:
1. Delete the Vercel branch-scoped env vars (optional, Vercel cleans up preview envs when
   branches are deleted)
2. Destroy the Turso branch: `turso db destroy godaily-pr-{PR_NUMBER} --yes`

### 4. Migrations on the new branch

When a Turso branch is created, it inherits the schema of the parent at branch time. This means
it already has the correct schema and no data — migrations don't need to be re-run unless the
branch was created from an empty parent.

If the parent DB is empty (e.g. first time setup or CI uses a separate parent), run migrations
explicitly after branch creation:

```bash
turso db shell godaily-pr-$PR < <(goose -dir pkg/store/migrations sqlite3 "" dump-sql)
```

Or, more practically, run the app's migration bootstrap against the branch URL:

```bash
TURSO_URL=$BRANCH_URL TURSO_AUTH_TOKEN=$BRANCH_TOKEN \
  go run ./cmd/migrate up
```

The migration command (or however the app runs Goose at startup) must be invoked against the
branch credentials before the Vercel deploy fires.

### 5. Existing `preview-e2e.yaml`

No changes needed. It still triggers on `deployment_status: success` for preview environments.
Because the deployment now points at a clean per-PR database, the tests will always have
consistent starting state.

## Sequence diagram

```
PR push
  │
  ├─ Vercel: sees push, runs ignoreCommand → exits 1 → skips build
  │
  └─ GHA: preview-deploy.yaml
       ├── turso db branch create godaily-pr-{N}
       ├── goose / migrate up (against branch URL)
       ├── vercel env add TURSO_URL (scoped to branch)
       ├── vercel env add TURSO_AUTH_TOKEN (scoped to branch)
       └── vercel deploy → triggers deployment_status: success
                                │
                                └─ GHA: preview-e2e.yaml
                                     └── playwright tests (clean DB)

PR closed
  └─ GHA: preview-cleanup.yaml
       └── turso db destroy godaily-pr-{N}
```

## New secrets required

| Secret | Where | Purpose |
|--------|-------|---------|
| `TURSO_API_TOKEN` | GitHub + Vercel build | Authenticate Turso CLI |
| `VERCEL_TOKEN` | GitHub | Deploy and manage env vars via CLI |
| `VERCEL_ORG_ID` | GitHub | Target correct Vercel org |
| `VERCEL_PROJECT_ID` | GitHub | Target correct Vercel project |

`TURSO_API_TOKEN` must also be available in Vercel's build environment if migrations are run
during the build step rather than from GHA.

## References

- [Vercel discussion: dynamic env vars before build (race condition confirmed)](https://github.com/vercel/vercel/discussions/8801)
- [Vercel discussion: per-preview dynamic env vars](https://github.com/vercel/vercel/discussions/8543)
- [Vercel CLI env docs](https://vercel.com/docs/cli/env)
- [Turso branching](https://docs.turso.tech/features/branching)
