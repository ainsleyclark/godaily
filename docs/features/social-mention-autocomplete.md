# Social @mention autocomplete + real Bluesky mentions

This doc captures the design for letting an editor tag arbitrary accounts
from the social-drafts editor by typing `@`, and for making Bluesky
mentions render as real clickable mentions. Pick this up when we want the
dashboard to drive mentions instead of relying solely on the curated
`social.Profiles` auto-attach at publish time.

## Background

The social-drafts editor (`dashboard/src/routes/social/drafts/+page.svelte`)
is a plain `<textarea>`, and `PATCH /social/drafts/{id}`
(`pkg/api/handlers/social/drafts/update.go`) only accepts `text`. Mentions
are an automatic, publish-time mechanism: a draft carries a `MentionSource`
(a curated `social.Profiles` slug), and `PublishDrafts`
(`pkg/services/social/publish_drafts.go`, ~L79-86) looks up that profile's
`Mentions` and hands them to the poster. There is no way to tag an
arbitrary account from the UI.

Current behaviour is also uneven per platform:

- **Mastodon** resolves inlined `@user@instance` natively (server-side);
  `Post` ignores `req.Mentions`.
- **Bluesky** inlined `@handles` render as **plain text** — the post
  builder (`buildFacets` in `pkg/services/social/platform/bluesky/bluesky.go`)
  only emits richtext facets for URLs, never mentions. `Post` ignores
  `req.Mentions`.
- **LinkedIn** needs a `urn:li:...` attached as an out-of-band annotation
  (`buildAnnotations`, `linkedin.go`). URNs are hardcoded in profiles and
  effectively unset, so LinkedIn mentions never actually fire today.

## Goal

Type `@` in the editor → live popup of real accounts → pick one → it
publishes as a real mention on every platform. Two product decisions are
locked in: **live per-platform search** (tag anyone, not just curated
profiles), and **real clickable Bluesky mentions**.

## Approach

- **No new storage for Bluesky/Mastodon.** Bluesky resolves `@handle`→DID at
  publish time (reuse existing `resolveDID`) and emits a `#mention` facet;
  Mastodon resolves the inlined `@user@instance` server-side. Tagging anyone
  on these two is just inlined text plus a publish-time facet pass.
- **LinkedIn forces persistence.** Its annotation needs a `urn:li:...` that
  cannot be derived from text, so the chosen URN is stored per-draft in a
  new `mentions` column. The DB is **SQLite** (`sqlc.yaml` engine `sqlite`),
  so store a JSON array in a `TEXT` column — there is no JSONB.
- **Search is an optional capability**, modelled on the existing
  `StatFetcher` interface and its wiring (`buildStatFetchers` in
  `bootstrap.go` → `Service.StatFetchers()` → `App.StatFetchers` →
  injected into the social `Handler`). Do NOT bolt search onto `Poster`:
  that keeps the `Poster` mock untouched and avoids forcing every platform
  to implement search.

## Phase 0 — LinkedIn API gating spike (do first, blocking for LinkedIn only)

LinkedIn live people/org search likely needs elevated products/scopes the
current token (`w_/r_organization_social`) lacks. Before building any
LinkedIn searcher, manually probe a typeahead/org-search endpoint with the
production `LINKEDIN_OAUTH_TOKEN` and record the status. A
`403 ACCESS_DENIED` / `QUERY_PARAM_NOT_ALLOWED` confirms it is unavailable.

- **If available** → implement a LinkedIn `Searcher` in Phase B.
- **If unavailable (expected)** → LinkedIn gets a graceful fallback: no live
  popup; the editor shows a manual "DisplayName + paste URN" mention field
  for LinkedIn rows (and/or offers curated `social.Profiles` mentions).
  Bluesky & Mastodon proceed with live search regardless.

Everything else in the plan is independent of this outcome.

## Phase A — Persist per-draft mentions

- Migration `pkg/db/migrations/0014_social_posts_mentions.sql` (Goose format,
  mirror `0013`): `ALTER TABLE social_posts ADD COLUMN mentions TEXT;` (+ Down
  drop).
- `pkg/store/socialposts/query.sql`: add `mentions` to `SocialPostCreate`
  columns/`VALUES`; add `mentions = COALESCE(sqlc.narg('mentions'), mentions)`
  to `SocialPostUpdate`. List/Find use `SELECT *` and pick it up
  automatically.
