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
		// Store bound time.Time values as "2006-01-02 15:04:05.999999999-07:00",
		// matching exactly what the libsql/Turso driver persists in production
		// (see libsql hrana value.go). The modernc default is time.Time.String()
		// ("… +0000 UTC"), which SQLite's date functions cannot parse — so without
		// this the local/dev/test DB stores timestamps in a format that diverges
		// from production and breaks datetime()-based time-window comparisons.
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		// _timezone=UTC keeps timestamps in UTC when scanned back out, matching the
		// rest of the codebase (which works exclusively in UTC) and the default
		// time.Time.String() round-trip behaviour this replaces.
		db, err = sql.Open("sqlite", url+sep+"_time_format=sqlite&_timezone=UTC")
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
