// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mcpserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

var (
	sentAt = time.Date(2026, 11, 18, 8, 0, 0, 0, time.UTC)

	sentIssue = digest.Issue{
		ID:      42,
		Slug:    "2026-11-18",
		Subject: "Go 1.27 freeze, iterators in practice",
		Status:  digest.IssueStatusSent,
		Summary: "Today's top Go news.",
		SentAt:  sentAt,
		Items: []news.Item{
			{ID: 7, Source: news.SourceHN, Title: "Go 1.27 freeze", URL: "https://go.dev/blog", Tag: "official", InDigest: true, Position: 1, Published: sentAt.Add(-time.Hour)},
		},
	}
)

func TestListIssues(t *testing.T) {
	t.Parallel()

	t.Run("Defaults and summary mapping", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		want := digest.IssueListOptions{Page: 1, PerPage: 20}
		h.Issues.EXPECT().Count(gomock.Any(), want).Return(int64(1), nil)
		h.Issues.EXPECT().List(gomock.Any(), want).Return([]digest.Issue{sentIssue}, nil)

		out := decode[ListIssuesOutput](t, h, "list_issues", map[string]any{})
		assert.Equal(t, int64(1), out.Total)
		assert.Equal(t, int64(1), out.Page)
		assert.Equal(t, int64(20), out.PerPage)
		require.Len(t, out.Issues, 1)
		assert.Equal(t, IssueSummary{
			ID:      42,
			Slug:    "2026-11-18",
			Subject: sentIssue.Subject,
			Status:  "sent",
			Summary: sentIssue.Summary,
			SentAt:  sentAt,
		}, out.Issues[0])
	})

	t.Run("Status filter and per_page cap", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		match := gomock.Cond(func(o digest.IssueListOptions) bool {
			return o.Status != nil && *o.Status == digest.IssueStatusDraft && o.Page == 3 && o.PerPage == 100
		})
		h.Issues.EXPECT().Count(gomock.Any(), match).Return(int64(0), nil)
		h.Issues.EXPECT().List(gomock.Any(), match).Return(nil, nil)

		out := decode[ListIssuesOutput](t, h, "list_issues", map[string]any{"status": "draft", "page": 3, "per_page": 500})
		assert.Empty(t, out.Issues)
		assert.Equal(t, int64(100), out.PerPage)
	})

	t.Run("Repository error is masked", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		h.Issues.EXPECT().Count(gomock.Any(), gomock.Any()).Return(int64(0), errors.New("libsql: connection reset"))

		msg := toolError(t, h, "list_issues", map[string]any{})
		assert.Contains(t, msg, "failed to count issues")
		assert.NotContains(t, msg, "libsql", "internal errors must not leak")
	})
}

func TestGetIssue(t *testing.T) {
	t.Parallel()

	t.Run("By numeric ID", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(sentIssue, nil)

		out := decode[digest.Issue](t, h, "get_issue", map[string]any{"key": "42"})
		assert.Equal(t, sentIssue, out)
	})

	t.Run("By slug", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.Issues.EXPECT().FindBySlug(gomock.Any(), "2026-11-18").Return(sentIssue, nil)

		out := decode[digest.Issue](t, h, "get_issue", map[string]any{"key": " 2026-11-18 "})
		assert.Equal(t, int64(42), out.ID)
		require.Len(t, out.Items, 1)
		assert.Equal(t, "Go 1.27 freeze", out.Items[0].Title)
	})

	t.Run("Latest", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.Issues.EXPECT().Latest(gomock.Any(), 1).Return([]digest.Issue{sentIssue}, nil)

		out := decode[digest.Issue](t, h, "get_issue", map[string]any{"key": "LATEST"})
		assert.Equal(t, int64(42), out.ID)
	})

	t.Run("Latest with no sent issues", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.Issues.EXPECT().Latest(gomock.Any(), 1).Return(nil, nil)

		assert.Contains(t, toolError(t, h, "get_issue", map[string]any{"key": "latest"}), "no issues have been sent")
	})

	t.Run("Draft re-fetches to include items", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		draft := sentIssue
		draft.ID, draft.Status = 43, digest.IssueStatusDraft

		h.Issues.EXPECT().List(gomock.Any(), gomock.Cond(func(o digest.IssueListOptions) bool {
			return o.Status != nil && *o.Status == digest.IssueStatusDraft && o.PerPage == 1
		})).Return([]digest.Issue{{ID: 43}}, nil)
		h.Issues.EXPECT().Find(gomock.Any(), int64(43)).Return(draft, nil)

		out := decode[digest.Issue](t, h, "get_issue", map[string]any{"key": "draft"})
		assert.Equal(t, digest.IssueStatusDraft, out.Status)
		assert.Len(t, out.Items, 1)
	})

	t.Run("No draft", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.Issues.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)

		assert.Contains(t, toolError(t, h, "get_issue", map[string]any{"key": "draft"}), "no draft issue")
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.Issues.EXPECT().FindBySlug(gomock.Any(), "2020-01-01").Return(digest.Issue{}, store.ErrNotFound)

		assert.Contains(t, toolError(t, h, "get_issue", map[string]any{"key": "2020-01-01"}), `issue "2020-01-01" not found`)
	})

	t.Run("Blank key", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		assert.Contains(t, toolError(t, h, "get_issue", map[string]any{"key": "  "}), "key is required")
	})

	t.Run("Zero ID falls back to slug lookup", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.Issues.EXPECT().FindBySlug(gomock.Any(), "0").Return(digest.Issue{}, store.ErrNotFound)

		toolError(t, h, "get_issue", map[string]any{"key": "0"})
	})
}

func TestGetIssueCandidates(t *testing.T) {
	t.Parallel()

	h := setup(t)
	h.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(sentIssue, nil)

	wantStart, wantEnd := digest.BuildWindow(sentAt)
	candidate := news.Item{ID: 9, Source: news.SourceReddit, Title: "Left out", URL: "https://example.com", Published: wantStart.Add(time.Hour)}
	h.Items.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, o news.ItemListOptions) ([]news.Item, error) {
		require.NotNil(t, o.From)
		require.NotNil(t, o.To)
		assert.Equal(t, wantStart, *o.From)
		assert.Equal(t, wantEnd, *o.To)
		require.NotNil(t, o.InDigest)
		assert.False(t, *o.InDigest)
		assert.Equal(t, news.ItemSortTop, o.Sort)
		assert.Equal(t, int64(5), o.PerPage)
		return []news.Item{candidate}, nil
	})

	out := decode[ItemsOutput](t, h, "get_issue_candidates", map[string]any{"key": "42", "limit": 5})
	require.Len(t, out.Items, 1)
	assert.Equal(t, "Left out", out.Items[0].Title)
}
