# Social org mentions — `@` typeahead in the drafts editor

This doc captures the design for tagging organisations and Pages from
the social-drafts editor by typing `@`, picking from our existing
curated list, and having the right identifier (URN on LinkedIn,
handle on Bluesky/Mastodon) attached to the draft so the live post
renders as a real clickable mention on every platform.

This is a narrower, ship-now alternative to
`social-mention-autocomplete.md`. That spec aimed to let editors tag
*arbitrary* accounts (people and orgs) via live typeahead against
each platform's search API. Phase 0 of that doc confirmed LinkedIn
typeahead is gated behind LinkedIn Marketing Partner approval — both
for general people/org search and for the followers-only person
search inside the Community Management API. LinkedIn person mentions
are also a *product* rule, not just an API gate: even from
linkedin.com's own composer, a Page can only @-tag accounts that
follow it. This spec drops the partner-gated parts and ships the
useful 80%.

## Background

GoDaily already has a curated mention catalog: `social.Profiles` in
`pkg/domain/social/profile.go` is a static `map[news.Source]Profile`
(16 entries), and `publish_drafts.go` (~L79-86) looks up
`ProfileFor(draft.MentionSource).Mentions` at publish time. Mention
attachment is uneven per platform:

- **Bluesky** ignores `req.Mentions`; `buildFacets` only emits
  richtext facets for URLs.
- **Mastodon** ignores `req.Mentions`; inlined `@user@instance` is
  resolved server-side at post time.
- **LinkedIn** consumes `req.Mentions` via `buildAnnotations` — but
  the curated `Profiles` entries don't carry LinkedIn URNs today
  (every existing LinkedIn `Mention.Handle` is empty), so the
  annotation pipeline produces no output in practice.

The dashboard social-drafts editor
(`dashboard/src/routes/social/drafts/+page.svelte`) is a plain
`<textarea>` (~L177-182). `PATCH /social/drafts/{id}` only accepts
`text`. No mention surface exists.

## Goal

In the editor textarea, type `@` → live popup of orgs from
`social.Profiles` → pick one → display name is inserted into the
text, the mention is attached to the draft, and at publish time the
right identifier reaches each platform. For an org not in the
curated list, a tiny inline "Add to this draft" form (paste
LinkedIn URL, type Bluesky/Mastodon handles) attaches a one-off
mention to just this draft without persisting to the catalog.

Out of scope:

- **LinkedIn person mentions** — partner-gated, and a product rule
  restricts Page mentions to followers anyway. Not pursued.
- **DB-backed catalog with CRUD** — `social.Profiles` stays static
  in code. Adding a new *reusable* mention is a code change.
  Per-draft one-offs cover the long tail.
- **Live typeahead against external platform APIs** — the popup
  searches only our in-memory `Profiles` map. Bluesky/Mastodon
  handles in the "Add to this draft" form are accepted at face
  value; LinkedIn URL is resolved server-side.
- **Scheduling** — separate concern.

## Approach

Three building blocks:

1. **Backfill LinkedIn URNs** into existing `social.Profiles`
   entries. Data-only.
2. **Per-draft `mentions` JSON column** on `social_posts`, written by
   the editor. At publish, per-draft mentions win; `Profiles` is
   the seed/default for the 2am auto-attach path (unchanged).
3. **One small Go endpoint** for LinkedIn URL resolution
   (`/v2/organizations?q=vanityName=`), reused only when the editor
   adds a one-off LinkedIn mention via paste-URL.

The `@` popup never hits an external platform API. Bluesky and
Mastodon validation in the "Add to this draft" form is optional —
the dashboard can hit those platforms' public search endpoints
client-side later if we want, but day-one accepts handles at face
value.

## Phase A — Backfill LinkedIn URNs in `social.Profiles`

Edit `pkg/domain/social/profile.go`:

