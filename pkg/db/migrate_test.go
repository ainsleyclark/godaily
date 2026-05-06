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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var schemaTables = []string{"issues", "items", "subscribers"}

func TestUp(t *testing.T) {
	t.Run("Creates Tables", func(t *testing.T) {
		conn, err := New(t.Context(), fileURL(t), "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })

		require.NoError(t, Up(t.Context(), conn))

		for _, table := range schemaTables {
			var name string
			err := conn.QueryRowContext(t.Context(),
				"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
			).Scan(&name)
			require.NoError(t, err, "table %q missing after Up", table)
			assert.Equal(t, table, name)
		}
	})

	t.Run("Idempotent", func(t *testing.T) {
		conn, err := New(t.Context(), fileURL(t), "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })

		require.NoError(t, Up(t.Context(), conn))
		require.NoError(t, Up(t.Context(), conn), "second Up must be a no-op")
	})
}

func TestDown(t *testing.T) {
	t.Run("Removes Tables", func(t *testing.T) {
		conn, err := New(t.Context(), fileURL(t), "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })

		require.NoError(t, Up(t.Context(), conn))
		for {
			err := Down(t.Context(), conn)
			if err != nil {
				break
			}
		}

		var count int
		err = conn.QueryRowContext(t.Context(),
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('issues','items','subscribers')",
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "all schema tables should be gone after Down")
	})
}
