// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbtest

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/db"
)

// Setup creates a new SQLite *sql.DB with all migrations applied,
// ready for testing. The DB is file-backed under t.TempDir() so it
// is torn down automatically when the test ends.
func Setup(t *testing.T) (context.Context, *sql.DB, func()) {
	t.Helper()

	// Allow some time for the test to run.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*3)

	// _time_format=sqlite makes the modernc driver persist bound time.Time values
	// as "2006-01-02 15:04:05.999999999-07:00" — exactly what the libsql/Turso
	// driver writes in production — instead of its default time.Time.String()
	// ("… +0000 UTC"), which SQLite's date functions cannot parse. _timezone=UTC
	// keeps timestamps in UTC when scanned back out. Together they let the test DB
	// faithfully reproduce production storage so datetime()-based time-window
	// queries are exercised against the real format.
	url := "file:" + filepath.Join(t.TempDir(), "godaily.db") + "?_time_format=sqlite&_timezone=UTC"
	conn, err := db.New(ctx, url, "")
	require.NoError(t, err, "opening sqlite database")

	require.NoError(t, db.Up(ctx, conn), "applying migrations")

	cleanup := func() {
		err = conn.Close()
		require.NoError(t, err, "closing sqlite database")
		cancel()
	}

	return ctx, conn, cleanup
}
