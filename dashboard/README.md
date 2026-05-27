# GoDaily Dashboard

Mission-control SPA for the GoDaily metrics API. SvelteKit + Tailwind v4, deployed as a static build (`@sveltejs/adapter-static`).

## Quick start

```bash
cp .env.example .env
pnpm install
pnpm dev
```

Open <http://localhost:5173>, you'll be redirected to `/login`. Paste the production `APISecret` to sign in. The Vite dev server proxies `/api/*` → `https://godaily.dev` so CORS is a non-issue locally.

## Build

```bash
pnpm build      # static SPA in ./build
pnpm preview    # serve the build locally
```

## Layout

- `src/lib/api/` — typed client + JSON shapes
- `src/lib/stores/` — auth (localStorage-backed bearer) + reactive date range
- `src/lib/components/` — KPI card, date picker, chart components, table components, generic UI primitives
- `src/routes/+page.svelte` — mission control landing
- `src/routes/login/+page.svelte` — password gate

## Auth

The dashboard is a single-admin tool. On `/login`, the user pastes the API secret, which we verify by calling `/api/metrics/summary` with it as a Bearer token. On success it's stored in `localStorage["godaily_api_secret"]` and attached to every subsequent request. On 401 we clear it, toast, and redirect back to `/login`.

## Deployment

See [DEPLOYMENT.md](./DEPLOYMENT.md).
