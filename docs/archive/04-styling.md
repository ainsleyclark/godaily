# Plan 04 — Styling (SCSS)

**Goal:** style the Browse page to match the mockup using the existing SCSS
system. No new build tooling.

**Depends on:** Plan 03 class names (coordinate before starting, or treat the
class list below as the contract).

## Where things live

- SCSS source: `web/assets/scss/` (`abstracts/_variables.scss`,
  `abstracts/_mixins.scss`, `components/`, `layout/`).
- Built by esbuild (`web/esbuild.mjs`) into `web/dist/`.
- Add a new partial `web/assets/scss/components/_browse.scss` and `@use`/import
  it wherever the component partials are aggregated (match how
  `_issues.scss`, `_digest-item.scss` are wired in).

## What to build (from the mockup)

- **Hero**: kicker label, large title with an accent-coloured word, sub copy.
  Reuse existing `.section__label/__title/__sub` if present.
- **Search box**: full-width pill input with leading icon and a `⌘K` hint chip.
- **Stat cards**: 4-up row of bordered cards (big number + small caption).
  Collapse to 2-up / 1-up on small screens.
- **Tabs**: horizontal scrollable row; active tab has underline + accent;
  count badges (pill) after each label.
- **3-column grid** (`.browse__grid`): left sidebar (sources + date range),
  center feed, right sidebar (trending). Use CSS grid; stack to single column
  under the tablet breakpoint (use existing breakpoint vars/mixins).
- **Source list**: checkbox-style rows with a coloured dot, label, right-aligned
  count; "+N more sources" toggle.
- **Sort bar**: segmented control (Hot/Top/New) + dropdowns (range, more
  filters) + RSS button + list/grid view toggle.
- **Browse item row**: source mark, title, snippet, meta line with section tag
  chip and the **"In digest"** chip, hover/focus states.
- **Trending sidebar**: numbered list, rank numerals, source + age meta.
- **Pagination**: prev/next + page indicator.

## Conventions

- Reuse `abstracts/_variables.scss` (colours, spacing, breakpoints) and
  `_mixins.scss` — don't hardcode hex/px where a token exists.
- Match the existing chip styling so the "In digest" `.tag` reads as a sibling
  of section tags.
- Mobile-first; verify the 3-column → single-column collapse.

## Verify

- Build assets (the project's pnpm/esbuild flow) and load `/browse/` in a
  browser at desktop, tablet, mobile widths.
- Check focus-visible states on every interactive control (a11y).

## Acceptance criteria

- Visual parity with the mockup at desktop; graceful responsive collapse.
- No hardcoded values where a SCSS token exists.
- Keyboard focus is visible on tabs, sources, sort, pagination.
