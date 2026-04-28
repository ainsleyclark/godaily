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

package db

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fileURL builds an ephemeral file: URL backed by t.TempDir(). The DB is
// torn down automatically when the test ends.
func fileURL(t *testing.T) string {
	t.Helper()
	return "file:" + filepath.Join(t.TempDir(), "godaily.db")
}

func TestNew(t *testing.T) {
	t.Run("File URL", func(t *testing.T) {
		conn, err := New(t.Context(), fileURL(t), "")
		require.NoError(t, err)
		require.NotNil(t, conn)
		t.Cleanup(func() { _ = conn.Close() })
		assert.NoError(t, conn.PingContext(t.Context()))
	})

	t.Run("Unsupported Scheme", func(t *testing.T) {
		conn, err := New(t.Context(), "ftp://nope", "")
		assert.Error(t, err)
		assert.Nil(t, conn)
	})
}

func TestMigrate(t *testing.T) {
	t.Run("Applies Schema", func(t *testing.T) {
		conn, err := New(t.Context(), fileURL(t), "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })

		require.NoError(t, Up(t.Context(), conn))

		// Each declared table must exist exactly once.
		for _, table := range []string{"issues", "items", "subscribers"} {
			var got string
			err := conn.QueryRowContext(t.Context(),
				"SELECT name FROM sqlite_master WHERE type='table' AND name = ?", table,
			).Scan(&got)
			require.NoError(t, err, "table %q not found", table)
			assert.Equal(t, table, got)
		}
	})

	t.Run("Idempotent", func(t *testing.T) {
		conn, err := New(t.Context(), fileURL(t), "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })

		require.NoError(t, Up(t.Context(), conn))
		require.NoError(t, Up(t.Context(), conn), "second call must be a no-op")
	})

	t.Run("Up Then Down", func(t *testing.T) {
		conn, err := New(t.Context(), fileURL(t), "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })

		require.NoError(t, Up(t.Context(), conn))
		require.NoError(t, Down(t.Context(), conn))

		var count int
		err = conn.QueryRowContext(t.Context(),
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('issues','items','subscribers')",
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "schema tables should be gone after Down")
	})
}
