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

package generate_test

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/domain/news"
	"github.com/ainsleyclark/godaily/web/generate"
)

func TestOGImages(t *testing.T) {
	t.Parallel()

	issue := news.Issue{
		ID:      3,
		Slug:    "2026-05-12",
		Subject: "GoDaily – May 12, 2026",
		Status:  news.IssueStatusSent,
		SentAt:  time.Date(2026, 5, 12, 8, 0, 0, 0, time.UTC),
		Items: []news.Item{
			{Title: "Go 1.26 released"},
			{Title: "Proposal: generic parse functions"},
			{Title: "Why Go modules are great"},
			{Title: "An extra article beyond the three shown"},
		},
	}

	ctrl := gomock.NewController(t)
	repo := mocknews.NewMockIssueRepository(ctrl)
	repo.EXPECT().List(gomock.Any(), gomock.Any()).Return([]news.Issue{issue}, nil)
	repo.EXPECT().Latest(gomock.Any(), 4).Return([]news.Issue{issue}, nil)
	repo.EXPECT().Find(gomock.Any(), issue.ID).Return(issue, nil)

	outDir := t.TempDir()
	staticDir := t.TempDir()
	assetsDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "app.css"), []byte("body{}"), 0o644))

	require.NoError(t, generate.Site(t.Context(), repo, 0, outDir, staticDir, assetsDir))

	cases := map[string]string{
		"home OG image":  filepath.Join(outDir, "og", "home.png"),
		"issue OG image": filepath.Join(outDir, "og", "issues", issue.Slug+".png"),
	}

	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f, err := os.Open(path)
			require.NoError(t, err)
			defer f.Close()
			_, err = png.Decode(f)
			assert.NoError(t, err, "file should be a valid PNG")
		})
	}
}
