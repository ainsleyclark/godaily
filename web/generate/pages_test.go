// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/web/generate"
)

func TestOGImages(t *testing.T) {
	t.Parallel()

	issue := digest.Issue{
		ID:      3,
		Slug:    "2026-05-12",
		Subject: "GoDaily – May 12, 2026",
		Status:  digest.IssueStatusSent,
		SentAt:  time.Date(2026, 5, 12, 8, 0, 0, 0, time.UTC),
		Items: []news.Item{
			{Title: "Go 1.26 released"},
			{Title: "Proposal: generic parse functions"},
			{Title: "Why Go modules are great"},
			{Title: "An extra article beyond the three shown"},
		},
	}

	ctrl := gomock.NewController(t)
	repo := mockdigest.NewMockIssueRepository(ctrl)
	repo.EXPECT().List(gomock.Any(), gomock.Any()).Return([]digest.Issue{issue}, nil)
	repo.EXPECT().Latest(gomock.Any(), 4).Return([]digest.Issue{issue}, nil)
	repo.EXPECT().Latest(gomock.Any(), 1).Return([]digest.Issue{}, nil).AnyTimes()
	repo.EXPECT().Find(gomock.Any(), issue.ID).Return(issue, nil)

	items := mocknews.NewMockItemRepository(ctrl)
	items.EXPECT().List(gomock.Any(), gomock.Any()).Return([]news.Item{}, nil).AnyTimes()
	items.EXPECT().Count(gomock.Any()).Return(int64(0), nil).AnyTimes()
	items.EXPECT().CountMatching(gomock.Any(), gomock.Any()).Return(int64(0), nil).AnyTimes()
	items.EXPECT().SourceCounts(gomock.Any()).Return([]news.SourceCount{}, nil).AnyTimes()
	items.EXPECT().TagCounts(gomock.Any()).Return([]news.TagCount{}, nil).AnyTimes()

	outDir := t.TempDir()
	staticDir := t.TempDir()
	assetsDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "app.css"), []byte("body{}"), 0o644))

	require.NoError(t, generate.Site(t.Context(), repo, items, 0, outDir, staticDir, assetsDir))

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
