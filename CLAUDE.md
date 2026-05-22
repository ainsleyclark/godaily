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

```
go test ./...
golangci-lint run ./... --fix --config=.golangci.yaml
```

The formatter is `gofumpt` (stricter than `gofmt`). The lint step applies it automatically with `--fix`.
