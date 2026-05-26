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

- `pkg/domain/X/` — value types, repository interfaces, service
  interfaces, operation DTOs (`*Options`, `*Result`, `*Response`),
  and pure domain logic for bounded context `X`.
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
| `news` | — | `items` |
| `digest` | `digest` | `issues` |
| `social` | `social` | `socialposts`, `socialmetrics` |
| `audience` | `audience` | `subscribers` |
| `engagement` | `engagement` | `emailevents`, `engagement` |

Notes:

- `domain/news` has no paired service — its items are consumed by
  `services/digest` (which builds issues from them) and by
  `services/social` (which posts about them). The mirror rule applies
  to service packages: every `services/X` imports its `domain/X`.
- `store/socialmetrics` writes to a domain that lives in
  `engagement.SocialMetric`, not `domain/social`. The mapping is
  documented in the store's package comment so the boundary is clear.

## Per-domain reference

These sections are the most important part of this doc. When you add
a new type, find its domain below and check the **Owns** and
**Does NOT own** lists before deciding where the file goes.

### `domain/news`

**Purpose.** The news domain models the raw content GoDaily collects
from external sources: individual articles before they are assembled
into a digest.

**Owns.**

- `Item` — a single news article (title, URL, source, score, etc.).
- `Source` — the identifier for a content origin (HN, Reddit, etc.).
- `Sources` (registry) — the canonical list of every source GoDaily
  ingests from.
- `Tag` — a source-specific hint attached to an item.
- `Author` — identity for an item's author.
- `Score` — per-source relevance/popularity normaliser.
- `Registry` — the runtime fetcher registry (Builder, Get, Materialise).
- `ItemListOptions` — filter/pagination for List queries.
- `ItemRepository` — persistence interface.

**Does NOT own.**

- ❌ `Issue`, `IssueStatus`, `IssueRepository` → `domain/digest`.
  Issues are the digest bounded context's aggregate, not news.
- ❌ `SocialPost`, `SocialProfile` → `domain/social`. A post about a
  news source is a social concept, not a news concept.
- ❌ `Subscriber` → `domain/audience`. The newsletter recipient is
  not a news entity.
- ❌ `EmailEvent`, `SocialMetric` → `domain/engagement`. Anything
  measured about how users react to content is engagement.

**Service.** None. Items are consumed by `services/digest` (which
builds issues from them) and by `services/social` (which picks a
featured item to post about).

**Stores.** `store/items`.

**Cross-references.** None at the import level. `engagement.LinkClicks`
holds an `ItemID`, but it does so by `int64`, not by importing
`news.Item`.

### `domain/digest`

**Purpose.** The digest domain models newsletter editions: the issues
GoDaily assembles from news items, sends to subscribers, and recaps
weekly.

**Owns.**

- `Issue` — a collection of items shipped as one newsletter edition.
- `IssueStatus` — `draft` / `sent` / `error`.
- `IssueRepository` — persistence interface.
- `Service` — the domain service interface for the collect→build→send
  workflow.
- `CollectOptions`, `CollectResponse`, `SourceItems` — operation DTOs
  for the collect step.
- `Period`, `Top`, `RankedItem`, `TopOptions` — recap value types
  representing the weekly top-clicked dataset.

**Does NOT own.**

- ❌ `Item`, `Source`, `ItemRepository` → `domain/news`. Items are
  upstream of any digest.
- ❌ `RecapService`, `Aggregator` — these are service implementations,
  not domain types, and live in `services/digest`.
- ❌ `Subscriber` → `domain/audience`.
- ❌ `EmailEvent`, `SocialMetric` → `domain/engagement`.

**Service.** `services/digest`.

- `collect.go` — fetches items from sources via the fetcher registry.
- `build.go` — ranks, filters, deduplicates items into a draft issue.
- `suggest.go` — AI-generated subject/summary.
- `preview.go` — renders the digest for preview.
- `send.go` — dispatches the issue via the email gateway.
- `recap.go` — assembles weekly recap datasets; RecapService stays
  here because it is a concrete service implementation.
- `run.go` — orchestrates the full daily run.

**Stores.** `store/issues`.

