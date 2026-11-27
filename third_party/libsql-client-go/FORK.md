# libsql-client-go (godaily fork)

A copy of [github.com/tursodatabase/libsql-client-go](https://github.com/tursodatabase/libsql-client-go)
at `v0.0.0-20260528064733-9d5d30a29a60`, with one patch. `go.mod` at the
repository root points at it with a `replace` directive.

## Why

The upstream driver sends every request to Turso through `http.DefaultClient`
and offers no way to pass in a client. GoDaily gives Turso its own HTTP client
with a short idle timeout, so stale keep-alive connections are discarded
without changing the client every other HTTP caller in the process shares.
See `idleTimeout` in `pkg/db/db.go`.

## The patch

`godaily.patch` is the full diff against upstream (apply with `patch -p1`).
It adds one option:

```go
libsql.NewConnector(url, libsql.WithHTTPClient(client))
```

Without the option the driver behaves exactly as upstream (it falls back to
`http.DefaultClient`).

It also fixes the seven findings gosec reports in upstream code, because CI
scans this directory: six unchecked `Close` errors in the WebSocket client
(which GoDaily doesn't use) and a bounds check before an `int` to `int32`
conversion in batch requests. Neither changes behaviour.

## Updating

Upstream is deprecated, so new releases are unlikely. Turso's docs still
recommend this driver for remote libSQL databases. To take an upstream
change, copy the new release over `libsql/`, `sqliteparser/`,
`sqliteparserutils/`, `go.mod`, `go.sum` and `LICENSE`, re-apply
`godaily.patch`, and run `gosec -exclude-generated ./...` and `go test ./...` here and at the repository root.
