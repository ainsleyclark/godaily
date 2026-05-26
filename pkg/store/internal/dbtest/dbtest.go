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

	url := "file:" + filepath.Join(t.TempDir(), "godaily.db")
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
