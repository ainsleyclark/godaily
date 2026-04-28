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

package dbtest

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/internal/db"
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