- For each entry with a known LinkedIn presence (Ardan Labs,
  JetBrains, GoPodcast, Fallthrough, DevTo), add a
  `Mention{Platform: social.LinkedIn, DisplayName: ..., Handle:
  "urn:li:organization:<id>"}`. Look up URNs once via
  `curl -H 'Authorization: Bearer $LINKEDIN_OAUTH_TOKEN' -H
  'LinkedIn-Version: 202604' "https://api.linkedin.com/v2/organizations?q=vanityName&vanityName=<slug>"`
  and paste the URN into the source — these are stable.
- Entries with no LinkedIn presence are left alone;
  `MentionsFor(LinkedIn)` returns nothing and nothing gets attached.
- Unit test: every `Mention{Platform: LinkedIn}` row has a `Handle`
  matching `^urn:li:organization:\d+$`.

## Phase B — Per-draft `mentions` column

- Migration `pkg/db/migrations/0014_social_posts_mentions.sql` (Goose
  format, mirror `0013`): `ALTER TABLE social_posts ADD COLUMN
  mentions TEXT;` (+ Down drop). SQLite has no JSONB; store JSON as
  text.
- `pkg/store/socialposts/query.sql`:
  - Add `mentions` to the column list / `VALUES` of `SocialPostCreate`.
  - Add `mentions = COALESCE(sqlc.narg('mentions'), mentions)` to
    `SocialPostUpdate`.
- No `sqlc.yaml` override — nullable `TEXT` generates
  `sql.NullString`. Run `sqlc generate`; commit
  `pkg/store/internal/sqlc/{models.go,query.sql.go}`.
- Domain `pkg/domain/social/post.go`: add `Mentions []Mention` to
  `Post` and `Mentions *[]Mention` to `PostUpdate`.
- `pkg/store/socialposts/store.go`: small `marshal/unmarshalMentions`
  helpers handling empty/`null`, wired into `Create`, `Update`,
  `transform`.
- Tests: round-trip mentions through Create→Find and Update.

The 2am build path is **not** modified — `MentionSource` continues
to seed mentions at publish time via `ProfileFor`. The new column
only carries editor-attached mentions.

## Phase C — LinkedIn paste-URL resolver

- New method on `*linkedin.Client` in
  `pkg/services/social/platform/linkedin/linkedin.go`, mirroring
  `resolveDID` in `bluesky.go`:

  ```go
  // ResolveOrgURL takes a LinkedIn company page URL and returns
  // (urn:li:organization:<id>, displayName, error). Accepts
  // linkedin.com/company/<slug> URLs. Returns ErrOrgNotFound on miss.
  func (c *Client) ResolveOrgURL(ctx context.Context, pageURL string) (string, string, error)
  ```

  Parse `<slug>` from the URL. Hit
  `GET {baseURL}/v2/organizations?q=vanityName&vanityName=<slug>`
  with the existing `Authorization` / `LinkedIn-Version` /
  `X-Restli-Protocol-Version` headers. Decode
  `{ elements: [{ id, localizedName, ... }] }`; build
  `urn:li:organization:<id>` and return with `localizedName`.
- New interface `LinkedInOrgResolver` in
  `pkg/services/social/platform/lookup.go` (new file) with a
  `//go:generate mockgen ...` directive. Does not touch `Poster`.
- Wiring mirrors `StatFetcher` exactly:
  `buildLinkedInOrgResolver` in `bootstrap.go` returns `nil` when
  LinkedIn creds are absent; `linkedInOrgResolver` field + accessor
  in `service.go`; `App.LinkedInOrgResolver` in `pkg/app.go`.
- Tests: `httptest.NewServer` returning a fixture response
  (success, empty `elements`, non-200).

## Phase D — Mention endpoints

Two new handlers under `pkg/api/handlers/social/mentions/`:

- `GET /social/mentions?q=` — typeahead over the in-process
  `social.Profiles` map. `strings.Contains` against `DisplayName`
  (case-insensitive). No DB query, no external API call. Returns
  `[{slug, displayName, mentions: [{platform, displayName,
  identifier}]}]` for the matched profiles. Min `q` length 1.
