# Domain Refactor Plan

This document is a follow-on to [architecture.md](./architecture.md). It
captures two refinements that came out of an architectural review and
that the existing migration plan does not cover:

1. **Split `domain/news` into `domain/news` (items) and `domain/digest`
   (issues)** — so every service has a same-named domain twin and the
   "one allowed exception" for `services/digest` disappears.
2. **Move service operation DTOs (`*Options`, `*Result`, etc.) into
   their matching `domain/` package** — so adapters (CLI, HTTP
   handlers) only need to import the domain to call the service.

Together these remove the last domain/service naming asymmetries and
let consumers of a service import a single package.

Read [architecture.md](./architecture.md) first — the layering
principle, the mirror rule, and the per-domain "Owns / Does NOT own"
lists there are the source of truth. This doc lists the moves; that
doc explains why each piece lives where it does.

## Why split `news` into `news` + `digest`

`pkg/domain/news` currently holds two distinct aggregates:

- **`Item`** — a single news article from a `Source`. Upstream of any
  digest. Owned by the `news` bounded context.
- **`Issue`** — a newsletter edition assembled from items. Owned by
  the `digest` bounded context (build / score / send / recap).

That conflation forces `services/digest` to import `domain/news` for
`IssueRepository`, which reads as "the digest service uses the news
domain's issue repository" — a small but real semantic mismatch. It is
also the reason architecture.md needs a "one allowed exception to the
mirror rule" for `services/digest` pairing with `domain/news`.

After the split:

| Domain | Service | Stores |
|---|---|---|
| `domain/news` | (no service) — read by `digest` and `social` | `store/items` |
| `domain/digest` | `services/digest` | `store/issues` |

The exception goes away. `services/digest` imports `domain/digest` for
`Issue`/`IssueRepository` *and* `domain/news` for `Item`/`ItemRepository`
— which honestly describes what it does: the digest service assembles
a digest from news items.

## Why move operation DTOs into the domain

Today, `pkg/cmd/social.go` has to alias two packages:

```go
social    "github.com/ainsleyclark/godaily/pkg/domain/social"
socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
```

because the CLI needs both `social.PostKind` (domain enum, one line)
*and* `socialsvc.RotateOptions` / `socialsvc.PostResult` (operation
DTOs). The collision is structural to having matching domain/service
package names.

The fix is a principle, not a workaround:

> **Service operation types live in the domain.** A service package
> owns *behaviour* (the methods, the orchestration, the
> implementation). The shape of the request and response is part of
> the domain's public language and lives in `pkg/domain/X/`.

The codebase already does this for one workflow: `news.CollectOptions`
and `news.CollectResponse` live in `pkg/domain/news/digest.go`
alongside the `news.Service` interface they belong to. This plan
generalises that pattern to every service.

Once the DTOs move, the CLI only imports `pkg/domain/social/`, no
alias is needed, and `a.Social.Rotate(...)` is called with
`social.RotateOptions{...}` — the same `social` import that gives you
`social.PostKind`. Go does not require you to import a package just
to call a method on a field whose type lives there.

The same principle resolves the `domain/social` ↔ `services/social`
naming collision for every other adapter (HTTP handlers in `pkg/api/`,
future webhook adapters), not just the CLI.

## The principle, restated

After this refactor, the contents of each layer are:

| Layer | Owns |
|---|---|
| `pkg/domain/X/` | Entities, value types, repository interfaces, **service interfaces, operation DTOs (`*Options`, `*Result`, `*Response`)**, domain-language enums (e.g. `platform.Name`). |
| `pkg/services/X/` | Concrete service structs that implement the domain's service interface. Workflows, orchestration, AI prompts, scheduling. Provider-specific adapters under `services/X/<provider>/`. |
| `pkg/store/<table>/` | SQL-backed implementations of `domain/X` repository interfaces. |

Adapters (CLI, HTTP, webhook handlers) depend on `domain/` only.
`pkg/app.go` is the one place that wires concrete services from
`services/` into the app struct.

## Migration steps

Each numbered step is a self-contained PR. After every step, run:

```
go test ./...
golangci-lint run ./... --fix --config=.golangci.yaml
```

Update [architecture.md](./architecture.md) in the same PR where the
change lands — keep the per-domain reference in that doc in sync.

### 1. Create `pkg/domain/digest/` and move issue types

Create the package and move from `pkg/domain/news/`:

- `issues.go` → `pkg/domain/digest/issue.go` — `Issue`, `IssueStatus`,
  `IssueRepository`, and the issue-shaped `ListOptions` (rename to
  `digest.IssueListOptions` if a generic `ListOptions` survives in
  `domain/news`).
- `digest.go` → `pkg/domain/digest/service.go` — the `Service`
  interface, `CollectOptions`, `CollectResponse`, and `SourceItems` if
  it is referenced only from the digest workflow. (Verify; if
  `SourceItems` is used by `domain/news` collectors, leave it in
  `news`.)
- `score.go` → `pkg/domain/digest/score.go` if the scoring logic
  operates on `Issue`. If it scores `Item`s, leave it in `news`.

Update package declarations to `package digest`. Add a one-line
package comment matching the new "Purpose" entry to be added to
architecture.md.

Update imports across the codebase:

- `pkg/services/digest/*.go`
- `pkg/store/issues/*.go`
- `pkg/mocks/digest/*` (regenerate, see below)
- `pkg/api/`, `web/handlers/`
- `pkg/cmd/`

Regenerate mocks for any moved interface:

