# Dashboard Deployment

The dashboard is a static SvelteKit SPA (`@sveltejs/adapter-static`) deployed as its own
Vercel project at `analytics.godaily.dev`, sharing the same git repo as the main app
(`godaily.dev`) but with `dashboard/` as its **Root Directory**.

```
┌─────────────────────────────────┐      ┌───────────────────────────────┐
│  Vercel project: godaily        │      │  Vercel project: godaily-     │
│  Root: /                        │      │  dashboard                    │
│  Domain: godaily.dev            │      │  Root: /dashboard             │
│  Build: bin/build.sh            │      │  Domain: analytics.godaily.dev│
│  Serves /api/* + marketing site │      │  Static SvelteKit SPA         │
└────────────┬────────────────────┘      └───────────────┬───────────────┘
             │                                           │
             │  Bearer + CORS                            │
             └◀── /api/metrics/* ────────────────────────┘
```

Why two projects (and not e.g. `godaily.dev/dashboard`): independent deploy cadence, full
Vercel preview URLs for the SPA, and the dashboard doesn't bloat the main Go binary.

## Why two `vercel.json` files

Each Vercel project reads `vercel.json` from its **Root Directory**, not from the
repo root. The dashboard project (Root Directory `dashboard/`) must have its own
`dashboard/vercel.json` defining `installCommand`, `buildCommand`, `framework`,
and `outputDirectory` — otherwise Vercel falls back to the repo-root
`vercel.json`, which is configured for the main Go app (`bash bin/build.sh`,
`cd web && pnpm install`, crons, Go function definition…) and breaks the dashboard
build. UI "Override" toggles only override Vercel's auto-detected defaults — they
do not override an explicit `vercel.json`, so the per-project file is the only
way to override.

## One-time setup

Run the helper, then complete the dashboard-only steps in the Vercel UI:

```bash
bin/setup-vercel-dashboard.sh
```

What it does (via Vercel CLI):

- `vercel link` to create or attach `dashboard/` to a Vercel project.
- `vercel env add PUBLIC_API_BASE_URL production` → `https://godaily.dev`.
- `vercel domains add analytics.godaily.dev`.

What you have to click through (CLI can't set these):

| Vercel UI setting | Value |
| --- | --- |
| Root Directory | `dashboard` |
| Framework Preset | SvelteKit (set by `dashboard/vercel.json`) |
| Build Command | (leave default — set by `dashboard/vercel.json`) |
| Install Command | (leave default — set by `dashboard/vercel.json`) |
| Output Directory | (leave default — set by `dashboard/vercel.json`) |
| Ignored Build Step (this project) | `git diff --quiet HEAD^ HEAD .` |

And on the existing **godaily** project (the main app):

| Vercel UI setting | Value |
| --- | --- |
| Ignored Build Step | `git diff --quiet HEAD^ HEAD ':(exclude)dashboard'` |

Both `Ignored Build Step` commands exit 0 (= skip build) when nothing changed in their
respective scope, so dashboard-only pushes don't redeploy the main app, and vice versa.

## CORS

The dashboard is cross-origin from the API. The Go backend
(`pkg/api/plugs/cors.go`, mounted at `pkg/api/mux/mux.go`) emits
`Access-Control-Allow-Origin: *` on every response and answers OPTIONS
preflight with 204.

The wildcard is safe because every protected route gates on a Bearer token —
a hostile browser can preflight but can't read anything without the secret —
and we never use cookies. Non-browser callers (curl, scripts, MCP/Claude
skills) ignore CORS entirely. The wildcard simply removes friction for any
browser-based tool we might point at the API later (preview URLs, staging,
ad-hoc dashboards).

Bearer auth (`Authorization: Bearer <APISecret>`) is unchanged: the dashboard
verifies the secret by calling `/api/metrics/summary` at login and stores it in
`localStorage`.

## Local development

```bash
cd dashboard
cp .env.example .env
pnpm install
pnpm dev   # http://localhost:5173
```

The Vite dev server proxies `/api/*` to `https://godaily.dev`, so you can develop
against production data without engaging CORS. Paste the production `APISecret`
on the login screen to authenticate.

## Alternative considered: serve via Go at `/dashboard`

We considered embedding the built SPA into the Go binary and serving it from
`godaily.dev/dashboard`. Single project, no CORS, simpler — but it couples the
dashboard's deploy cadence to the main app and ships SPA assets through the Go
serverless function. We picked the separate-project path instead.