- `POST /social/mentions/resolve-linkedin` body `{url}` — calls
  `LinkedInOrgResolver.ResolveOrgURL`; returns
  `{displayName, identifier}` where identifier is the URN. 404 on
  not-found, 503 when resolver is nil.

Mounted in `pkg/api/handlers/social/handler.go` next to the existing
sub-handlers. Swag-annotated request/response types.

Tests: typeahead happy path; empty query (400); resolver-absent (503);
LinkedIn lookup (success / not-found).

## Phase E — Drafts update endpoint accepts mentions

- `pkg/api/handlers/social/drafts/update.go`: extend
  `SocialDraftUpdateRequest` with `Mentions []SocialDraftMention`
  (`{Platform, DisplayName, Identifier}`); reject mentions whose
  platform ≠ the draft's platform (400); map `Identifier`→`Handle`
  on `social.Mention`; pass through as `PostUpdate{Text, Mentions}`
  (full-list replace semantics).
- Run `make openapi-ts`; commit
  `docs/openapi/swagger.{yaml,json}` and
  `dashboard/src/lib/api/schema.d.ts`.
- Tests: mentions round-trip; cross-platform rejection; empty list
  clears existing mentions.

## Phase F — Publish-path: per-draft mentions win

`pkg/services/social/publish_drafts.go` (~L79-86): if
`draft.Mentions` is non-empty use it; else fall back to
`ProfileFor(draft.MentionSource).Mentions`. Tiny change. Test
override precedence.

## Phase G — Real Bluesky mention facets

`pkg/services/social/platform/bluesky/bluesky.go`:

- Generalise `facetFeature` to carry `did` +
  `$type: app.bsky.richtext.facet#mention`.
- Add `buildMentionFacets(ctx, text, mentions, resolve)` that scans
  the post-`TruncatePost` text for each Bluesky `Mention.Handle`
  token, computes **UTF-8 byte ranges** the same way `buildFacets`
  does for links, resolves via existing `resolveDID` (drop-and-log
  on failure), and merges into `record["facets"]`.
- Update the "req.Mentions is ignored" doc comment.
- Tests: facet byte-range correctness on a multibyte string;
  multiple mentions; mention + URL coexisting; resolve-failure drop.

LinkedIn and Mastodon clients are not modified — LinkedIn already
consumes `req.Mentions`; Mastodon inlines.

## Dashboard — `@` popup + inline "Add to this draft"

New `dashboard/src/lib/components/MentionAutocomplete.svelte` wraps
the existing textarea. On input, reads `selectionStart`, scans back
to an active `@`-token; if present, shows a caret-anchored popup:

- 200-250ms debounced `GET /social/mentions?q=`.
- Keyboard nav (↑/↓/Enter/Esc), outside-click close.
- On select: splice the curated `DisplayName` into the text at the
  `@`-token position (LinkedIn) or splice the platform handle
  (`@handle` for Bluesky, `@user@instance` for Mastodon), and push
  `{platform, displayName, identifier}` for the *current draft's
  platform only* onto a deduped `mentions` array. Identifier comes
  from the profile's `MentionsFor(draft.platform)[0].Handle`.

Footer of the popup: a small **"+ Add to this draft"** action.
Opens an inline form (platform-aware):

- **Bluesky/Mastodon**: a single handle input. On submit, splice
  the handle into the text and append a mention.
