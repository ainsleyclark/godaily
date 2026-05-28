# pkg/api — Handler Guide

This document describes how to write and test HTTP handlers in GoDaily.

## Structure

Each handler group lives in its own package under `pkg/api/handlers/<name>/`. Every package
contains at minimum:

| File | Purpose |
|---|---|
| `handler.go` | `Handler` struct, `New`, and `Routes` |
| `<route>.go` | One handler per file, named after the route |
| `<route>_test.go` | Tests for that handler |

## Writing a handler

### 1. Define the Handler struct

Declare only the interfaces the handler actually needs — never embed the whole `App`.

```go
type Handler struct {
    things domain.ThingService
    config *env.Config
}

func New(a *godaily.App) *Handler {
    return &Handler{
        things: a.Service.Things,
        config: a.Config,
    }
}

func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
    kit.Get("/things", h.List, auth)
    kit.Post("/things", h.Create, auth)
}
```

### 2. Write the handler

Every handler must return either `api.OK` or `api.Error` — never write raw `c.JSON` calls.

```go
// List handles GET /things.
func (h *Handler) List(c *webkit.Context) error {
    ctx := c.Context()

    things, err := h.things.List(ctx)
    if err != nil {
        slog.ErrorContext(ctx, "Failed to list things", "err", err)
        return api.Error(c, http.StatusInternalServerError, "Failed to retrieve things")
    }

    return api.OK(c, http.StatusOK, things, "Successfully retrieved things")
}
```

**Response rules:**

- `api.OK(c, status, data, message)` — all successful responses
- `api.Error(c, status, message)` — all error responses
- Pass `nil` as data when there is no payload; `api.OK` coerces it to `{}` so the JSON shape stays consistent

**Logging rules:**

Log when the information adds context beyond what the HTTP status code already conveys.
Skip it when the error is self-explanatory from the response.

```go
// Log — worth an audit trail; callers won't know which event type failed or why
slog.ErrorContext(ctx, "Failed to process Resend webhook event", "type", domainEvt.Type, "err", err)

// Log — security event worth recording
slog.WarnContext(ctx, "Rejected Resend webhook with invalid signature", "err", err)

// No log needed — the 400 already tells the story
return api.Error(c, http.StatusBadRequest, "Error reading request body")
```

Always use `slog.WarnContext` / `slog.ErrorContext` (never the bare variants) so the
request context propagates.

## Testing a handler

Use a `Test` struct + `setup` closure. The recorder is stored on the struct so each
sub-test can assert on `deps.Recorder.Code` after calling the handler directly.

```go
func TestHandleThing(t *testing.T) {
    t.Parallel()

    type Test struct {
        Handler  *Handler
        Context  *webkit.Context
        Recorder *httptest.ResponseRecorder
        Things   *mockthings.MockThingService
    }

    setup := func(t *testing.T, req *http.Request) Test {
        t.Helper()

        ctrl := gomock.NewController(t)
        things := mockthings.NewMockThingService(ctrl)
        rec := httptest.NewRecorder()

        return Test{
            Handler:  &Handler{things: things},
            Recorder: rec,
            Context:  webkit.NewContext(rec, req),
            Things:   things,
        }
    }

    t.Run("Returns things on success", func(t *testing.T) {
        t.Parallel()

        req := httptest.NewRequest(http.MethodGet, "/things", nil)
        deps := setup(t, req)
        deps.Things.EXPECT().List(gomock.Any()).Return([]domain.Thing{{ID: 1}}, nil)

        err := deps.Handler.List(deps.Context)

        assert.NoError(t, err)
        assert.Equal(t, http.StatusOK, deps.Recorder.Code)
    })

    t.Run("Service error returns internal server error", func(t *testing.T) {
        t.Parallel()

        req := httptest.NewRequest(http.MethodGet, "/things", nil)
        deps := setup(t, req)
        deps.Things.EXPECT().List(gomock.Any()).Return(nil, errors.New("db down"))

        _ = deps.Handler.List(deps.Context)

        assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
    })
}
```

**Rules:**

- Call `t.Parallel()` at both the top-level test and inside every `t.Run`
- Name sub-tests as human-readable sentences describing the scenario
- Set up `EXPECT()` calls on the `Test` struct fields after `setup`, not inside it
- Use `assert` (not `require`) after the handler call — a failed assertion shouldn't abort sibling cases
- When the handler is expected to return an error, discard it with `_ =` and assert on `Recorder.Code` instead

### Notes

 - Store JSON payloads in `testdata/`. Load them with a small helper.
