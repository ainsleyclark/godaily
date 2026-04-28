// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
	"github.com/tursodatabase/libsql-client-go/libsql"

	// Pure-Go SQLite driver, registered as "sqlite". Used directly for
	// file-backed URLs (local dev, tests) and as the underlying file
	// connector for libsql when given a file: URL.
	_ "modernc.org/sqlite"
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

	if strings.HasPrefix(url, "file:") {
		db, err := sql.Open("sqlite", url)
		if err != nil {
			return nil, errors.Wrap(err, "opening sqlite database")
		}

		if err = db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, errors.Wrap(err, "pinging sqlite database")
		}
		return db, nil
	}

	var opts []libsql.Option
	if token != "" {
		opts = append(opts, libsql.WithAuthToken(token))
	}

	connector, err := libsql.NewConnector(url, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "creating libsql connector")
	}

	db := sql.OpenDB(connector)
	if err = db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, errors.Wrap(err, "pinging libsql database")
	}

	return db, nil
}