- **LinkedIn**: a single URL input ("Paste a LinkedIn company
  page URL"). On submit, calls
  `POST /social/mentions/resolve-linkedin`; on success, inserts the
  resolved `DisplayName` into the text at the `@`-token position
  and appends a mention with the resolved URN. The catalog is **not**
  modified — this is draft-scoped only.

`+page.svelte`: replace the bare textarea (L177-182) with
`MentionAutocomplete`; add `mentionsByDraft` state seeded from
`d.mentions` in `load()`; pass it in `save()`; include mention
changes in `isDirty`.

`dashboard/src/lib/api/client.ts`: `searchMentions(q)`,
`resolveLinkedInOrg(url)`, extend `updateSocialDraft(id, text,
mentions)` body. `types.ts`: add `SocialMention`,
`SocialDraftMention`, `LinkedInOrgResolution` aliases.

## Codegen / tests / lint (run + commit every generated artifact)

Order: `sqlc generate` → `go generate
./pkg/services/...` (for the new `LinkedInOrgResolver` mock) →
`make openapi-ts`. Then `go test ./...` and `golangci-lint run ./...
--fix --config=.golangci.yaml` (gofumpt; CI fails if not applied).
Dashboard via `pnpm`.

Must-commit generated files: `pkg/store/internal/sqlc/{models.go,
query.sql.go}`, `pkg/mocks/social/LinkedInOrgResolver.go`,
`docs/openapi/swagger.{yaml,json}`,
`dashboard/src/lib/api/schema.d.ts`.

## Verification

- `go test ./...` green, including resolver, mentions endpoints,
  store round-trip, publish precedence, and Bluesky facet tests.
  Highest-value test: Bluesky facet byte-range assertion on a
  multibyte string.
- Manual: in the dashboard editor, type `@arda` on a draft for
  each platform → popup → pick → confirm the text updates and a
  mention chip appears. Save, reload, confirm persistence.
- Manual: on a LinkedIn draft, click "+ Add to this draft", paste
  `linkedin.com/company/tailscale`, confirm the resolved display
  name is inserted into the text and the mention is attached.
- E2E publish a Bluesky draft with a mention; confirm the live
  post renders the @-handle as a clickable mention (facet
  present).
- E2E publish a LinkedIn draft whose text contains the curated
  `DisplayName` and a mention with the right URN; confirm the
  organisation renders as a real @-tag.

## Critical files

- `pkg/db/migrations/0014_social_posts_mentions.sql`,
  `pkg/store/socialposts/{query.sql,store.go}`,
  `pkg/domain/social/{post.go,profile.go}`
- `pkg/services/social/platform/{linkedin/linkedin.go,
  lookup.go (new), bluesky/bluesky.go}`,
  `pkg/services/social/{bootstrap.go,service.go,publish_drafts.go}`,
  `pkg/app.go`
- `pkg/api/handlers/social/{handler.go,
  mentions/{search.go,resolve_linkedin.go} (new),
  drafts/update.go}`
- `dashboard/src/routes/social/drafts/+page.svelte`,
  `dashboard/src/lib/components/MentionAutocomplete.svelte` (new),
  `dashboard/src/lib/api/{client.ts,types.ts}`

## Risks / notes

- **LinkedIn `buildAnnotations` silently drops mentions whose
  `DisplayName` is not a case-sensitive substring of the post
  text.** Mitigated here because the editor *inserts* the
  `DisplayName` into the text at pick time, so the substring match
  is guaranteed at draft time. The remaining risk is an editor
  later removing the name from the text without removing the
  chip; the publish-time WARN log makes it discoverable. Acceptable.
- **Auto-attached mentions from the 2am path are still subject to
  the substring rule.** Featured-post text is AI-reframed and may
  not echo the source `DisplayName`. The fix is to update the
  featured-post prompt (`pkg/services/featured/`) to mandate the
  source `DisplayName` appears verbatim. Small, separate change;
  can land alongside Phase A.
- **`social.Profiles` stays static.** Editing it is a code change.
  Per-draft mentions are the editor's escape hatch. Promoting a
  repeated ad-hoc mention into the curated map is a manual
  engineering task — acceptable while the set is small.
- **Bluesky `Mention.Handle` is the raw `@handle`, not the DID.**
  The DID is resolved at publish time, mirroring `resolveDID` for
  URL facets. A typo silently drops the mention at publish with a
  WARN log — same behaviour as today's link facets.
- **LinkedIn `buildAnnotations` uses byte offsets, not UTF-16 code
  units** — the LinkedIn API technically requires UTF-16. Existing
  curated `DisplayName`s are all ASCII so this isn't a live bug;
  flag as a constraint ("ASCII display names only") until/unless
  we hit it.
