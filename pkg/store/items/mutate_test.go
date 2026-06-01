// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package items_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtest"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/items"
)

func TestUnlinkFromIssue(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	is := issues.New(db)
	s := items.New(db)
	published := time.Date(2026, time.May, 1, 8, 0, 0, 0, time.UTC)

	mkIssue := func(t *testing.T, slug string, status digest.IssueStatus) digest.Issue {
		t.Helper()
		iss, err := is.Create(ctx, digest.Issue{
			Slug: slug, Subject: "S", Status: status, SentAt: published,
		})
		require.NoError(t, err)
		return iss
	}
	mkItem := func(t *testing.T, issueID int64, pos int, url string) int64 {
		t.Helper()
		got, err := s.Create(ctx, &issueID, pos, news.Item{
			Source: news.SourceGoBlog, Tag: news.TagArticle,
			Title: "x", URL: url, Published: published,
		})
		require.NoError(t, err)
		return got.ID
	}

	t.Run("Unlinks draft item and preserves row", func(t *testing.T) {
		issue := mkIssue(t, "2026-05-02", digest.IssueStatusDraft)
		id := mkItem(t, issue.ID, 0, "https://example.com/u1")

		require.NoError(t, s.UnlinkFromIssue(ctx, issue.ID, id))

		got, err := s.Find(ctx, id)
		require.NoError(t, err)
		assert.False(t, got.InDigest, "row should remain but be unlinked")
		assert.EqualValues(t, 0, got.Position)
	})

	t.Run("Non-draft returns ErrIssueNotDraft", func(t *testing.T) {
		issue := mkIssue(t, "2026-05-03", digest.IssueStatusSent)
		id := mkItem(t, issue.ID, 0, "https://example.com/u2")

		err := s.UnlinkFromIssue(ctx, issue.ID, id)
		assert.ErrorIs(t, err, digest.ErrIssueNotDraft)
	})

	t.Run("Missing issue returns ErrNotFound", func(t *testing.T) {
		err := s.UnlinkFromIssue(ctx, 999_999, 1)
		assert.ErrorIs(t, err, store.ErrNotFound)
	})

	t.Run("Item not in this issue returns ErrNotFound", func(t *testing.T) {
		issue := mkIssue(t, "2026-05-04", digest.IssueStatusDraft)
		// Item not in this issue (id from another issue).
		other := mkIssue(t, "2026-05-04-other", digest.IssueStatusDraft)
		otherID := mkItem(t, other.ID, 0, "https://example.com/u3")

		err := s.UnlinkFromIssue(ctx, issue.ID, otherID)
		assert.ErrorIs(t, err, store.ErrNotFound)
	})
}

func TestReorderInIssue(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	is := issues.New(db)
	s := items.New(db)
	published := time.Date(2026, time.June, 1, 8, 0, 0, 0, time.UTC)

	mkIssue := func(t *testing.T, slug string, status digest.IssueStatus) digest.Issue {
		t.Helper()
		iss, err := is.Create(ctx, digest.Issue{
			Slug: slug, Subject: "S", Status: status, SentAt: published,
		})
		require.NoError(t, err)
		return iss
	}
	mkItem := func(t *testing.T, issueID int64, pos int, url string) int64 {
		t.Helper()
		got, err := s.Create(ctx, &issueID, pos, news.Item{
			Source: news.SourceGoBlog, Tag: news.TagArticle,
			Title: url, URL: url, Published: published,
		})
		require.NoError(t, err)
		return got.ID
	}
	listIDs := func(t *testing.T, issueID int64) []int64 {
		t.Helper()
		got, err := s.List(ctx, news.ItemListOptions{IssueID: &issueID})
		require.NoError(t, err)
		ids := make([]int64, len(got))
		for i, it := range got {
			ids[i] = it.ID
		}
		return ids
	}

	t.Run("Rewrites positions in supplied order", func(t *testing.T) {
		issue := mkIssue(t, "2026-06-02", digest.IssueStatusDraft)
		a := mkItem(t, issue.ID, 0, "https://example.com/a")
		b := mkItem(t, issue.ID, 1, "https://example.com/b")
		c := mkItem(t, issue.ID, 2, "https://example.com/c")

		require.NoError(t, s.ReorderInIssue(ctx, issue.ID, []int64{c, a, b}))

		assert.Equal(t, []int64{c, a, b}, listIDs(t, issue.ID))
	})

	t.Run("Non-draft returns ErrIssueNotDraft", func(t *testing.T) {
		issue := mkIssue(t, "2026-06-03", digest.IssueStatusSent)
		a := mkItem(t, issue.ID, 0, "https://example.com/d")
		b := mkItem(t, issue.ID, 1, "https://example.com/e")

		err := s.ReorderInIssue(ctx, issue.ID, []int64{b, a})
		assert.ErrorIs(t, err, digest.ErrIssueNotDraft)
	})

	t.Run("Mismatched ids leave order untouched", func(t *testing.T) {
		issue := mkIssue(t, "2026-06-04", digest.IssueStatusDraft)
		a := mkItem(t, issue.ID, 0, "https://example.com/f")
		b := mkItem(t, issue.ID, 1, "https://example.com/g")

		// Missing one + spurious id.
		err := s.ReorderInIssue(ctx, issue.ID, []int64{a, 999_999})
		assert.ErrorIs(t, err, store.ErrNotFound)

		// Order preserved.
		assert.Equal(t, []int64{a, b}, listIDs(t, issue.ID))
	})

	t.Run("Missing issue returns ErrNotFound", func(t *testing.T) {
		err := s.ReorderInIssue(ctx, 999_999, []int64{1})
		assert.ErrorIs(t, err, store.ErrNotFound)
	})

	t.Run("Subset of ids is rejected", func(t *testing.T) {
		issue := mkIssue(t, "2026-06-05", digest.IssueStatusDraft)
		a := mkItem(t, issue.ID, 0, "https://example.com/h")
		_ = mkItem(t, issue.ID, 1, "https://example.com/i")

		err := s.ReorderInIssue(ctx, issue.ID, []int64{a})
		assert.ErrorIs(t, err, store.ErrNotFound)
	})
}
