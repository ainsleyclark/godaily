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

package generate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSitemap(t *testing.T) {
	t.Parallel()

	issue := digest.Issue{
		ID:     1,
		Slug:   "2026-04-28",
		SentAt: time.Date(2026, 4, 28, 8, 0, 0, 0, time.UTC),
	}

	tt := map[string]struct {
		w       website
		outDir  func(t *testing.T) string
		wantErr bool
		checks  []string
	}{
		"No issues": {
			w:      website{},
			outDir: func(t *testing.T) string { t.Helper(); return t.TempDir() },
			checks: []string{
				`https://godaily.dev/`,
				`<priority>1.0</priority>`,
				`urlset`,
			},
		},
		"With issues": {
			w:      website{Issues: []digest.Issue{issue}},
			outDir: func(t *testing.T) string { t.Helper(); return t.TempDir() },
			checks: []string{
				`https://godaily.dev/`,
				`https://godaily.dev/issues/2026-04-28/`,
				`<lastmod>2026-04-28</lastmod>`,
				`<priority>0.8</priority>`,
			},
		},
		"Write error": {
			w:       website{},
			outDir:  func(t *testing.T) string { t.Helper(); return filepath.Join(t.TempDir(), "nonexistent") },
			wantErr: true,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			outDir := test.outDir(t)
			err := sitemap(test.w, outDir)
			assert.Equal(t, test.wantErr, err != nil)

			if !test.wantErr {
				path := filepath.Join(outDir, "sitemap.xml")
				assert.FileExists(t, path)
				data, readErr := os.ReadFile(path) //nolint:gosec
				require.NoError(t, readErr)
				for _, want := range test.checks {
					assert.Contains(t, string(data), want)
				}
			}
		})
	}
}