**Cross-references.** `services/digest` also imports `domain/news`
(`ItemRepository`, `Item`) to read the items it assembles into issues.
The import direction is one-way: `digest` consumes `news`, never the
reverse.

### `domain/social`

**Purpose.** The social domain models GoDaily's outbound presence on
third-party platforms (Bluesky, LinkedIn, Mastodon, etc.).

**Owns.**

- `Post` — a single outbound post on one platform.
- `PostKind` — `featured` / `new_source` / `recap` / `spotlight` /
  `cta` / `community`.
- `Platform` — the platform identifier (`bluesky`, `linkedin`,
  `mastodon`). A domain enum; concrete HTTP clients live in
  `services/social/platform`.
- `Profile` — the social metadata (display name, mentions, blurb,
  announceability) attached to a news source so it can be spotlighted
  or announced.
- `Profiles` — the curated registry of all profiles.
- `PostRepository` — persistence interface.
- `PostOptions`, `PostResult` — operation DTOs for the daily featured
  post path.
- `RotateOptions` — operation DTO for the Tue/Fri rotation path.

**Does NOT own.**

- ❌ Platform HTTP clients (Bluesky API, LinkedIn API, Mastodon API)
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

### `domain/audience`

**Purpose.** The audience bounded context covers everyone who receives
GoDaily — subscribers and their lifecycle (confirm, unsubscribe,
bounce, suppress). The package is named for the bounded context rather
than the `Subscriber` entity so that `audience.SubscriberRepository`
reads without stutter and the package can absorb future audience-shaped
concepts (segments, preferences, suppression tiers) without a rename.

**Owns.**

- `Subscriber` — the recipient record (email, tokens, confirmation
  and unsubscribe state, bounce/suppression timestamps).
- `SubscriberRepository` — persistence interface, including the
  health side effects (`MarkBounced`, `MarkComplained`,
  `MarkSuppressed`).
- `SubscriberService` — the service interface for the subscription
  lifecycle.

**Does NOT own.**

- ❌ The raw email events that *cause* bouncing → `domain/engagement`.
  Subscriber state is the **derived** outcome; the events themselves
  are engagement data.
- ❌ Provider-specific webhook parsing (Resend payloads, etc.) →
  `pkg/gateway/email`.

**Service.** `services/audience`.

- `service.go` — subscription lifecycle: subscribe, confirm,
  unsubscribe, reactivate. Implements the `SubscriberHealth`
  interface consumed by `services/engagement` so engagement events
  can update subscriber state without a reverse import.

**Stores.** `store/subscribers`.

**Cross-references.** None. The flow is one-way:
`services/engagement` consumes a `SubscriberHealth` interface
satisfied by `services/audience`.

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
  imports `domain/social`. Every `services/X` imports its `domain/X`.
- **Domain packages are named for the bounded context they cover, not
  for any single aggregate inside them.** This is what makes
  `social.PostRepository` and `audience.SubscriberRepository` read
  without stutter — the package name carries the context, the type
  name carries the aggregate.
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

✅ All steps below have been completed on the `claude/beautiful-albattani-g3IX6`
branch. The current tree matches the target package map above.

1. ✅ **Rename `domain/contacts` → `domain/audience`; `services/subscriber` → `services/audience`**
   — so the mirror rule holds and `SubscriberRepository` no longer stutters.
2. ✅ **Split `domain/news` into `domain/news` (items) and `domain/digest` (issues)**
   — `Issue`, `IssueRepository`, the digest `Service` interface, and collect DTOs
   moved to `domain/digest`. `domain/news` retains `Item`, `Source`, and the
   fetcher registry.
3. ✅ **Move social operation DTOs (`PostOptions`, `PostResult`, `RotateOptions`)
   into `domain/social`**
4. ✅ **Move `platform.Name` into `domain/social` as `Platform`**
5. ✅ **Move recap DTOs (`Period`, `Top`, `RankedItem`, `TopOptions`) into
   `domain/digest`**
6. ✅ **Sweep remaining service orphan DTOs** — only `platform.Result` remains in
   `services/social/platform` (kept there intentionally; the plan's scope note
   says to leave `Poster`, `Result`, and clients in the platform package).
7. ✅ **Update `architecture.md`** (this file).
