# Vercel Routing Notes

## Metrics endpoints: doubled-path issue

`@vercel/go` maps each `.go` file to a route using the full file path (not just the directory).
Because metrics handlers live in subdirectories that mirror their filename — e.g.
`api/metrics/summary/summary.go` — Vercel exposes them at the doubled path
`/api/metrics/summary/summary` rather than the clean `/api/metrics/summary`.

This is fixed with `vercel.json` rewrites that map the public clean URLs to the actual function
paths. See the `rewrites` array in `vercel.json`.

## Alternative: single metrics handler

Consolidate all metrics handlers into one `api/metrics.go` with an internal `http.ServeMux`
routing on the path suffix. One Vercel function handles all `/api/metrics/*` traffic — clean URLs
with no rewrites needed. Worth considering if the rewrite list grows or if cold-start isolation
per endpoint stops being useful.
