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
	"net/http"
	"strings"

	"github.com/ainsleydev/webkit/pkg/enforce"
	"github.com/pkg/errors"
	// Turso DB Driver, for more info, visit:
	// https://docs.turso.tech/sdk/go/quickstart
	//_ "turso.tech/database/tursogo"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
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
		url = url + "?authToken=" + token
		db, err = sql.Open("libsql", url)
		if err == nil {
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

// CloseIdleConnections drops the idle sockets the libsql HTTP driver has
// pooled. Call it at the start of each serverless invocation.
//
// The driver sends every query as a POST on http.DefaultClient, which keeps
// idle keep-alive sockets between requests. On Vercel a warm instance is
// frozen between invocations, and Turso can close those sockets in the
// meantime. The next invocation then writes its first query into a dead
// socket and fails with `Post ".../v2/pipeline": EOF`, before Turso sees it.
//
// Closing idle sockets up front means each invocation dials a fresh
// connection instead. Nothing is retried or replayed, so no query can run
// twice, and it works the same over HTTP/1.1 and HTTP/2. Connections in use
// by other in-flight requests are not touched. The cost is one new
// connection per invocation; within an invocation, sockets are still reused.
//
// It also closes idle sockets to other hosts that use http.DefaultClient,
// which only means they reconnect on next use.
func CloseIdleConnections() {
	http.DefaultClient.CloseIdleConnections()
}
