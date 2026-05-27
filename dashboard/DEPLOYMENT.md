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
| Framework Preset | SvelteKit (auto-detected) |
| Build Command | `pnpm build` (default) |
| Install Command | `pnpm install` (default) |
| Output Directory | `build` |
| Ignored Build Step (this project) | `git diff --quiet HEAD^ HEAD .` |

And on the existing **godaily** project (the main app):

| Vercel UI setting | Value |
| --- | --- |
| Ignored Build Step | `git diff --quiet HEAD^ HEAD ':(exclude)dashboard'` |

Both `Ignored Build Step` commands exit 0 (= skip build) when nothing changed in their
respective scope, so dashboard-only pushes don't redeploy the main app, and vice versa.

## CORS

The dashboard is cross-origin from the API, so the Go backend
(`pkg/api/plugs/cors.go`, mounted at `pkg/api/mux/mux.go`) emits CORS headers
for the constants `env.DashboardURL` and `env.AppURL` defined in
`pkg/env/env.go`:

```
https://analytics.godaily.dev
https://godaily.dev
```

To allow a new origin (e.g. a staging URL), edit those constants — the allow-list
is intentionally not an env var. Preview URLs (`*.vercel.app`) aren't in the list;
for previews, run `pnpm dev` locally against prod via the Vite proxy.

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
