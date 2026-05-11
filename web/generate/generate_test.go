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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/web/generate"
)

func TestSite(t *testing.T) {
	t.Parallel()

	issue := news.Issue{
		ID:      1,
		Slug:    "2026-04-28",
		Subject: "GoDaily - April 28, 2026",
		Status:  news.IssueStatusSent,
		Items:   []news.Item{},
	}

	tt := map[string]struct {
		mock      func(*mocknews.MockIssueRepository)
		wantErr   bool
		wantFiles []string
	}{
		"Happy path no issues": {
			mock: func(repo *mocknews.MockIssueRepository) {
				repo.EXPECT().List(gomock.Any()).Return([]news.Issue{}, nil)
				repo.EXPECT().Latest(gomock.Any(), 4).Return([]news.Issue{}, nil)
			},
			wantFiles: []string{
				"index.html",
				"sitemap.xml",
				"rss.xml",
				filepath.Join("thank-you", "index.html"),
				filepath.Join("unsubscribed", "index.html"),
				filepath.Join("issues", "index.html"),
			},
		},
		"Happy path with issue": {
			mock: func(repo *mocknews.MockIssueRepository) {
				repo.EXPECT().List(gomock.Any()).Return([]news.Issue{issue}, nil)
				repo.EXPECT().Latest(gomock.Any(), 4).Return([]news.Issue{issue}, nil)
				repo.EXPECT().Find(gomock.Any(), issue.ID).Return(issue, nil)
			},
			wantFiles: []string{
				"index.html",
				"sitemap.xml",
				"rss.xml",
				filepath.Join("thank-you", "index.html"),
				filepath.Join("unsubscribed", "index.html"),
				filepath.Join("issues", "index.html"),
				filepath.Join("issues", issue.Slug, "index.html"),
			},
		},
		"List error": {
			mock: func(repo *mocknews.MockIssueRepository) {
				repo.EXPECT().List(gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		"Latest error": {
			mock: func(repo *mocknews.MockIssueRepository) {
				repo.EXPECT().List(gomock.Any()).Return([]news.Issue{}, nil)
				repo.EXPECT().Latest(gomock.Any(), 4).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		"Find error": {
			mock: func(repo *mocknews.MockIssueRepository) {
				repo.EXPECT().List(gomock.Any()).Return([]news.Issue{issue}, nil)
				repo.EXPECT().Latest(gomock.Any(), 4).Return([]news.Issue{issue}, nil)
				repo.EXPECT().Find(gomock.Any(), issue.ID).Return(news.Issue{}, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocknews.NewMockIssueRepository(ctrl)
			test.mock(repo)

			outDir := t.TempDir()
			staticDir := t.TempDir()
			assetsDir := t.TempDir()

			// Write a sentinel asset file to verify copying.
			require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "app.css"), []byte("body{}"), 0o644))

			err := generate.Site(t.Context(), repo, 0, outDir, staticDir, assetsDir)
			assert.Equal(t, test.wantErr, err != nil)

			for _, f := range test.wantFiles {
				assert.FileExists(t, filepath.Join(outDir, f))
			}
			if !test.wantErr {
				assert.FileExists(t, filepath.Join(outDir, "assets", "app.css"))
			}
		})
	}
}
