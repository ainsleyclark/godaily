// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package db owns the *sql.DB lifecycle and schema evolution for godaily.
//
// It does not contain query logic — that lives in internal/store. The split
// keeps connection wiring and migrations free of typed access concerns.
package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"

	"github.com/ainsleydev/webkit/pkg/enforce"
	"github.com/pkg/errors"
	// Turso DB Driver, for more info, visit:
	// https://docs.turso.tech/sdk/go/quickstart
	//_ "turso.tech/database/tursogo"
	"github.com/tursodatabase/libsql-client-go/libsql"
)

// New opens a *sql.DB against the given URL.
//
// Supported schemes:
//   - libsql://, https://, http://, wss://, ws:// — Turso remote (token required).
//   - file: — local SQLite file (no token), useful for dev and tests.
//
// The connection is verified with PingContext before returning.
func New(ctx context.Context, url, token string) (*sql.DB, error) {
	enforce.NotEqual(url, "", "database url is required")

	var db *sql.DB
	var err error

	if strings.HasPrefix(url, "file:") {
		db, err = sql.Open("sqlite", url)
	} else {
		var connector driver.Connector
		connector, err = libsql.NewConnector(url, remoteOptions(token)...)
		if err == nil {
			db = sql.OpenDB(connector)
			// The libsql HTTP driver marks a connection streamClosed=true after
			// any non-transactional query (e.g. the ping below). ResetSession
			// never resets that flag, so the post-ping idle connection hands
			// driver.ErrBadConn back on the next use. Discard idle connections
			// immediately so the poisoned connection is never re-pooled.
			db.SetMaxIdleConns(0)
		}
	}

	if err != nil {
		return nil, err
	}

	if err = db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, errors.Wrap(err, "pinging libsql database")
	}

	return db, nil
}

// remoteOptions configures the libsql HTTP driver for Turso.
//
// The driver sends every query as a POST to /v2/pipeline on
// http.DefaultClient, which keeps idle keep-alive sockets between requests.
// On Vercel a warm instance is frozen between invocations, and Turso closes
// those idle sockets in the meantime. The next query reuses a dead socket and
// fails with `Post ".../v2/pipeline": EOF` before Turso ever sees it. Go's
// transport already retries that case on a fresh connection, but only for
// requests it considers replayable, which excludes a plain POST.
//
// An Idempotency-Key header marks the request replayable (see
// net/http.Request.isReplayable), so the transport retries it on a fresh
// connection, and only when a reused connection failed.
//
// Caveats:
//   - This covers HTTP/1.1 only, which is what the production error (a bare
//     EOF) indicates. Over HTTP/2 a dead connection surfaces as
//     "unexpected EOF" and is not retried.
//   - Turso is not known to act on the header. If it ever deduplicated on it,
//     this constant key would be wrong and must be removed.
//   - If the connection dies after Turso ran the query but before the reply
//     arrives, the transport replays it, so a non-idempotent write could run
//     twice. That is far rarer than the stale-socket case this fixes.
func remoteOptions(token string) []libsql.Option {
	opts := []libsql.Option{
		libsql.WithRequestHeaders(map[string]string{
			"Idempotency-Key": "godaily-libsql",
		}),
	}
	if token != "" {
		opts = append(opts, libsql.WithAuthToken(token))
	}
	return opts
}
