# AGENTS.md

## What is godaily?

`godaily` is a Go CLI that delivers a daily digest of the Go community to your
inbox. Each morning it pulls fresh items from Hacker News, Reddit, Lobsters,
Dev.to, Medium, GitHub trending, YouTube, golangbridge and godevblog; hands
them to Anthropic Claude for summarisation and ranking; renders the result
into an email; and ships it via Resend.

It's a single static binary configured entirely through environment variables,
designed to run unattended on a GitHub Actions cron (weekday mornings, 08:00
London).

This file sits next to `README.md`. The README is for users; this file is for
contributors and agents working inside the repo.

## Project Overview

`godaily` is a single-binary Go CLI driven by environment variables and run on
a GitHub Actions schedule (weekday mornings, 08:00 London). The pipeline is:

1. **Fetch** — `internal/source` pulls items from each provider (Hacker News,
   Reddit, Lobsters, Dev.to, Medium, GitHub trending, YouTube, golangbridge,
   godevblog).
2. **Score / register** — `internal/news` defines the shared item model,
   provider registry and scoring/priority logic.
3. **Synthesise** — `internal/synth` calls Anthropic Claude to summarise and
   rank items. Style guidance lives in `internal/synth/style.md`.
4. **Render & send** — `internal/cron` orchestrates the full run; templates
   are in `internal/cron/email.html` and `email.txt`. `internal/email` handles
   Resend delivery.
5. **Entry point** — `cmd/godaily/main.go` wires the CLI (`urfave/cli/v3`).

`cmd/gen-examples` regenerates the fixtures under `examples/`.

## Repository Layout

```
godaily/
├── cmd/
│   ├── godaily/         # CLI entry point
│   └── gen-examples/    # Example regenerator
├── internal/
│   ├── cron/            # Orchestration + email templates
│   ├── db/              # *sql.DB lifecycle + embedded goose migrations
│   ├── email/           # Resend client
│   ├── news/            # Item model, registry, scoring
│   ├── source/          # Per-provider fetchers
│   ├── store/           # sqlc-generated queries + hand-rolled domain logic
│   └── synth/           # Claude synthesis (client, prompt, suggestions)
├── examples/            # Sample raw + rendered digests
└── .github/workflows/   # Daily cron + CI
```

## Build, Test & Lint

Use the `Makefile` targets — don't reinvent commands.

| Command                  | Purpose                                                    |
|--------------------------|------------------------------------------------------------|
| `make build`             | Build the `godaily` binary.                                |
| `make run`               | Run the full digest (sends email).                         |
| `make run-dry`           | Dry run; writes JSON to `examples/news.json`.              |
| `make generate`          | `go generate ./...` then regenerate the example digest.    |
| `make format`            | `go fmt ./...`.                                            |
| `make test`              | Unit tests with coverage (excludes `gen` and `res` paths). |
| `make test-race`         | Tests with `-race`.                                        |
| `make test-integration`  | Hits real provider endpoints (`integration` build tag).    |
| `make lint`              | `golangci-lint run --fix` against `.golangci.yaml`.        |
| `make cover`             | Open the HTML coverage report.                             |
| `make lic`               | Re-stamp MIT license headers on `.go` files.               |
| `make all`               | `lic` + `format` + `lint` + `test`.                        |
| `make sqlc`              | Regenerate query code from `internal/store/*.sql`.         |
| `make migrate-up`        | Apply pending migrations against `TURSO_URL`.              |
| `make migrate-down`      | Roll back the most recent migration.                       |

Before declaring work done: run `make all` (or at minimum `make lint` and
`make test`).

## Configuration

Reads env vars; `.env` in the working directory is auto-loaded.

| Variable             | Purpose                                          |
|----------------------|--------------------------------------------------|
| `RESEND_TOKEN`       | API token for Resend (digest delivery).          |
| `ANTHROPIC_API_KEY`  | Claude API key for synthesis.                    |
| `YOUTUBE_API_KEY`    | YouTube Data API key.                            |
| `GITHUB_TOKEN`       | GitHub API token (trending source).              |
| `EMAIL_SEND_ADDRESS` | Recipient address for the digest.                |
| `TURSO_URL`          | libsql/Turso URL, or `file:./godaily.db` locally.|
| `TURSO_AUTH_TOKEN`   | Turso auth token (omit for `file:` URLs).        |

Copy `.env.example` to `.env` for local dev. Never commit `.env`.

## Tooling & Dependencies

- **Go**: 1.26.x (`go.mod`); golangci-lint pinned to Go 1.25 syntax in
  `.golangci.yaml`.
