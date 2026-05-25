# Social Service

The social service handles two posting paths that share the same per-platform publish loop:

- **Featured** (`Post`) — runs Mon–Fri. Loads the day's digest, picks the most engaging item via AI, and reframes it for each configured platform.
- **Rotation** (`Rotate`) — runs Tue/Wed/Fri. Walks a day-specific candidate list, picks the first eligible one, and generates a platform-specific post without referencing the digest.

## Architecture

```
Service.Rotate()
  └── pickCandidates()         pick day's candidate list (or ForceKind)
        └── Candidate.Eligible()   check DB / cooldowns; return CandidateContext
              └── Candidate.Generate()   call rotation prompt generator
                    └── rotation.Kind()  run() → AI → parse text
                          └── publish()  post to each platform, write social_posts row
```

## How to add a new rotation post kind

### 1. Register the kind constant

Add the new kind to `pkg/domain/news/social.go`:

```go
const SocialPostKindMyKind SocialPostKind = "my_kind"
```

If the candidate uses a subject key for idempotency, no migration is needed — the subject is stored as a free-form string in `social_posts`.

### 2. Add the prompt generator

Create `pkg/services/social/prompts/rotation/my_kind.go`:

```go
package rotation

// MyKindPayload is the input the candidate passes to the generator.
type MyKindPayload struct {
    // ... fields the AI needs
}

func MyKind(ctx context.Context, p ai.Prompter, platform social.Platform, payload MyKindPayload) (string, error) {
    return run(ctx, p, platform, myKindSystem, payload)
}

const myKindSystem = `...kind-specific guidance for the AI...`
```

`run()` handles platform profile selection, AI call, JSON parsing, and char-limit checks.

### 3. Add the candidate

Create `pkg/services/social/candidates/my_kind.go` implementing the `Candidate` interface:

```go
package candidates

type MyKind struct {
    posts news.SocialPostRepository
}

func NewMyKind(posts news.SocialPostRepository) *MyKind { ... }

func (c *MyKind) Kind() news.SocialPostKind { return news.SocialPostKindMyKind }

// Eligible checks prerequisites (DB state, cooldowns) and returns a
// CandidateContext when ready. Return (_, false, nil) to skip silently.
func (c *MyKind) Eligible(ctx context.Context, now time.Time) (socialsvc.CandidateContext, bool, error) {
    // Set Subject for idempotency; use platformAnchor for the DB probe.
    return socialsvc.CandidateContext{Subject: "my_kind:...", Payload: rotation.MyKindPayload{...}}, true, nil
}

func (c *MyKind) Generate(ctx context.Context, p ai.Prompter, platform socialgw.Platform, cctx socialsvc.CandidateContext) (string, error) {
    payload := cctx.Payload.(rotation.MyKindPayload)
    return rotation.MyKind(ctx, p, platform, payload)
}
```

### 4. Wire the candidate

In `pkg/app.go`, add to `buildRotationCandidates`:

```go
out = append(out, candidates.NewMyKind(posts))
```

### 5. Add day routing

In `pkg/services/social/rotation.go`, update `pickCandidates` to include the new kind in the appropriate weekday's `orderedByKinds` call:

```go
case time.Tuesday:
    return orderedByKinds(
        s.candidates,
        news.SocialPostKindNewSource,
        news.SocialPostKindSpotlight,
        news.SocialPostKindCTA,
        news.SocialPostKindMyKind, // add here
    ), nil
```
