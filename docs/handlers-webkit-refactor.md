# Handlers refactor: webkit + structs + route-level middleware

## Context

The API package today routes every handler through a callback pattern:

```go
func HandleBuild(w http.ResponseWriter, r *http.Request) {
    api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
        // ...
    })(w, r)
}
```

The `*godaily.App` is smuggled into every request via `context.WithValue` (`api.WithApp` / `api.GetApp`), and auth + rate limiting are baked into `api.Handle` / `api.HandleAuth`. This means:

- Every handler is double-indented behind a closure and reads awkwardly.
- Each handler couples to the entire `*App`, hiding its real dependencies.
- Tests have to set `api.WithApp(ctx, app)` on every request to avoid a panic in `GetApp`.
- Auth is invisible at the route table — you can't tell whether `/digest/build` is protected without opening `build.go`.
- Routing is on stdlib `http.ServeMux`. Response helpers (`api.JSON`, `api.OK`, `api.Error`) and decoder wiring are hand-rolled.

The refactor does three things at once:

1. **Migrate the HTTP layer to [webkit](https://github.com/ainsleydev/webkit)** (`github.com/ainsleydev/webkit/pkg/webkit`). Webkit is a thin chi-based framework: handlers are `func(c *webkit.Context) error`, middleware is `webkit.Plug` (per-route variadic), and it provides built-in response helpers (`c.JSON`, `c.NoContent`, `c.BindJSON`) and a centralised `ErrorHandler`.
2. **Group handlers into per-domain `Handler` structs** with a `Routes(kit *webkit.Kit)` method — exactly the seekscuba pattern referenced in the request.
3. **Apply auth/rate-limit as webkit plugs at route registration**, so the route table tells you what's protected.

Outcome: handlers read as `func (h *Handler) Build(c *webkit.Context) error`, dependencies are explicit, `WithApp`/`GetApp`/`AppHandler`/`HandleAuth`/`Handle` are deleted, and the home-grown response/error helpers go away in favour of webkit's.

## Approach

### 1. Add webkit dependency, replace stdlib mux

- `go get github.com/ainsleydev/webkit/pkg/webkit` and import it in `pkg/api/mux/mux.go`.
- Replace `http.NewServeMux()` with `webkit.New()`.
- The trailing-slash strip currently done in the outer handler stays — wrap the underlying `http.Handler` exposed by the Kit (verify the exact method during implementation; chi's `ServeHTTP` is on the router). The Vercel entrypoint `api/index.go` keeps working: it serves the same `http.Handler` shape, just sourced from webkit instead of `http.NewServeMux`.

### 2. New plugs package: `pkg/api/plugs`

Move cross-cutting concerns out of `pkg/api/app.go` and into composable `webkit.Plug`s (signature `func(webkit.Handler) webkit.Handler`).

- `plugs.Auth(secret string) webkit.Plug` — replaces `api.HandleAuth`. Uses the existing `api.Authenticated` helper. If `secret == ""` it's a no-op (preserves current dev/CI behaviour). On failure returns a webkit error so the centralised `ErrorHandler` formats the 401.
- `plugs.RateLimit(limiter *api.RateLimiter) webkit.Plug` — wraps the existing `pkg/api/limiter.go` logic (keep the limiter type as-is, only the wrapper shape changes).
- Apply rate-limit globally via `kit.Plug(plugs.RateLimit(api.Limiter))`; apply auth per-route via the variadic plug arg on `kit.Get(...)`.

Delete `api.Handle`, `api.HandleAuth`, `api.AppHandler`, `api.WithApp`, `api.GetApp`, `appContextKey` from `pkg/api/app.go`. Delete the hand-rolled `api.JSON`, `api.OK`, `api.Error` — those are replaced by `c.JSON`, `c.NoContent`, and returned errors. Keep `api.Authenticated`, `api.Decoder`, `api.ParseDateWindow`, `api.IsWeekend` — still useful inside handlers.

Configure a single webkit `ErrorHandler` (in `mux.go`) that:
- Maps known sentinel errors to status codes (e.g. `errors.Is(err, ErrUnauthorized) → 401`).
- Logs server errors via `slog`.
- Renders the body as `{"error": "..."}` to match the existing wire format so frontend callers don't break.

### 3. Handler structs (narrow fields, `*App` constructor)

Each existing folder under `pkg/api/handlers/` becomes a `Handler` struct that **holds only the narrow services it actually uses**, with a `New(*godaily.App)` constructor that extracts those fields. Handler bodies reference `h.runner`, not `h.app.Runner` — the struct fields are the dependency contract; `New` is just glue.

This keeps the mux wiring to one line per domain while keeping handlers and their tests decoupled from the full `*App`.

Folders → structs:

- `handlers/digest/` → `digest.Handler{ runner, subscribers, emailEvents, cache, slack, config }`
- `handlers/metrics/` → `metrics.Handler{ metricsRepo, issuesRepo }` (folds nested `metrics/issues/` in — see below)
- `handlers/social/` → `social.Handler{ social *socialsvc.Service, slack, config }`
- `handlers/issues/` → `issues.Handler{ issuesRepo }`
- `handlers/items/` → `items.Handler{ itemsRepo }`
- `handlers/webhooks/` → `webhooks.Handler{ emailEvents, config }`
- `handlers/healthz.go` → stays a plain func (no deps).

Representative shape (one file per package, e.g. `handlers/digest/handler.go`):

```go
type Handler struct {
    runner      news.Service
    subscribers contacts.SubscriberService
    emailEvents *svcengagement.EventService
    cache       cache.Store
    slack       slack.Sender
    config      *env.Config
}

func New(a *godaily.App) *Handler {
    return &Handler{
        runner:      a.Runner,
        subscribers: a.Subscribers,
        emailEvents: a.EmailEvents,
        cache:       a.Cache,
        slack:       a.Slack,
        config:      a.Config,
    }
}
```

Each existing `HandleBuild` / `HandleSend` / etc. becomes a webkit handler method:

```go
func (h *Handler) Build(c *webkit.Context) error {
    ctx := c.Context()
    now := time.Now().UTC()
    if !force && api.IsWeekend(now) {
        hook.Heartbeat(ctx, h.config.BetterStackBuildHeartbeatURL)
        return c.NoContent(http.StatusOK)
    }
    if err := h.runner.Build(ctx, now); err != nil {
        return fmt.Errorf("build failed: %w", err) // ErrorHandler renders 500
    }
    hook.Heartbeat(ctx, h.config.BetterStackBuildHeartbeatURL)
    return c.NoContent(http.StatusOK)
}
```

Conversions are mechanical:
- `a.Runner` → `h.runner`, `a.Config` → `h.config`, etc.
- `api.OK(w)` → `return c.NoContent(http.StatusOK)`.
- `api.JSON(w, status, v)` → `return c.JSON(status, v)`.
- `api.Error(w, status, msg)` → `return &webkit.HTTPError{Code: status, Message: msg}` (exact type per `errors.go`; verify during implementation).
- `r.URL.Query()` stays the same (accessed via `c.Request.URL.Query()`); the existing `api.Decoder` (gorilla/schema) keeps working.
- Path params: `r.PathValue("slug")` → `c.Param("slug")`.

Each handler package also gets a `Routes(kit *webkit.Kit)` method, mirroring the seekscuba example:

```go
func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
    kit.Get("/build",  h.Build, auth)
    kit.Get("/send",   h.Send,  auth)
    kit.Get("/collect", h.Collect, auth)
    kit.Post("/subscribe", h.Subscribe)         // public
    kit.Get("/confirm", h.Confirm)              // public
    // ...
}
```

For `metrics/issues/slug.go` (nested package): fold into the same `metrics.Handler` and expose `(h *Handler) IssueBySlug`. Drop the nested package — the file split was only because handler names collided in the package, which the struct method form removes.

### 4. Mux wiring (`pkg/api/mux/mux.go`)

The mux becomes the place where dependencies are wired and plugs are applied. Auth and rate limiting are documented at the route table via webkit's `Group` and the variadic plug arg.

```go
func Handler(app *godaily.App) http.Handler {
    kit := webkit.New()
    kit.Plug(plugs.RateLimit(api.Limiter)) // global
    auth := plugs.Auth(app.Config.APISecret)

    digestH  := digest.New(app)
    metricsH := metrics.New(app)
    socialH  := social.New(app)
    issuesH  := issues.New(app)
    itemsH   := items.New(app)
    webhookH := webhooks.New(app)

    kit.Get("/healthz", handlers.Healthz)

    kit.Group("/digest", func(k *webkit.Kit) { digestH.Routes(k, auth) })
    kit.Group("/metrics", func(k *webkit.Kit) { metricsH.Routes(k, auth) })
    kit.Group("/social",  func(k *webkit.Kit) { socialH.Routes(k, auth) })

    kit.Get("/issues/{slug}", issuesH.BySlug, auth)
    kit.Get("/items/{id}",    itemsH.ByID,    auth)
    kit.Post("/webhooks/resend", webhookH.Resend) // self-verifies signature

    kit.Post("/subscribe", digestH.Subscribe)
    kit.Get("/confirm",    digestH.Confirm)
    // `/unsubscribe` currently accepts any method — register both GET and POST explicitly.

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Existing Vercel trailing-slash strip stays.
        if p := r.URL.Path; p != "/" && strings.HasSuffix(p, "/") {
            r2 := r.Clone(r.Context())
            r2.URL.Path = p[:len(p)-1]
            r = r2
        }
        kit.ServeHTTP(w, r) // exact method name on *Kit to be verified against webkit's wrap.go
    })
}
```

Notes:
- The `WithApp` injection in the outer handler is deleted. No more context smuggling.
- Public vs protected for each existing route follows current code: anything wrapped in `api.HandleAuth` today gets the `auth` plug; anything wrapped in plain `api.Handle` is registered without it.
- Rate-limit applied globally as a `kit.Plug(...)` preserves today's behaviour (every route limited). If we want to skip it on webhooks, register them in a separate sub-group without inheriting the global plug — decide during implementation.

### 5. Tests

All ~20 `_test.go` files currently do:

```go
r = r.WithContext(api.WithApp(r.Context(), a))
HandleBuild(w, r)
```

Becomes:

```go
h := &digest.Handler{
    runner: mockRunner,
    slack:  mockSlack,
    config: &env.Config{},
}
c := webkit.NewContext(w, r)
err := h.Build(c)
```

Tests bypass `New(*App)` entirely and construct the struct directly with only the fields each test exercises — no more assembling a half-filled `*godaily.App`. Tests already live in the same package as their handler (e.g. `package digest`), so unexported fields are accessible. Keep the existing mocks in `pkg/mocks/` — only the wiring at the top of each test changes.

For tests that want to verify auth/rate-limit behaviour, add focused plug tests in `pkg/api/plugs/*_test.go` rather than testing them indirectly through every handler.

## Critical files

- **New:** `pkg/api/plugs/plugs.go` (Auth, RateLimit) — webkit.Plug wrappers around existing helpers.
- **Modify:** `pkg/api/app.go` — delete (`WithApp`/`GetApp`/`Handle`/`HandleAuth`/`AppHandler`/`appContextKey` all go). File can likely be removed entirely.
- **Modify:** `pkg/api/response.go` — delete `JSON`/`OK`/`Error` (replaced by webkit). If the file becomes empty, remove it.
- **Modify:** `pkg/api/mux/mux.go` — switch to webkit Kit, register routes via per-handler `Routes(kit, auth)`, configure `ErrorHandler` for `{"error": "..."}` body shape. Focal point of the refactor.
- **New:** `go.mod` — add `github.com/ainsleydev/webkit` dependency.
- **Pattern repeated per domain folder:** add `handler.go` with the struct + `New(*godaily.App)` + `Routes(kit, auth)`. Convert each existing `Handle*` function into a method with webkit signature `func(*webkit.Context) error`. Update the matching `_test.go`. Representative example pair: `pkg/api/handlers/digest/build.go` + `pkg/api/handlers/digest/build_test.go`.
- **Collapse:** `pkg/api/handlers/metrics/issues/slug.go` folds into `pkg/api/handlers/metrics/`.
- **Verify (no functional change expected):** `api/index.go` — should still receive an `http.Handler` from `mux.Handler(app)`.

## Reuse

- `api.Authenticated` (`pkg/api/auth.go`) — the new `plugs.Auth` is a thin wrapper around it.
- `api.Limiter` and `api.RateLimiter.Limit` (`pkg/api/limiter.go`) — re-exposed via `plugs.RateLimit`, no behaviour change to the limiter itself.
- `api.Decoder`, `api.ParseDateWindow`, `api.IsWeekend` — unchanged, still called directly from handler methods.
- `webkit.Context.JSON`, `webkit.Context.NoContent`, `webkit.Context.BindJSON`, `webkit.Context.Param` — replace the hand-rolled equivalents.
- All `pkg/mocks/` mocks — unchanged; tests just stop assembling them into a `*godaily.App`.

## Verification

1. `go test ./...` — every existing handler test must pass after migration. Tests are the strongest behavioural contract here.
2. `golangci-lint run ./... --fix --config=.golangci.yaml` — per CLAUDE.md.
3. `go build ./...` to confirm the mux wiring compiles with the new constructors and webkit.
4. Smoke test locally by running the API and hitting:
   - `GET /healthz` → 200 (public).
   - `GET /digest/build` without `Authorization` → 401 (auth plug engaged), body shape `{"error": "..."}`.
   - `GET /digest/build` with `Authorization: Bearer <secret>` → expected behaviour.
   - A request burst → 429 from rate-limit plug.
   - Path-param routes (e.g. `GET /issues/{slug}`, `GET /items/{id}`) resolve correctly under chi's pattern syntax.
5. Confirm `rg -n "WithApp|GetApp|HandleAuth|api\\.Handle\\(|api\\.JSON|api\\.OK|api\\.Error" pkg/` returns zero hits when the refactor lands.

## Open items to verify during implementation

- **Webkit's `http.Handler` accessor.** webkit.Kit needs to expose either `ServeHTTP` or an `.HTTPHandler()` for the Vercel entrypoint. If not present, wrap chi's underlying router directly.
- **Webkit error type.** Confirm the exact error type for status-coded responses (likely `webkit.HTTPError` in `errors.go`); adapt `api.Error` callsites to it.
- **Chi path-param syntax** (`{slug}`) is the same as Go 1.22 stdlib, so existing patterns shouldn't change. Confirm during smoke test.

## Out of scope

- Reorganising service packages or the `*godaily.App` struct itself — only the `api/handlers/*` and `pkg/api/*` layers change.
- Changing the Vercel entrypoint (`api/index.go`) beyond a no-op verification.
- Replacing gorilla/schema with webkit's `schema.go` helpers — keep `api.Decoder` to minimise diff; a follow-up can migrate it.