- **CLI**: `github.com/urfave/cli/v3`.
- **Errors**: `github.com/pkg/errors` (see Errors guideline below).
- **Anthropic SDK**: `github.com/anthropics/anthropic-sdk-go`.
- **Resend**: `github.com/resend/resend-go/v3`.
- **Tests**: `github.com/stretchr/testify` (`assert` + `require`).
- **Internal helpers**: `github.com/ainsleydev/webkit` (notably the `enforce`
  package — see Constructors).

---

# Go Coding Guidelines

The remainder of this file mirrors the Go guidelines from
[ainsley.dev/guidelines/go](https://github.com/ainsleydev/website/tree/main/sites/ainsley-dev/content/guidelines/go).
Treat these as the canonical style for this repo.

## General

### Code Style

- **Formatting**: Use `gofmt` for standard Go formatting (`gofumpt` is
  enabled via `golangci-lint`).
- **File naming**: snake_case for files, test files end with `_test.go`.
  - Integration tests use `_integration_test.go`.
- **Generated files**: `*.gen.go` files are auto-generated — do not edit.
- **Error handling**: Always check and handle errors appropriately.
- **Imports**: Standard library, third-party, then internal imports.

### Interfaces and Abstraction

- Keep interfaces small and focused (single responsibility).
- Prefer returning concrete types unless abstraction is required for testing
  or swapping implementations.
- Document interface expectations explicitly (e.g. *implementations must be
  thread-safe*).

### Defining Types

- Keep structs small and cohesive; split if too many responsibilities.
- Prefer to use the `type` keyword once for multiple type declarations.

```go
type (
    // Environment contains env-specific variable configurations.
    Environment struct {
        Dev        EnvVar `json:"dev,omitempty"`
        Staging    EnvVar `json:"staging,omitempty"`
        Production EnvVar `json:"production,omitempty"`
    }
    // EnvVar is a map of variable names to their configurations.
    EnvVar map[string]EnvValue
    // EnvValue represents a single env variable configuration.
    EnvValue struct {
        Source EnvSource `json:"source"`
        Value  any       `json:"value,omitempty"`
        Path   string    `json:"path,omitempty"`
    }
)
```

### Naming Conventions

- **Integration tests**: end with `_integration_test.go`.
- **Generated files**: `*.gen.go`.
- **Interfaces**: often end in `-er` (`Reader`, `Writer`, `Store`).
- **Package names**: short, lowercase, single-word names where possible.

## Comments

- Document all exported types, functions and constants with Go doc comments.
- All comments end with a full stop, including inline and multi-line.
- Within function bodies, only keep comments that explain *why* something is
  done, not *what* — the code shows what.
- Keep high-level comments that explain flow or purpose of a section
  (e.g. *Try loading template file first*, *Fallback to static markdown*).
- Remove obvious comments that just restate the code.

```go
// Generator handles file scaffolding operations for WebKit projects.
type Generator interface {
    // Bytes writes raw bytes to a file with optional configuration.
    //
    // Returns an error when the file failed to write.
    Bytes(path string, data []byte, opts ...Option) error
}
```

## Function Patterns

### Context

Use `context.Context` as the first parameter for functions that perform I/O
or can be cancelled.

```go
func Run(ctx context.Context, cmd Command) (Result, error) {
    select {
    case <-ctx.Done():
        return Result{}, ctx.Err()
    default:
        // Execute command.
    }
}
```

## Constructors & Funcs

Constructors validate all required dependencies using `enforce` helpers and
return pointer types. Use them in the context of being called from a `cmd`
package.

### New

- Prefer `NewX()` constructors over global initialisation. If it's the only
  constructor in the package, name it `New()`.

### Enforce

Use `github.com/ainsleydev/webkit/pkg/enforce`:

- Not nil → `enforce.NotNil()`
- Boolean → `enforce.True()`
- Equality → `enforce.Equal()` / `enforce.NotEqual()`
- No error → `enforce.NoError()`

These provide simple runtime guarantees and exit the program with a helpful
message if a condition fails.

```go
func NewGenerator(fs afero.Fs, manifest *manifest.Tracker, printer *printer.Console) *FileGenerator {
    enforce.NotNil(fs, "file system is required")
    enforce.NotNil(manifest, "manifest is required")
    enforce.NotNil(printer, "printer is required")

    return &FileGenerator{
        Printer:  printer,
        fs:       fs,
        manifest: manifest,
    }
}
```

### Context (in constructors)

Same rule as for functions — `context.Context` first if I/O or cancellation
is involved.

## Control Flow

### Maps Over Switch

Prefer maps with function values over switch statements when dispatching by
string or integer key — more maintainable, extensible and testable.

**Prefer**

```go
type handlerFunc func(input Request) (Response, error)

var handlers = map[string]handlerFunc{
    "create": handleCreate,
    "update": handleUpdate,
    "delete": handleDelete,
}

func dispatch(action string, req Request) (Response, error) {
    handler, exists := handlers[action]
    if !exists {
        return Response{}, fmt.Errorf("unknown action: %s", action)
    }
    return handler(req)
}
```

**Avoid**

```go
func dispatch(action string, req Request) (Response, error) {
    switch action {
    case "create":
        return handleCreate(req)
    case "update":
        return handleUpdate(req)
    case "delete":
        return handleDelete(req)
    default:
        return Response{}, fmt.Errorf("unknown action: %s", action)
    }
}
```

### Exceptions

- Type switches (`switch v := value.(type)`) are appropriate for type
  assertions.
- Switch is acceptable for complex conditions or ranges.
- Small, simple switches (2–3 cases) where a map adds unnecessary complexity.

## Errors

- Always check errors; never ignore with `_` unless absolutely necessary.
- If ignoring, add a comment explaining why.
- Return errors up the stack; don't just log and continue unless appropriate.
- Prioritise clarity over depth of stack trace — add context that helps
  debugging, not repetition.

### Domain Error Types

Define custom errors to give context and allow type-based handling, instead
of relying on generic `fmt.Errorf`. Use them only for domain-specific cases
where inspecting or handling by type is useful.

```go
type ErrInsufficientBalance struct {
    Amount float64
}

func (e ErrInsufficientBalance) Error() string {
    return fmt.Sprintf("insufficient balance: need %.2f", e.Amount)
}

if balance < withdrawAmount {
    return ErrInsufficientBalance{Amount: withdrawAmount}
}
```

### Using errors.Wrap

Always use `errors.Wrap` from `github.com/pkg/errors` to add context. Use
`fmt.Errorf` only when more than one non-error argument needs formatting.

```go
func LoadConfig(fs afero.Fs, path string) (*Config, error) {
    data, err := afero.ReadFile(fs, path)
    if err != nil {
        return nil, errors.Wrap(err, "reading config file")
    }
    return parseConfig(data)
}

func ValidatePort(port int) error {
    if port < 1024 || port > 65535 {
        return fmt.Errorf("invalid port %d: must be between 1024 and 65535", port)
    }
    return nil
}
```

## Testing

All Go tests are written in one of two ways:

1. As a **test table**, or
2. As individual **`t.Run` subtests**.

Use test tables for most cases. Use `t.Run` subtests when:

- The number of input arguments in the table exceeds **3**, or
- Assertions get complex (we **never** use `if` statements in test tables), or
- Individual cases need unique setup that would force a setup function in the
  table.

### General Rules

- Always call `t.Parallel()` at the top of every test function and within each
  subtest, unless:
  - It's an integration test (`_integration_test.go`).
  - It performs file I/O, shell commands, or interacts with SOPS or OS files.
  - It has the potential to fail with `--race`.
- Always use `t.Context()` when a `context.Context` is required, not
  `context.Background()`.
- All assertions use the `assert` library (and `require` when necessary).
- Prefer one assertion per test when possible.
- Never use `else` blocks — use assert logic instead.
- Never redeclare variables like `test := test` (no shadowing).
- Use `got` for the variable holding actual results.
- Test names: capitalised first word, spaces between words, not full title
  case (e.g. `"Payload default"`, `"GoLang explicit true"`).
- Always include all relevant test cases, including edge and error
  conditions.
- If 100% coverage isn't possible, explain *why* in a brief note above the
  test function (no inline comments).

### Test Organisation

- **One test function per exported function/method** — add new cases as
  subtests within the existing test function rather than spawning new ones.
- Only create a new test function when:
  - Testing a distinctly different aspect (e.g. `TestTracker_Add` vs
    `TestTracker_Save`).
  - The original would become unwieldy (>200 lines).
- Group related cases with descriptive subtest names.
- Aim for comprehensive coverage within each test function rather than
  fragmenting tests.

### Test Tables

- Format: `map[string]struct{}` keyed by the test name.
- Loop reads `for name, test := range tt` — the variable is `tt`.
- Field names:
  - `input` for inputs
  - `want` for expected outputs
  - `wantErr` if the function returns an error
- Error assertion:

```go
assert.Equal(t, test.wantErr, err != nil)
```

- No `if`, `switch`, or branching logic inside the loop.
- No code comments inside the test unless explaining *why*.

```go
func TestExample(t *testing.T) {
    t.Parallel()

    tt := map[string]struct {
        input string
        want  string
    }{
        "Example Case": {input: "foo", want: "bar"},
    }

    for name, test := range tt {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            got := DoSomething(test.input)
            assert.Equal(t, test.want, got)
        })
    }
}
```

### Subtests with `t.Run`

- Use `require` for preconditions (setup or function calls that must not
  fail).
- Use `assert` for validation of expected outputs.
- Use `t.Log()` to describe sections within a subtest instead of comments
  when assertions are bigger.
- Maintain readability and determinism — tests should clearly convey intent
  and run independently.
- Each test is self-contained with no shared mutable state.

```go
func TestApp_OrderedCommands(t *testing.T) {
    t.Parallel()

    t.Run("Missing Skipped", func(t *testing.T) {
        t.Parallel()

        app := &App{Commands: map[Command]CommandSpec{}}
        commands := app.OrderedCommands()
        assert.Len(t, commands, 0)
    })

    t.Run("Default Populated", func(t *testing.T) {
        t.Parallel()

        app := &App{}
        err := app.applyDefaults()
        require.NoError(t, err)

        commands := app.OrderedCommands()
        require.Len(t, commands, 4)
        assert.Equal(t, "format", commands[0].Name)
    })
}
```

### Mocking

Mocks are introduced only when a test depends on an **external interface** or
system boundary — Terraform execution, encryption providers, file I/O
wrappers, etc.

- Prefer fakes or real in-memory types where possible.
- Place generated mocks under `internal/mocks/`, prefixed `Mock`
  (e.g. `MockInfraManager`).
- Clean up with `defer ctrl.Finish()`; avoid over-mocking.
- Use [`gomock`](https://pkg.go.dev/go.uber.org/mock/gomock).
- Generate into `internal/mocks/`:

```bash
go tool go.uber.org/mock/mockgen -source=gen.go -destination ../mocks/fs.go -package=mocks
```

### Setup Functions

- If a test repeats setup logic (creating `App` instances, defaults, common
  fixtures), look for a `setup(t)` function.
- If none exists, create one to encapsulate reusable logic.
- `setup(t)` must:
  - Accept `t *testing.T`.
  - Return values needed by multiple subtests.
  - Call `t.Helper()` first.

```go
func setup(t *testing.T) *App {
    t.Helper()

    app := &App{Name: "web", Type: AppTypeGoLang, Path: "./"}
    err := app.applyDefaults()
    require.NoError(t, err)

    return app
}

func TestApp_OrderedCommands(t *testing.T) {
    t.Parallel()

    t.Run("Default Populated", func(t *testing.T) {
        t.Parallel()

        app := setup(t)
        commands := app.OrderedCommands()
        require.Len(t, commands, 4)
        assert.Equal(t, "format", commands[0].Name)
    })
}
```

---

## Database

Stage 1 introduced two packages:

- `internal/db` owns the `*sql.DB` lifecycle. `db.New(ctx, url, token)` opens
  a connection — Turso URLs (`libsql://`, `https://`, `wss://`) go through
  the `tursodatabase/libsql-client-go` driver; `file:` URLs use the pure-Go
  `modernc.org/sqlite` driver. `db.Migrate` / `db.Down` apply embedded
  `goose` migrations from `internal/db/migrations/`.
- `internal/store` owns typed query access. `*.sql` files are sqlc input,
  `*.sql.go` files are sqlc output (regenerate with `make sqlc` or
  `go generate ./internal/store/...`). Hand-written `*.go` files only
  appear when there is logic beyond a single query — e.g.
  `subscribers.go` generates the confirm/unsubscribe tokens before
  delegating to the generated `CreateSubscriber`.

### Adding a migration

1. Create `internal/db/migrations/000N_name.sql` with `-- +goose Up` and
   `-- +goose Down` blocks.
2. If it changes the schema shape used by queries, run `make sqlc` so the
   generated models stay in sync.
3. Ship via `make migrate-up` (production) or against a local `file:` DB
   for dev.

## Repo-Specific Notes for Agents

- **Don't commit secrets.** `.env` is git-ignored; never add real tokens to
  `.env.example`, fixtures or tests.
- **Examples are fixtures, not docs.** `examples/` is regenerated by
  `make generate` / `cmd/gen-examples` — edit the generator, not the output.
- **Synthesis style** lives in `internal/synth/style.md`. Treat it as the
  source of truth for digest tone — change it there, not in prompt strings
  scattered across code.
- **Source providers** (`internal/source/*.go`) follow a common shape: a
  fetcher + a transform step. Add new sources by mirroring an existing one
  and wiring through `internal/news/registry.go` and `sources.go`.
- **Integration tests** in `internal/source/source_integration_test.go` hit
  the network. Skip via the default `make test`; run explicitly with
  `make test-integration`.
- **License headers** are managed by `make lic` (`addlicense`). Don't write
  them by hand.
- **Before opening a PR**: `make all` should pass cleanly.
