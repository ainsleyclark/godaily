// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/web/generate"
)

func TestSite(t *testing.T) {
	t.Parallel()

	issue := digest.Issue{
		ID:      1,
		Slug:    "2026-04-28",
		Subject: "GoDaily - April 28, 2026",
		Status:  digest.IssueStatusSent,
		Items:   []news.Item{},
	}

	sent := digest.IssueStatusSent
	draft := digest.IssueStatusDraft
	sentOpts := digest.IssueListOptions{Status: &sent}
	draftOpts := digest.IssueListOptions{Status: &draft}

	tt := map[string]struct {
		mock      func(*mockdigest.MockIssueRepository)
		wantErr   bool
		wantFiles []string
		// wantMissing entries are "file:substring" pairs asserting the named
		// output file does not contain the given substring.
		wantMissing []string
	}{
		"Happy path no issues": {
			mock: func(repo *mockdigest.MockIssueRepository) {
				repo.EXPECT().List(gomock.Any(), sentOpts).Return([]digest.Issue{}, nil)
				repo.EXPECT().Latest(gomock.Any(), 4).Return([]digest.Issue{}, nil)
				repo.EXPECT().List(gomock.Any(), draftOpts).Return([]digest.Issue{}, nil)
			},
			wantFiles: []string{
				"index.html",
				"sitemap.xml",
				"rss.xml",
				filepath.Join("thank-you", "index.html"),
				filepath.Join("unsubscribed", "index.html"),
				filepath.Join("issues", "index.html"),
				filepath.Join("browse", "index.html"),
			},
		},
		"Happy path with issue": {
			mock: func(repo *mockdigest.MockIssueRepository) {
				repo.EXPECT().List(gomock.Any(), sentOpts).Return([]digest.Issue{issue}, nil)
				repo.EXPECT().Latest(gomock.Any(), 4).Return([]digest.Issue{issue}, nil)
				repo.EXPECT().List(gomock.Any(), draftOpts).Return([]digest.Issue{}, nil)
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
				filepath.Join("browse", "index.html"),
				filepath.Join("og", "home.png"),
				filepath.Join("og", "issues", issue.Slug+".png"),
			},
		},
		"List error": {
			mock: func(repo *mockdigest.MockIssueRepository) {
				repo.EXPECT().List(gomock.Any(), sentOpts).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		"Latest error": {
			mock: func(repo *mockdigest.MockIssueRepository) {
				repo.EXPECT().List(gomock.Any(), sentOpts).Return([]digest.Issue{}, nil)
				repo.EXPECT().Latest(gomock.Any(), 4).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		"Find error": {
			mock: func(repo *mockdigest.MockIssueRepository) {
				repo.EXPECT().List(gomock.Any(), sentOpts).Return([]digest.Issue{issue}, nil)
				repo.EXPECT().Latest(gomock.Any(), 4).Return([]digest.Issue{issue}, nil)
				repo.EXPECT().List(gomock.Any(), draftOpts).Return([]digest.Issue{}, nil)
				repo.EXPECT().Find(gomock.Any(), issue.ID).Return(digest.Issue{}, errors.New("db error"))
			},
			wantErr: true,
		},
		"Draft issue rendered as live copy": {
			mock: func(repo *mockdigest.MockIssueRepository) {
				draftIssue := digest.Issue{
					ID:      2,
					Slug:    "2026-04-29",
					Subject: "GoDaily - April 29, 2026",
					Status:  digest.IssueStatusDraft,
					Items:   []news.Item{},
				}
				repo.EXPECT().List(gomock.Any(), sentOpts).Return([]digest.Issue{}, nil)
				repo.EXPECT().Latest(gomock.Any(), 4).Return([]digest.Issue{}, nil)
				repo.EXPECT().List(gomock.Any(), draftOpts).Return([]digest.Issue{draftIssue}, nil)
				repo.EXPECT().Find(gomock.Any(), draftIssue.ID).Return(draftIssue, nil)
			},
			wantFiles: []string{
				// The draft surfaces as its own live-copy page and OG image...
				filepath.Join("issues", "2026-04-29", "index.html"),
				filepath.Join("og", "issues", "2026-04-29.png"),
			},
			wantMissing: []string{
				// ...but never appears in the sitemap or RSS feed until sent.
				"sitemap.xml:2026-04-29",
				"rss.xml:2026-04-29",
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mockdigest.NewMockIssueRepository(ctrl)
			test.mock(repo)

			// The browse page (rendered for cases that get past the early
			// issue queries) pulls item data and the latest issue ID.
			repo.EXPECT().Latest(gomock.Any(), 1).Return([]digest.Issue{}, nil).AnyTimes()
			items := mocknews.NewMockItemRepository(ctrl)
			items.EXPECT().List(gomock.Any(), gomock.Any()).Return([]news.Item{}, nil).AnyTimes()
			items.EXPECT().Count(gomock.Any()).Return(int64(0), nil).AnyTimes()
			items.EXPECT().CountMatching(gomock.Any(), gomock.Any()).Return(int64(0), nil).AnyTimes()
			items.EXPECT().SourceCounts(gomock.Any()).Return([]news.SourceCount{}, nil).AnyTimes()
			items.EXPECT().TagCounts(gomock.Any()).Return([]news.TagCount{}, nil).AnyTimes()

			outDir := t.TempDir()
			staticDir := t.TempDir()
			assetsDir := t.TempDir()

			// Write a sentinel asset file to verify copying.
			require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "app.css"), []byte("body{}"), 0o644))

			err := generate.Site(t.Context(), repo, items, 0, outDir, staticDir, assetsDir)
			assert.Equal(t, test.wantErr, err != nil)

			for _, f := range test.wantFiles {
				assert.FileExists(t, filepath.Join(outDir, f))
			}
			for _, pair := range test.wantMissing {
				name, substr, _ := strings.Cut(pair, ":")
				data, readErr := os.ReadFile(filepath.Join(outDir, name))
				require.NoError(t, readErr)
				assert.NotContains(t, string(data), substr)
			}
			if !test.wantErr {
				assert.FileExists(t, filepath.Join(outDir, "assets", "app.css"))
			}
		})
	}
}