- No `sqlc.yaml` override needed — nullable TEXT generates `sql.NullString`.
  Run `sqlc generate`; commit
  `pkg/store/internal/sqlc/{models.go,query.sql.go}`.
- Domain `pkg/domain/social/post.go`: add `Mentions []Mention` to `Post` and
  `Mentions *[]Mention` to `PostUpdate`.
- `pkg/store/socialposts/store.go`: marshal/unmarshal mentions JSON in
  `Create`, `Update`, and `transform` (small `marshal/unmarshalMentions`
  helpers handling empty/`null`).
- Tests: round-trip mentions through Create→Find and Update.

## Phase B — Search capability + endpoint

- New `pkg/services/social/platform/search.go`: `Account{DisplayName,
  Identifier, Subtitle, AvatarURL}` and `Searcher{ Platform();
  Search(ctx, query) ([]Account, error) }` with a `//go:generate mockgen ...
  Searcher` directive (separate interface — the `StatFetcher` precedent).
- Per-platform `Search` on the existing `Client`s:
  - Bluesky (`bluesky.go`): `app.bsky.actor.searchActorsTypeahead?q=&limit=8`
    against `appViewURL` (no auth). `Identifier = "@"+handle`.
  - Mastodon (`mastodon.go`): `/api/v2/search?type=accounts&resolve=true&q=`
    via the configured token; stub the SDK call behind a func field (like
    `postStatusFunc`) for testability. `Identifier = "@"+acct`.
  - LinkedIn: only if Phase 0 passes; else no searcher registered.
- Wiring — mirror `StatFetcher` exactly: `buildSearchers` in
  `pkg/services/social/bootstrap.go`; `searchers` field + `Searchers()`
  accessor in `service.go`; `App.Searchers` in `pkg/app.go`; inject
  `searchers` map into the parent social `Handler`
  (`pkg/api/handlers/social/handler.go`).
- New handler `pkg/api/handlers/social/search_accounts.go`:
  `GET /social/accounts/search?platform=&q=` (auth). Validate platform;
  require `q` (min len 2); look up `h.searchers[platform]`; return `200`
  empty list when absent (graceful LinkedIn fallback). swag-annotated
  `SocialAccount` response. Register in `Routes`.
- Run `go generate ./pkg/services/...`; commit `pkg/mocks/social/Searcher.go`.
- Tests: per-platform searcher (Bluesky httptest, Mastodon stub) + handler
  (valid / unknown platform / searcher-absent→empty).

## Phase C — Publish path: consume mentions + real Bluesky facets

- `pkg/services/social/publish_drafts.go` (~L79-86): if `draft.Mentions` is
  non-empty use it, else fall back to the existing `ProfileFor(...).Mentions`.
  Per-draft mentions take precedence; profile stays the auto-generated
  default.
- `bluesky.go`: generalize `facetFeature` to carry `did` +
  `$type: app.bsky.richtext.facet#mention`; add `buildMentionFacets(ctx,
  text, resolve)` that regex-scans `@handle` tokens, computes **UTF-8 byte
  ranges** exactly like `buildFacets`, resolves each via `resolveDID`
  (drop+log on failure so a bad handle never fails the post), and merges into
  `record["facets"]`. Resolve against the post-`TruncatePost` text so offsets
  stay aligned. Update the "req.Mentions is ignored" doc comment.
- LinkedIn (`linkedin.go`) and Mastodon (`mastodon.go`): no change — LinkedIn
  already consumes `req.Mentions` via `buildAnnotations` (now fed real URNs);
  Mastodon handles are inlined in text.
- Tests: bluesky facet byte-range correctness (multibyte rune before
  `@handle`, multiple mentions, mention+URL coexisting, resolve-failure drop);
  publish_drafts draft-mentions-override-profile precedence.

## Phase D — Update endpoint accepts mentions

- `pkg/api/handlers/social/drafts/update.go`: extend
  `SocialDraftUpdateRequest` with `Mentions []SocialDraftMention`
  (`{Platform, DisplayName, Identifier}`); map Identifier→`Handle`; set
  `PostUpdate{Text, Mentions}` (full-list replace semantics — the editor
  always sends the current set). Reject mentions whose platform ≠ the draft's
  platform (400). Update swag annotations.
