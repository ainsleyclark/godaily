// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"path/filepath"
	"testing"

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
		require.NoError(t, conn.PingContext(t.Context()))
	})

	t.Run("Unsupported Scheme", func(t *testing.T) {
		conn, err := New(t.Context(), "ftp://nope", "")
		require.Error(t, err)
		require.Nil(t, conn)
	})
}
