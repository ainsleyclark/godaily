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

