# GoDaily — Development Notes

## Code generation

### sqlc (database queries)

`pkg/store/internal/sqlc/` is fully generated — **never hand-edit it**.

When you change any `.sql` file under `pkg/store/*/query.sql` or add a migration:

1. If the migration adds a nullable timestamp column, add a `*time.Time` override to `sqlc.yaml`:
   ```yaml
   - column: "table.column_name"
     go_type:
       type: "Time"
       import: "time"
       pointer: true
   ```
2. Run:
   ```
   sqlc generate
   ```

### OpenAPI → dashboard types

The OpenAPI 3.1 contract (`docs/openapi/swagger.{yaml,json}`) is generated from
the swag annotations on the API handlers via `make openapi`. The SvelteKit
dashboard consumes that contract: `dashboard/src/lib/api/schema.d.ts` is
generated from the spec by `openapi-typescript` (`pnpm gen:api`), and the typed
HTTP client in `dashboard/src/lib/api/client.ts` is built on `openapi-fetch`.

`schema.d.ts` is fully generated — **never hand-edit it**. Friendly type aliases
live in `dashboard/src/lib/api/types.ts`.

When you change any API handler's swag annotations (params, response shapes,
new routes), regenerate both the contract and the dashboard types together:

```
make openapi-ts
```

CI fails if either the committed contract or `schema.d.ts` is out of date, so
always commit the regenerated files. Because swag does not emit `required`
markers, every field in `schema.d.ts` is optional; `types.ts` asserts presence
for response payloads — model genuine optionality on the Go struct instead.

### Mocks (gomock)

`pkg/mocks/` is fully generated. When you add or remove a method on any interface that has a `//go:generate` directive, run:

```
go generate ./pkg/domain/... ./pkg/services/...
```

### DB migrations (Goose)

Every migration file must use Goose annotations or it will fail to parse:

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE ...;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ...;
-- +goose StatementEnd
```

## Testing & linting

Always run these commands locally before committing:

```
go test ./...
golangci-lint run ./... --fix --config=.golangci.yaml
```

**Important:** The `--fix` flag auto-corrects formatting and most linting issues. If the linter fails in CI, it's usually because formatting wasn't applied locally.

### Formatting details

- The formatter is `gofumpt` (stricter than `gofmt`). Running `golangci-lint run ./... --fix` applies it automatically.
- gofumpt enforces strict spacing and line breaking rules. Always run the lint command with `--fix` to apply corrections before pushing.
- If linting fails in CI on formatting (e.g. "File is not properly formatted (gofumpt)"), run the lint command locally with `--fix` and commit the changes.

### Troubleshooting lint failures

**Go version mismatch**: If golangci-lint complains about Go version, ensure your local Go version matches `go.mod` (currently 1.26.3).

**File formatting issues**: Run `golangci-lint run ./... --fix --config=.golangci.yaml` — this corrects formatting in-place. Always do this before committing.

## Commit style

Use conventional commits with the following format:

```
<type>: <description>
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`

**Rules:**
- Capitalize the first letter of the description
- Keep it concise and descriptive
- Examples: `feat: Add user authentication`, `docs: Update API documentation`, `fix: Resolve memory leak in cache`