- Run `make openapi-ts`; commit `docs/openapi/swagger.{yaml,json}` and
  `dashboard/src/lib/api/schema.d.ts`.
- Tests: mentions round-trip; cross-platform rejection.

## Phase E — Dashboard autocomplete (Svelte 5 runes + Tailwind, no new deps)

- `dashboard/src/lib/api/client.ts`: add `searchAccounts(platform, q)` (reuse
  the `subscriberList` debounced-GET pattern); extend
  `updateSocialDraft(id, text, mentions)` body to `{ text, mentions }`.
- `dashboard/src/lib/api/types.ts`: add `SocialAccount` + `SocialDraftMention`
  aliases.
- New `dashboard/src/lib/components/MentionAutocomplete.svelte`: wraps the
  textarea; on input read `selectionStart`, scan back to an active
  `@`-token, show a caret-anchored popup; 200-250ms debounced
  `searchAccounts`; on select splice the inline token into `text`
  (Bluesky/Mastodon insert `@handle`/`@acct`; **LinkedIn inserts the plain
  DisplayName** — matching `Profile.Mention`'s LinkedIn special-case — and
  records the URN separately); maintain a deduped `mentions` array; keyboard
  nav (↑/↓/Enter/Esc), outside-click close. For the Phase 0
  LinkedIn-unavailable case, render a manual DisplayName+URN form instead of
  the live popup.
- `+page.svelte`: replace the bare textarea (L177-182) with
  `MentionAutocomplete`; add `mentionsByDraft` state seeded from `d.mentions`
  in `load()`; pass it in `save()`; include mention changes in `isDirty`.

## Codegen / tests / lint (run + commit every generated artifact)

Regen in order: `sqlc generate` → `go generate ./pkg/domain/...
./pkg/services/...` → `make openapi-ts`. Then `go test ./...` and
`golangci-lint run ./... --fix --config=.golangci.yaml` (gofumpt; CI fails if
not applied). Dashboard typecheck/build via `pnpm`.

Must-commit generated files: `pkg/store/internal/sqlc/{models.go,
query.sql.go}`, `pkg/mocks/social/Searcher.go`, `docs/openapi/swagger.{yaml,
json}`, `dashboard/src/lib/api/schema.d.ts`.

## Verification

- `go test ./...` green, including the new searcher, facet,
  publish-precedence, store round-trip, and update-handler tests. The
  highest-value test is the Bluesky facet byte-range assertion against a
  multibyte string.
- Manual: `pnpm dev` against staging API — type `@` in a Bluesky and a
  Mastodon draft, confirm the live popup populates, select, save, reload →
  mentions persist. Confirm LinkedIn shows live search OR the manual-URN
  fallback per Phase 0.
- End-to-end publish a Bluesky draft with a selected mention and confirm the
  live post renders the @-handle as a real clickable mention (facet present).
  The `godaily` MCP skill can trigger publish/refresh but cannot exercise the
  editor UI, so the autocomplete itself is verified manually in the browser.

## Critical files

- `pkg/db/migrations/0014_social_posts_mentions.sql`,
  `pkg/store/socialposts/{query.sql,store.go}`, `pkg/domain/social/post.go`
- `pkg/services/social/platform/search.go` (new),
  `pkg/services/social/{bootstrap.go,service.go}`, `pkg/app.go`
- `pkg/services/social/platform/{bluesky/bluesky.go,mastodon/mastodon.go}`
- `pkg/services/social/publish_drafts.go`
- `pkg/api/handlers/social/{handler.go,search_accounts.go (new),
  drafts/update.go}`
- `dashboard/src/routes/social/drafts/+page.svelte`,
  `dashboard/src/lib/components/MentionAutocomplete.svelte` (new),
  `dashboard/src/lib/api/{client.ts,types.ts}`

## Risks / notes

- **LinkedIn live search may be impossible** with current API access —
  Phase 0 decides this; the fallback keeps the feature shippable for
  Bluesky/Mastodon.
- Bluesky `resolveDID` runs one network call per unique handle at publish
  time; acceptable given the tiny draft volume, and failures degrade
  gracefully.
- Scope is large and spans Go store/services/API + SvelteKit + four
  generated artifacts; land Phase 0 first, then the Bluesky/Mastodon path,
  before committing to LinkedIn UI.
