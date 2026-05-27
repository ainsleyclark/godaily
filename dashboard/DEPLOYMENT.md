# Dashboard Deployment

The dashboard is a standalone SvelteKit SPA (built with `@sveltejs/adapter-static`). It's intentionally not deployed yet — this doc lays out the two paths and the trade-offs so we can pick one before shipping.

## Option A — Served by the Go app at `/dashboard` (recommended)

Build the SPA to static files and serve them from the existing Go server.

```bash
cd dashboard && pnpm install && pnpm build  # outputs to dashboard/build/
```

Then, in the Go app:

1. Embed `dashboard/build/` via `//go:embed`, or copy it into `web/dist/dashboard/` as part of `bin/build.sh`.
2. Add a route group in `pkg/api/mux` that serves the static files under `/dashboard`, with `/dashboard/*` falling back to `index.html` so client-side routing works.
3. No `vercel.json` changes — the existing Go function handles it.

**Pros**
- Single Vercel project, single domain (`godaily.dev/dashboard`).
- No CORS — the dashboard calls `/api/metrics/*` on the same origin.
- One deploy. One set of secrets.

**Cons**
- Dashboard rebuilds ship with the main app.
- Requires a small Go change (~20 lines) to mount the static handler.

## Option B — Separate Vercel project

Create a second Vercel project rooted at `dashboard/`.

- Set `PUBLIC_API_BASE_URL=https://godaily.dev`.
- Deploys to e.g. `dashboard.godaily.dev`.

**Required backend changes**
- CORS allowlist on the metrics endpoints (currently none): allow the dashboard origin and `Authorization` request header. Without this, browsers will block every metrics call.
- Consider moving `APISecret` into the dashboard build env, or keep using the manual sign-in (current implementation).

**Pros**
- Independent deploy cadence.
- Full Vercel feature set (preview URLs, edge functions, etc.).

**Cons**
- CORS surface area on the API.
- Two projects, two sets of env vars, two builds to monitor.
- The bearer token sits in localStorage on a different origin — slightly larger blast radius if XSS lands.

## Recommendation

**Option A for production**, Option B is fine for short-lived staging. The metrics dashboard is internal-facing and changes will be infrequent enough that shipping it alongside the main app is the simplest path.

## Local development

Regardless of deployment target:

```bash
cd dashboard
cp .env.example .env
pnpm install
pnpm dev   # http://localhost:5173
```

The Vite dev server proxies `/api/*` to `https://godaily.dev`, so you can develop against production data without CORS issues. Paste the production `APISecret` on the login screen to authenticate.
