// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package featured

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/mocks/ai"
)

func TestBuildCandidates(t *testing.T) {
	t.Parallel()

	t.Run("Score orders items within a section, not across", func(t *testing.T) {
		t.Parallel()
		// Three discussions plus a lower-scored release. Within the discussion
		// section the higher score comes first; the release still appears
		// (round-robin), proving score never ranks one section above another.
		items := []news.Item{
			{Title: "Quiet", URL: "lo", Source: news.SourceReddit, Tag: news.TagDiscussion, Score: 0.2},
			{Title: "Lively", URL: "hi", Source: news.SourceReddit, Tag: news.TagDiscussion, Score: 0.9},
			{Title: "Middling", URL: "mid", Source: news.SourceReddit, Tag: news.TagDiscussion, Score: 0.5},
			{Title: "Go 1.30 release", URL: "rel", Source: news.SourceGoRelease, Tag: news.TagRelease, Score: 0.1},
		}
		got := buildCandidates(items)
		urls := candidateURLs(got)
		assert.Contains(t, urls, "rel")
		assert.Less(t, indexOf(urls, "hi"), indexOf(urls, "mid"))
		assert.Less(t, indexOf(urls, "mid"), indexOf(urls, "lo"))
	})

	t.Run("Discussion survives a proposal-heavy day", func(t *testing.T) {
		t.Parallel()
		// The user's complaint: a glut of proposals must not crowd out a strong
		// discussion. Round-robin guarantees the discussion reaches the shortlist.
		items := make([]news.Item, 0, 21)
		for i := 0; i < 20; i++ {
			items = append(items, news.Item{
				Title:  "Proposal",
				URL:    fmt.Sprintf("prop-%d", i),
				Source: news.SourceGitHub,
				Tag:    news.TagProposal,
				Score:  0.9,
			})
		}
		items = append(items, news.Item{
			Title: "Future of Go", URL: "disc", Source: news.SourceReddit, Tag: news.TagDiscussion, Score: 0.4,
		})

		got := buildCandidates(items)
		assert.Contains(t, candidateURLs(got), "disc")
	})

	t.Run("No section exceeds the per-section cap", func(t *testing.T) {
		t.Parallel()
		items := make([]news.Item, 0, 10)
		for i := 0; i < 10; i++ {
			items = append(items, news.Item{
				Title: "Proposal", URL: fmt.Sprintf("p-%d", i),
				Source: news.SourceGitHub, Tag: news.TagProposal, Score: float64(i),
			})
		}
		got := buildCandidates(items)
		assert.Len(t, got, perSectionCap)
	})

	t.Run("Accepted proposals are their own section; shipped folds into proposals", func(t *testing.T) {
		t.Parallel()
		// Accepted proposals stand alone in their own section; open and shipped
		// proposals share the Proposals section and together respect the single
		// per-section cap. So the accepted item is always present alongside up to
		// perSectionCap open/shipped proposals.
		items := []news.Item{
			{Title: "Open", URL: "p1", Source: news.SourceGitHub, Tag: news.TagProposal, Score: 0.5},
			{Title: "Accepted", URL: "p2", Source: news.SourceGitHub, Tag: news.TagProposalAccepted, Score: 0.6},
			{Title: "Shipped", URL: "p3", Source: news.SourceGitHub, Tag: news.TagProposalShipped, Score: 0.7},
			{Title: "Another open", URL: "p4", Source: news.SourceGitHub, Tag: news.TagProposal, Score: 0.4},
		}
		got := buildCandidates(items)
		urls := candidateURLs(got)
		assert.Contains(t, urls, "p2", "accepted proposal is its own section and always shortlisted")
		// 1 accepted + 3 open/shipped (capped at perSectionCap) = 4.
		assert.Len(t, got, 1+perSectionCap)
	})

	t.Run("Excluded sections never reach the shortlist", func(t *testing.T) {
		t.Parallel()
		// Jobs, social posts, events, conferences/meet-ups, and trending repos
		// are filtered out regardless of score; only the article remains.
		items := []news.Item{
			{Title: "Senior Go role", URL: "job", Source: news.SourceHNJobs, Tag: news.TagJobs, Score: 5.0},
			{Title: "A toot", URL: "soc", Source: news.SourceMastodon, Tag: news.TagSocial, Score: 4.0},
			{Title: "GopherCon", URL: "conf", Source: news.SourceGitHub, Tag: news.TagConference, Score: 3.0},
			{Title: "Local meet-up", URL: "evt", Source: news.SourceGitHub, Tag: news.TagEvent, Score: 3.0},
			{Title: "Hot repo", URL: "trend", Source: news.SourceGitHubTrending, Tag: news.TagTrending, Score: 9.0},
			{Title: "Real article", URL: "art", Source: news.SourceMedium, Tag: news.TagArticle, Score: 0.4},
		}
		got := buildCandidates(items)
		require.Len(t, got, 1)
		assert.Equal(t, "art", got[0].URL)
	})

	t.Run("All-excluded issue yields no candidates", func(t *testing.T) {
		t.Parallel()
		onlyExcluded := []news.Item{
			{Title: "Job A", URL: "a", Source: news.SourceHNJobs, Tag: news.TagJobs, Score: 3.0},
			{Title: "Meet-up", URL: "b", Source: news.SourceGitHub, Tag: news.TagEvent, Score: 2.0},
		}
		assert.Empty(t, buildCandidates(onlyExcluded))
	})

	t.Run("Total cap respected across diverse sections", func(t *testing.T) {
		t.Parallel()
		// Enough sections, each with plenty of items, to exceed maxCandidates.
		tags := []news.Tag{
			news.TagRelease, news.TagProposal, news.TagArticle, news.TagTutorial,
			news.TagDiscussion, news.TagVideo, news.TagSecurity,
		}
		var items []news.Item
		for _, tag := range tags {
			for i := 0; i < perSectionCap; i++ {
				items = append(items, news.Item{
					URL: fmt.Sprintf("%s-%d", tag, i), Title: "t", Tag: tag, Score: float64(i),
				})
			}
		}
		assert.Len(t, buildCandidates(items), maxCandidates)
	})
}

