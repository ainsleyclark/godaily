# Package Architecture

This document is the source of truth for how GoDaily's Go packages are
laid out under `pkg/`. It exists so that anyone — human or agent —
adding or moving code knows where it belongs without guessing. If you
are about to create a new package, add a new type, or wonder whether a
change "fits" the structure, read this first.

The current tree on `main` does not yet match this target. The
[Migration plan](#migration-plan) at the end lists the moves needed
to reach it.

## Layering principle

GoDaily uses **horizontal layers, strictly mirrored by domain**. Three
layers, each a sibling under `pkg/`:

- `pkg/domain/X/` — value types, repository interfaces, and pure
  domain logic for bounded context `X`.
- `pkg/services/X/` — behaviour: workflows and orchestration that
  operate on `domain/X` types.
- `pkg/store/<table>/` — persistence: SQL-backed implementations of
  the repository interfaces from `domain/X`.

We chose horizontal layers over vertical DDD slices because the
domains genuinely share types (engagement metrics reference news
items; social posts reference issues), and sqlc emits a single
generated package under `pkg/store/internal/sqlc/` that all stores
depend on. Horizontal layers avoid import-cycle gymnastics and match
the conventions of most large Go codebases.

## The mirror rule

> For every domain `X`, there is a matching `services/X` (when the
> domain has behaviour) and one or more `store/*` packages (when the
> domain is persisted). Nothing exists in `services/` or `store/`
> without a corresponding `domain/` package. A service package only
> imports the `domain/X` whose name it shares; cross-domain types are
> referenced by ID (e.g. `int64`), not by importing another domain.

To audit the rule:

```
# every services/X must import domain/X
for d in pkg/services/*/; do
  name=$(basename "$d")
  grep -q "domain/$name" "$d"*.go || echo "MISMATCH: $name"
done
```

A single domain may map to multiple stores when it spans more than one
table (e.g. `domain/social` is persisted via both `store/socialposts`
and `store/socialmetrics`). That's fine — the package comment of each
store names the domain it serves.

## Package map (target)

| Domain (`pkg/domain/`) | Service (`pkg/services/`) | Store (`pkg/store/`) |
|---|---|---|
| `news` | `digest` | `issues`, `items` |
| `social` | `social` | `socialposts`, `socialmetrics` |
| `subscriber` | `subscriber` | `subscribers` |
| `engagement` | `engagement` | `emailevents`, `engagement` |

Notes:

- `services/digest` is named for the workflow it drives, not the
  domain. This is the one allowed exception to the mirror rule —
  digest builds, sends, and recaps GoDaily issues, so "digest"
  describes what it does more clearly than "news" would. It still
  imports only `domain/news` from the domain layer.
- `store/socialmetrics` writes to a domain that lives in
  `engagement.SocialMetric`, not `domain/social`. The mapping is
  documented in the store's package comment so the boundary is clear.

## Per-domain reference

These sections are the most important part of this doc. When you add
a new type, find its domain below and check the **Owns** and
**Does NOT own** lists before deciding where the file goes.

### `domain/news`

**Purpose.** The news domain models the content GoDaily collects,
ranks, and ships in each issue.

**Owns.**

- `Item` — a single news article (title, URL, source, score, etc.).
- `Issue` — a collection of items shipped as one digest.
- `IssueStatus` — `draft` / `sent` / `error`.
- `Source` — the identifier for a content origin (HN, Reddit, etc.).
- `Sources` (registry) — the canonical list of every source GoDaily
  ingests from.
- `Tag` — a source-specific hint attached to an item.
- `Author` — identity for an item's author.
- `Score` — per-source relevance/popularity normaliser.
- `Registry` — the runtime fetcher registry (Builder, Get, Materialise).
- `ListOptions`, `ItemListOptions` — filter/pagination for List queries.
- `IssueRepository`, `ItemRepository` — persistence interfaces.

**Does NOT own.**

- ❌ `SocialPost`, `SocialProfile` → `domain/social`. A post about a
  news source is a social concept, not a news concept.
- ❌ `Subscriber` → `domain/subscriber`. The newsletter recipient is
  not a news entity.
- ❌ `EmailEvent`, `SocialMetric` → `domain/engagement`. Anything
  measured about how users react to content is engagement.

**Service.** `services/digest`.

- `collect.go` — fetches items from sources via the fetcher registry.
- `build.go` — ranks, filters, deduplicates items into a draft issue.
- `suggest.go` — AI-generated subject/summary.
- `preview.go` — renders the digest for preview.
- `send.go` — dispatches the issue via the email gateway.
- `recap.go` — assembles the weekly recap (folded in from
  `services/recap`).
- `run.go` — orchestrates the full daily run.

**Stores.** `store/issues`, `store/items`.

**Cross-references.** None at the import level. `engagement.LinkClicks`
holds an `ItemID`, but it does so by `int64`, not by importing
`news.Item`.

### `domain/social`

**Purpose.** The social domain models GoDaily's outbound presence on
third-party platforms (Bluesky, LinkedIn, Mastodon, etc.).

**Owns.**

- `Post` (currently `SocialPost`) — a single outbound post on one or
  more platforms.
- `PostKind` (currently `SocialPostKind`) — `featured` / `new_source`
  / `recap` / `spotlight` / `cta` / `community`.
- `Profile` (currently `SocialProfile`) — the social metadata
  (display name, mentions, blurb, announceability) attached to a
  news source so it can be spotlighted or announced.
- `Profiles` — the curated registry of all profiles.
- `PostRepository` — persistence interface.

**Does NOT own.**

- ❌ Platform clients (Bluesky API, LinkedIn API, Mastodon API)
  → `pkg/services/social/platform`. The domain is provider-agnostic.
- ❌ `SocialMetric` (likes/reposts/impressions) → `domain/engagement`.
  Metrics about how a post performed are engagement signals, not
  social-domain types.
- ❌ The scheduling logic that picks "what to post when" lives in
  `services/social`, not in the domain.

**Service.** `services/social`.

- `candidate.go` / `candidates/` — candidate selection.
- `rotation.go` — Tue/Fri rotation slot logic.
- `schedule.go` — when each kind runs.
- `service.go` — top-level entry points.
- `prompts/` — AI prompts for post bodies.

**Stores.** `store/socialposts` (posts themselves).
`store/socialmetrics` writes engagement metrics about posts and
belongs to the engagement domain — see below.

**Cross-references.** Posts hold an optional `IssueID` (`int64`)
referencing `news.Issue`. No import dependency.

### `domain/subscriber`

**Purpose.** People who have signed up to receive GoDaily by email,
plus their lifecycle (confirm, unsubscribe, bounce, suppress).

**Owns.**

- `Subscriber` — the recipient record (email, tokens, confirmation
  and unsubscribe state, bounce/suppression timestamps).
- `SubscriberRepository` — persistence interface, including the
  health side effects (`MarkBounced`, `MarkComplained`,
  `MarkSuppressed`).

**Does NOT own.**

- ❌ The raw email events that *cause* bouncing → `domain/engagement`.
  Subscriber state is the **derived** outcome; the events themselves
  are engagement data.
- ❌ Provider-specific webhook parsing (Resend payloads, etc.) →
  `pkg/gateway/email`.

**Service.** `services/subscriber`.

- `service.go` — subscription lifecycle: subscribe, confirm,
  unsubscribe, reactivate. Implements the `SubscriberHealth`
  interface consumed by `services/engagement` so engagement events
  can update subscriber state without a reverse import.

**Stores.** `store/subscribers`.

**Cross-references.** None. The flow is one-way:
`services/engagement` consumes a `SubscriberHealth` interface
satisfied by `services/subscriber`.

### `domain/engagement`

**Purpose.** Everything we measure about how users and platforms
interact with GoDaily — email lifecycle events, the aggregates
derived from them, and social-platform engagement counters.

**Owns.**

- `EmailEvent` — a single email lifecycle event (delivered, opened,
  clicked, bounced, complained).
- `EmailEventType` — the typed enum of the above, with `Valid()`.
- `IssueStats` — per-issue aggregate (open rate, click rate, etc.).
- `LinkClicks` — per-link click counts within an issue.
- `MetricsFilter` — date-range filter for aggregate queries.
- `SummaryStats` — headline numbers over a period.
- `IssueEngagement` — extended per-issue aggregate.
- `SocialMetric` — latest engagement counts for a social post on a
  platform (likes, reposts, comments, impressions).
- `EmailEventRepository`, `MetricsRepository`, `MetricsReporter`,
  `SocialMetricRepository` — persistence and reporting interfaces.

**Does NOT own.**

- ❌ `Subscriber` (even though bounces affect subscriber state) →
  `domain/subscriber`. Engagement records the **event**; subscriber
  owns the **resulting state**.
- ❌ `Post`, `Profile` → `domain/social`. We measure posts, we
  don't define them.
- ❌ Provider webhook DTOs → `pkg/gateway/email`,
  `pkg/services/social/platform`.

**Service.** `services/engagement` (renamed from `services/metrics`,
absorbs `services/emailevent`).

- `roundup.go` — aggregate queries (the existing metrics service).
- `events.go` — email event ingestion and the
  bounce/complaint → subscriber-health hand-off (moved from
  `services/emailevent`).

**Stores.** `store/emailevents` (raw events table) and
`store/engagement` (the aggregate metrics queries — renamed from
`store/metrics`).

**Cross-references.** Holds foreign-key IDs to `news.Issue`,
`news.Item`, `subscriber.Subscriber`, and `social.Post`. All by
`int64`. The only **interface** dependency is the
`SubscriberHealth` interface, which is **defined in
`services/engagement`** and **satisfied by `services/subscriber`** —
a standard "consumer defines the interface" pattern that keeps the
import direction clean.

## Where shared code lives

| Concern | Location |
|---|---|
| Generated sqlc code | `pkg/store/internal/sqlc/` |
| DB type conversion helpers | `pkg/store/internal/dbtypes/` |
| DB test fixtures and helpers | `pkg/store/internal/dbtest/` |
| Pure value-level helpers tied to a domain | inside that `pkg/domain/X/` |
| Pure utilities with no domain meaning | `pkg/util/Xutil/` (e.g. `aiutil`) |
| Provider-specific external clients | `pkg/gateway/<provider>/` |
| Source scrapers | `pkg/source/<source>/` |
| AI providers | `pkg/ai/<provider>/` |
| HTTP routing and handlers | `pkg/api/`, `web/` |
| Cross-service composition (wiring) | `pkg/app.go` |
| Embedded YAML data (conferences, meetups) | `pkg/data/` |
| Email templates | `pkg/templates/` |

If your helper has no obvious home above, default to
`pkg/util/<thing>util/`. Do not put miscellaneous code into a
domain or service package just to give it a parent.

## Where new code goes

A decision tree for adding new code:

1. **New bounded context?** Add `pkg/domain/X/` first. Define the
   types and repository interfaces. Then add `pkg/services/X/` and
   `pkg/store/<table>/` only when there is real behaviour or
   persistence to write. Update the [Package map](#package-map-target).
2. **New type on an existing domain?** Add it to the existing
   `pkg/domain/X/` package. Never duplicate a type across domains.
3. **New workflow or orchestration?** Add a new file inside
   `pkg/services/X/`. Do not create a new sibling service package
   unless it owns a distinct domain.
4. **New persistence for an existing domain?** Add
   `pkg/store/<table>/`. The package comment must name the domain it
   serves.
5. **New external integration?** Add `pkg/gateway/<provider>/`. The
   domain stays provider-agnostic.
6. **New cross-cutting helper?** Add `pkg/util/<thing>util/`. Single
   word plus the `util` suffix.

## Naming rules

- **Single word, lowercase, no dashes or underscores.** This applies
  to every package under `pkg/`.
- **Domain and service names match exactly.** `services/social`
  imports `domain/social`. The one allowed exception is
  `services/digest` (workflow-named); document any future exceptions
  here.
- **Store packages are named after their database table.** Plural is
  fine (`items`, `subscribers`, `socialposts`). The package comment
  must name the domain whose interface the store implements.
- **No `emailevent` (singular) domain.** Email events live in
  `domain/engagement` because they *are* engagement signals.
  Splitting them out would re-blur the very boundary this layout
  exists to clarify.
- **Every package keeps a package comment.** One sentence, present
  tense, matching the "Purpose" line in this doc. This keeps
  `go doc` and this file in sync.

## Migration plan

The current tree differs from the target. Each bullet below is a
separate, reviewable PR. None of them changes behaviour — they move
files and update imports only. Run `go test ./...` and
`golangci-lint run ./... --fix --config=.golangci.yaml` after each.

1. **Split `domain/news`**
   - Move `pkg/domain/news/social-post.go` →
     `pkg/domain/social/post.go` (rename type `SocialPost` → `Post`,
     `SocialPostKind` → `PostKind`).
   - Move `pkg/domain/news/social-profiles.go` →
     `pkg/domain/social/profile.go` (rename `SocialProfile` →
     `Profile`).
   - Move `pkg/domain/news/subscriber.go` →
     `pkg/domain/subscriber/subscriber.go`.
   - Update all imports; regenerate mocks with `go generate
     ./pkg/domain/...`.
2. **Rename `services/metrics` → `services/engagement`**
   - Rename the directory and the package declaration.
   - Update all callers (`pkg/app.go`, `pkg/api/`).
3. **Rename `store/metrics` → `store/engagement`**
   - Rename the directory and package; update the
     `sqlc.yaml` package path if needed; re-run `sqlc generate`.
4. **Fold `services/emailevent` into `services/engagement`**
   - Move `service.go` → `services/engagement/events.go`.
   - Merge the `Service` struct constructors if they were separate.
   - Delete `pkg/services/emailevent/`.
5. **Fold `services/recap` into `services/digest`**
   - Move `recap.go` → `services/digest/recap.go`.
   - Delete `pkg/services/recap/`.
   - Update callers (web handlers, social rotation).
6. **Audit the mirror rule**
   - Run the audit snippet from [The mirror rule](#the-mirror-rule).
   - Update every package comment to match its "Purpose" line in
     this doc.
