// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var schemaTables = []string{"issues", "items", "subscribers", "social_posts", "email_events"}

func TestUp(t *testing.T) {
	t.Run("Creates Tables", func(t *testing.T) {
		conn, err := New(t.Context(), fileURL(t), "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })

		require.NoError(t, Up(t.Context(), conn))

		for _, table := range schemaTables {
			var name string
			err := conn.QueryRowContext(
				t.Context(),
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
		err = conn.QueryRowContext(
			t.Context(),
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('issues','items','subscribers','social_posts','email_events')",
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "all schema tables should be gone after Down")
	})
}