```
go generate ./pkg/domain/...
```

Update `sqlc.yaml` if any override path references `domain/news`
issue types; rerun `sqlc generate` if needed.

### 2. Move social operation DTOs into `domain/social`

Move:

- `pkg/services/social/service.go::PostOptions` → `pkg/domain/social/operations.go`
- `pkg/services/social/service.go::PostResult` → same file
- `pkg/services/social/rotation.go::RotateOptions` → same file

Keep `Service`, `Rotate`, `Post`, `WithCandidates`, and all
candidate/scheduling logic in `pkg/services/social/`.

Update `pkg/cmd/social.go`:

- Remove the `socialsvc` alias and the import of
  `pkg/services/social`.
- Replace `socialsvc.PostOptions{...}`, `socialsvc.RotateOptions{...}`,
  `socialsvc.PostResult` with the `social.` equivalents (already
  imported from `domain/social`).

Update `pkg/api/social/` and any other call sites the same way.

If `RotateOptions` is referenced from web handlers or tests, update
those imports too.

### 3. Move `platform.Name` into `domain/social`

`platform.Name` (`bluesky`, `linkedin`, `mastodon`) is domain
vocabulary — what platforms GoDaily posts to — not an implementation
detail. The HTTP clients stay in `pkg/services/social/platform/` as
adapters.

Move:

- `pkg/services/social/platform/platform.go::Name` and its constants
  (`Bluesky`, `LinkedIn`, `Mastodon`) → `pkg/domain/social/platform.go`.
- Keep the `Poster` interface, `Result` struct, and concrete clients
  (`bluesky/`, `linkedin/`, `mastodon/`) in
  `pkg/services/social/platform/`. Update them to use
  `social.Platform` (or `social.PlatformName` if you prefer the
  longer form) from the domain.

After this step, `RotateOptions.Platforms` is `[]social.Platform` and
the field lives in `domain/social` with no cross-package dependency
on `services/social/platform`.

### 4. Move digest recap DTOs into `domain/digest`

Move from `pkg/services/digest/recap.go`:

- `Period`, `Top`, `RankedItem`, `TopOptions` → `pkg/domain/digest/recap.go`.

Keep `RecapService` and `NewRecapService` in
`pkg/services/digest/recap.go`. The service implements the workflow;
the DTOs are domain language.

`RankedItem` currently embeds `engagement.ItemMetrics`. That stays —
`domain/digest` is allowed to import `domain/engagement` the same way
`engagement` holds foreign-key IDs back to digest. Cross-domain
*value* types embedded by ID-bearing aggregates are fine; the rule
about referencing other domains by ID applies to entities, not to
read-side aggregates.

### 5. Sweep the remaining services for orphan DTOs

After steps 2 and 4, search for any remaining `Options` / `Result` /
`Request` / `Response` types defined under `pkg/services/`:

```
grep -rn '^type \(.*Options\|.*Result\|.*Request\|.*Response\)' pkg/services/
```

Each match is either:

- An operation DTO that belongs in the matching `pkg/domain/X/` — move
  it.
- A purely internal helper struct (e.g. `confirmData` in
  `services/subscriber`) — leave it.

Document the call: a type is "internal" only if it never appears in
the signature of an exported service method. Anything an adapter can
see crossing the service boundary moves to the domain.

### 6. Update architecture.md

- Replace the [Package map (target)](./architecture.md#package-map-target)
  table:

  | Domain | Service | Store |
  |---|---|---|
  | `news` | — | `items` |
  | `digest` | `digest` | `issues` |
  | `social` | `social` | `socialposts`, `socialmetrics` |
  | `subscriber` | `subscriber` | `subscribers` |
  | `engagement` | `engagement` | `emailevents`, `engagement` |

- Delete the "one allowed exception to the mirror rule" note — every
  service now matches its domain.
- Add a new `### domain/digest` section under "Per-domain reference"
  with its own Owns / Does NOT own / Service / Stores / Cross-references.
- Update the existing `### domain/news` section: it no longer owns
  `Issue`, `IssueStatus`, the digest `Service` interface, or the
  collect DTOs. `Item`, `Source`, `Sources`, `Registry`, `Tag`,
  `Author`, `Score` (if item-scoped), and `ItemRepository` remain.
- Update the layering principle table to list "service interfaces,
  operation DTOs" as something `pkg/domain/X/` owns.
- Delete the corresponding step from the existing migration plan if
  it has been completed, or leave it if it has not.

## Verification

After every step, in addition to tests and lint:

```
# Every services/X imports its domain/X
for d in pkg/services/*/; do
  name=$(basename "$d")
  grep -q "domain/$name" "$d"*.go || echo "MISMATCH: $name"
done

# No adapter imports a services package just for DTOs
grep -rn '"github.com/ainsleyclark/godaily/pkg/services/' pkg/cmd/ pkg/api/ web/
```

The second check should only return lines that call service methods
or construct services (mainly `pkg/app.go`). Adapters importing a
services package *just* to reference a DTO type indicates a DTO that
still needs to move.

## Scope notes

- This plan does **not** address the `domain/contacts` →
  `domain/subscriber` rename. That is already covered by step 1 of
  the existing migration plan in architecture.md and should land
  before or independently of this work.
- This plan does **not** introduce a service interface in
  `domain/social`, `domain/subscriber`, or `domain/engagement` if
  none exists today. Add them only when an adapter needs to depend
  on the contract rather than the concrete struct — usually for
  testing. The DTO moves above are valuable independently.