func candidateURLs(cands []candidate) []string {
	urls := make([]string, len(cands))
	for i, c := range cands {
		urls[i] = c.URL
	}
	return urls
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

func TestFeature(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
	items := []news.Item{
		{Title: "Go 1.30 released", URL: "https://go.dev/blog/go1.30", Source: news.SourceGoRelease, Tag: news.TagRelease, Score: 0.9},
		{Title: "Discussion", URL: "u2", Source: news.SourceReddit, Tag: news.TagDiscussion, Score: 0.4},
	}

	t.Run("Nil prompter errors", func(t *testing.T) {
		t.Parallel()
		_, err := Feature(t.Context(), nil, day, items)
		require.Error(t, err)
	})

	t.Run("Empty items returns ErrNoCandidates", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		// No prompt call expected.

		_, err := Feature(t.Context(), p, day, nil)
		assert.ErrorIs(t, err, ErrNoCandidates)
	})

	t.Run("Happy path parses Featured", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)

		resp := `{"title":"Go 1.30 released","url":"https://go.dev/blog/go1.30","source":"go_release","tag":"release","hook":"Go 1.30 ships generic type inference improvements that simplify constraints."}`
		p.EXPECT().
			Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]byte(resp), nil)

		got, err := Feature(t.Context(), p, day, items)
		require.NoError(t, err)
		assert.Equal(t, "Go 1.30 released", got.Title)
		assert.Equal(t, "https://go.dev/blog/go1.30", got.URL)
		assert.Equal(t, news.SourceGoRelease, got.Source)
		assert.Equal(t, news.TagRelease, got.Tag)
		assert.Contains(t, got.Hook, "Go 1.30")
	})

	t.Run("Strips markdown fences", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)

		fenced := "```json\n" + `{"title":"t","url":"u","source":"s","tag":"article","hook":"h"}` + "\n```"
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte(fenced), nil)

		got, err := Feature(t.Context(), p, day, items)
		require.NoError(t, err)
		assert.Equal(t, "u", got.URL)
	})

	t.Run("AI error wrapped", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("boom"))

		_, err := Feature(t.Context(), p, day, items)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ai")
	})

	t.Run("Empty response errors", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("  "), nil)

		_, err := Feature(t.Context(), p, day, items)
		require.Error(t, err)
	})

	t.Run("Missing required fields errors", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]byte(`{"title":"t","url":"","source":"s","tag":"article","hook":"h"}`), nil)

		_, err := Feature(context.Background(), p, day, items)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing")
	})
}

func TestParseFeatured(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input   string
		wantErr bool
	}{
		"Happy":        {input: `{"title":"t","url":"u","source":"s","tag":"article","hook":"h"}`, wantErr: false},
		"Bad JSON":     {input: `not json`, wantErr: true},
		"Missing hook": {input: `{"title":"t","url":"u","source":"s","tag":"article"}`, wantErr: true},
		"Missing url":  {input: `{"title":"t","url":"","source":"s","tag":"article","hook":"h"}`, wantErr: true},
		"Empty body":   {input: `   `, wantErr: true},
		// Regression: the model emitted a valid object, then second-guessed
		// itself in prose and re-emitted. Trailing content must not break parsing.
		"Trailing self-correction": {
			input:   "{\"title\":\"t\",\"url\":\"u\",\"source\":\"s\",\"tag\":\"article\",\"hook\":\"h\"}\n\nWait, the schema doesn't include score. Let me output correctly:\n\n{\"title\":\"t2\",\"url\":\"u2\"}",
			wantErr: false,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := parseFeatured([]byte(test.input))
			assert.Equal(t, test.wantErr, err != nil)
		})
	}
}
