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
	"net/http"
	"strings"
	"time"

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
	return open(ctx, url, token, idleTimeout)
}

// idleTimeout is how long a pooled Turso connection may sit idle before it is
// discarded instead of reused.
//
// On Vercel a warm instance is frozen between invocations, and Turso can close
// an idle connection in the meantime. Nothing notices until the next query is
// written into the dead socket and fails with `Post ".../v2/pipeline": EOF`.
// net/http checks a pooled connection's idle wall time when it takes it out
// of the pool (HTTP/1.1 and HTTP/2), so a frozen gap longer than this always
// gets a fresh connection. Nothing is retried.
//
// It must stay below Turso's idle timeout, which is not documented. Queries
// within an invocation are milliseconds apart, so a small value costs at most
// a reconnect after a pause, such as an AI call during a build.
const idleTimeout = 2 * time.Second

// open is New with a configurable idle timeout, for tests.
func open(ctx context.Context, url, token string, idle time.Duration) (*sql.DB, error) {
	enforce.NotEqual(url, "", "database url is required")

	var db *sql.DB
	var err error

	if strings.HasPrefix(url, "file:") {
		db, err = sql.Open("sqlite", url)
	} else {
		// Turso gets its own HTTP client, so its idle timeout leaves
		// http.DefaultClient alone. WithHTTPClient comes from the fork in
		// third_party/libsql-client-go.
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.IdleConnTimeout = idle
		opts := []libsql.Option{
			libsql.WithHTTPClient(&http.Client{Transport: transport}),
		}
		if token != "" {
			opts = append(opts, libsql.WithAuthToken(token))
		}
		var connector driver.Connector
		connector, err = libsql.NewConnector(url, opts...)
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
