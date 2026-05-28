# Plan 05 — Progressive Enhancement (optional JS)

**Goal:** make the SSR page feel live — filtering without full reloads,
infinite scroll, and a ⌘K search palette — without breaking the no-JS baseline.

**Depends on:** Plans 02–04 working end-to-end via querystring. This layer is
purely additive; if JS is disabled everything still works via links.

## Approach

The site is near-static Templ with minimal JS (`web/assets/js/`). Keep it light
— vanilla JS or a tiny helper, no SPA framework. Two viable styles:

- **Fetch + swap (recommended):** intercept clicks on filter links and the
  search input, fetch the same URL with an `Accept`/header flag, and swap the
  `.browse__feed` innerHTML. Requires the handler (Plan 02) to optionally render
  **just the feed partial** when the request is an enhancement request (e.g.
  `X-Requested-With` or `?partial=1`). Add that branch in the handler.
- **htmx:** same idea with attributes instead of hand-written fetch. Only adopt
  if the team wants the dependency.

## Features

1. **Live filters/tabs/sort:** clicking updates the URL (`history.pushState`)
   and swaps the feed; no scroll jump. Falls back to normal navigation.
2. **Debounced search:** `q` input updates the feed ~250ms after typing; also
   pushes to URL.
3. **Infinite scroll:** when the last item nears the viewport, fetch
   `?page=N+1&partial=1` and append. Keep a manual "Load more" fallback.
4. **⌘K palette:** keyboard shortcut focuses/opens the search box (matches the
   `⌘K` hint in the mockup).
5. **View toggle:** list/grid switch — toggle a class on the feed; persist in
   `localStorage`.

## Constraints

- Baseline must work with JS off (all controls are real links/forms).
- Don't refetch the sidebars on every filter unless counts change — swap only
  the feed for speed.
- Respect `prefers-reduced-motion` for any transitions.

## Verify

- With JS disabled: every control still navigates correctly.
- With JS on: filtering/scroll/search update the feed and the URL stays
  shareable (paste URL in a new tab → same state).

## Acceptance criteria

- No regression to the SSR baseline.
- URL always reflects current state (back/forward buttons work).
- Infinite scroll + manual fallback both function.
