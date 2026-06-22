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

func TestItems_Store(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	is := issues.New(db)
	issue, err := is.Create(ctx, digest.Issue{
		Slug:    "2026-04-28",
		Subject: "GoDaily - April 28, 2026",
		Status:  digest.IssueStatusSent,
		SentAt:  time.Date(2026, time.April, 28, 8, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	s := items.New(db)

	published := time.Date(2026, time.April, 28, 7, 0, 0, 0, time.UTC)
	itemWithAuthor := news.Item{
		Source:    news.SourceGoBlog,
		Title:     "Go 1.30 released",
		URL:       "https://go.dev/blog/go1.30",
		Snippet:   "Headline post",
		Score:     0.9,
		Author:    &news.Author{Name: "Ainsley", Username: "ainsleyclark"},
		Published: published,
	}
	itemNoAuthor := news.Item{
		Source:    news.SourceHN,
		Title:     "Trending discussion",
		URL:       "https://news.ycombinator.com/item?id=1",
		Score:     0.5,
		Published: published,
	}

	var firstID int64

	t.Run("Create with issue", func(t *testing.T) {
		t.Log("Happy path with author")
		{
			got, err := s.Create(ctx, &issue.ID, 0, itemWithAuthor)
			require.NoError(t, err)
			assert.NotZero(t, got.ID)
			assert.Equal(t, news.SourceGoBlog, got.Source)
			assert.Equal(t, itemWithAuthor.Title, got.Title)
			assert.Equal(t, itemWithAuthor.URL, got.URL)
			assert.Equal(t, itemWithAuthor.Snippet, got.Snippet)
			assert.InDelta(t, itemWithAuthor.Score, got.Score, 1e-9)
			require.NotNil(t, got.Author)
			assert.Equal(t, "Ainsley", got.Author.Name)
			assert.Equal(t, "ainsleyclark", got.Author.Username)
			firstID = got.ID
		}

		t.Log("Without author returns nil Author")
		{
			got, err := s.Create(ctx, &issue.ID, 1, itemNoAuthor)
			require.NoError(t, err)
			assert.NotZero(t, got.ID)
			assert.Nil(t, got.Author)
		}
	})

	t.Run("Create without issue", func(t *testing.T) {
		unlinked := news.Item{
			Source:    news.SourceGoBlog,
			Title:     "Go proposal accepted",
			URL:       "https://go.dev/blog/proposal-accepted",
			Snippet:   "An unlinked item",
			Score:     0.7,
			Published: published,
		}
		got, err := s.Create(ctx, nil, 2, unlinked)
		require.NoError(t, err)
		assert.NotZero(t, got.ID)
		assert.Equal(t, unlinked.Title, got.Title)
	})

	t.Run("Find", func(t *testing.T) {
		t.Log("Happy path")
		{
			got, err := s.Find(ctx, firstID)
			require.NoError(t, err)
			assert.Equal(t, firstID, got.ID)
			require.NotNil(t, got.Author)
			assert.Equal(t, "Ainsley", got.Author.Name)
		}

		t.Log("Not found")
		{
			_, err := s.Find(ctx, 999_999)
			require.Error(t, err)
			assert.Equal(t, store.ErrNotFound, err)
		}
	})

	t.Run("List by issue", func(t *testing.T) {
		t.Log("Returns rows ordered by position")
		{
			got, err := s.List(ctx, news.ItemListOptions{IssueID: &issue.ID})
			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.Equal(t, itemWithAuthor.Title, got[0].Title)
			assert.Equal(t, itemNoAuthor.Title, got[1].Title)
		}

		t.Log("Empty for unknown issue")
		{
			unknown := int64(999_999)
			got, err := s.List(ctx, news.ItemListOptions{IssueID: &unknown})
			require.NoError(t, err)
			assert.Empty(t, got)
		}
	})

	t.Run("List by date range", func(t *testing.T) {
		from := published.Add(-time.Hour)
		to := published.Add(time.Hour * 2)
		got, err := s.List(ctx, news.ItemListOptions{From: &from, To: &to})
		require.NoError(t, err)
		assert.NotEmpty(t, got)
	})

	t.Run("List with no filter returns all", func(t *testing.T) {
		got, err := s.List(ctx, news.ItemListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, got)
	})

	t.Run("Create upserts issue_id on conflict", func(t *testing.T) {
		// Regression: collect inserts items with issue_id=nil; build then re-inserts
		// with a real issue_id. The ON CONFLICT clause must update issue_id so the
		// social service can find items by issue.
		unlinked := news.Item{
			Source:    news.SourceHN,
			Title:     "Upsert test item",
			URL:       "https://news.ycombinator.com/item?id=upsert",
			Score:     0.8,
			Published: published,
		}
		// First insert: no issue (simulates collect).
		got, err := s.Create(ctx, nil, 10, unlinked)
		require.NoError(t, err)
		assert.Nil(t, got.Author) // sanity

		// Second insert: same URL+tag, now with an issue (simulates build).
		got2, err := s.Create(ctx, &issue.ID, 10, unlinked)
		require.NoError(t, err)
		assert.Equal(t, got.ID, got2.ID, "upsert must not create a new row")

		// List by issue must now find the item.
		byIssue, err := s.List(ctx, news.ItemListOptions{IssueID: &issue.ID})
		require.NoError(t, err)
		found := false
		for _, it := range byIssue {
			if it.ID == got.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "item must be linked to the issue after upsert")
	})

	t.Run("Create freezes published on conflict", func(t *testing.T) {
		// Guarantee for accepted proposals: their Published is sourced from
		// updated_at, which bumps on any later activity (a comment re-surfaces
		// the issue in a later collection window). The upsert must preserve the
		// first-seen published so the item lands in exactly one digest and never
		// re-enters a later window.
		first := news.Item{
			Source:    news.SourceGitHub,
			Tag:       news.TagProposalAccepted,
			Title:     "Accepted proposal",
			URL:       "https://github.com/golang/go/issues/freeze",
			Score:     0.9,
			Published: published,
		}
		created, err := s.Create(ctx, nil, 20, first)
		require.NoError(t, err)

		// Re-collected days later with a bumped published (same URL+tag).
		bumped := first
		bumped.Published = published.Add(72 * time.Hour)
		again, err := s.Create(ctx, nil, 20, bumped)
		require.NoError(t, err)

		assert.Equal(t, created.ID, again.ID, "upsert must not create a new row")
		assert.Equal(t, published, again.Published, "published must stay frozen at the first-seen value")
	})

	t.Run("FindByURLInIssue", func(t *testing.T) {
		t.Log("Matches the canonical URL")
		{
			id, ok, err := s.FindByURLInIssue(ctx, issue.ID, itemWithAuthor.URL)
			require.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, firstID, id)
		}

		t.Log("Matches the original URL")
		{
			withOriginal := news.Item{
				Source:      news.SourceHN,
				Title:       "Linked discussion",
				URL:         "https://example.com/canonical",
				OriginalURL: "https://news.ycombinator.com/item?id=orig",
				Score:       0.6,
				Published:   published,
			}
			created, err := s.Create(ctx, &issue.ID, 20, withOriginal)
			require.NoError(t, err)

			id, ok, err := s.FindByURLInIssue(ctx, issue.ID, withOriginal.OriginalURL)
			require.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, created.ID, id)
		}

		t.Log("A miss returns ok=false without an error")
		{
			id, ok, err := s.FindByURLInIssue(ctx, issue.ID, "https://nowhere.example.com")
			require.NoError(t, err)
			assert.False(t, ok)
			assert.Zero(t, id)
		}

		t.Log("Lookup is scoped to the issue")
		{
			_, ok, err := s.FindByURLInIssue(ctx, 999_999, itemWithAuthor.URL)
			require.NoError(t, err)
			assert.False(t, ok)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		t.Log("Hard-deletes a single row")
		{
			created, err := s.Create(ctx, nil, 30, news.Item{
				Source:    news.SourceHN,
				Title:     "Off-topic item",
				URL:       "https://news.ycombinator.com/item?id=delete-me",
				Score:     0.1,
				Published: published,
			})
			require.NoError(t, err)

			require.NoError(t, s.Delete(ctx, created.ID))

			_, err = s.Find(ctx, created.ID)
			require.Error(t, err)
			assert.Equal(t, store.ErrNotFound, err)
		}

		t.Log("Missing id returns ErrNotFound")
		{
			err := s.Delete(ctx, 999_999)
			require.Error(t, err)
			assert.Equal(t, store.ErrNotFound, err)
		}
	})

	t.Run("DeleteByIssue", func(t *testing.T) {
		require.NoError(t, s.DeleteByIssue(ctx, issue.ID))
		got, err := s.List(ctx, news.ItemListOptions{IssueID: &issue.ID})
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	// MUST be last: closing the DB makes every subsequent query fail.
	t.Run("Query Error On Closed DB", func(t *testing.T) {
		require.NoError(t, db.Close())

		t.Log("Find")
		{
			_, err := s.Find(ctx, 1)
			assert.Error(t, err)
			assert.NotErrorIs(t, err, store.ErrNotFound)
		}

		t.Log("List by issue")
		{
			id := int64(1)
			_, err := s.List(ctx, news.ItemListOptions{IssueID: &id})
			assert.Error(t, err)
		}

		t.Log("Count")
		{
			_, err := s.Count(ctx)
			assert.Error(t, err)
		}

		t.Log("CountMatching")
		{
			_, err := s.CountMatching(ctx, news.ItemListOptions{})
			assert.Error(t, err)
		}

		t.Log("SourceCounts")
		{
			_, err := s.SourceCounts(ctx)
			assert.Error(t, err)
		}

		t.Log("TagCounts")
		{
			_, err := s.TagCounts(ctx)
			assert.Error(t, err)
		}

		t.Log("Create")
		{
			_, err := s.Create(ctx, nil, 0, news.Item{Source: news.SourceHN, Title: "x", URL: "x"})
			assert.Error(t, err)
		}

		t.Log("DeleteByIssue")
		{
			assert.Error(t, s.DeleteByIssue(ctx, 1))
		}

		t.Log("Delete")
		{
			err := s.Delete(ctx, 1)
			assert.Error(t, err)
			assert.NotErrorIs(t, err, store.ErrNotFound)
		}

		t.Log("FindByURLInIssue")
		{
			_, ok, err := s.FindByURLInIssue(ctx, 1, "https://go.dev")
			assert.Error(t, err)
			assert.False(t, ok)
		}
	})
}

func TestItems_Browse(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	is := issues.New(db)
	issue, err := is.Create(ctx, digest.Issue{
		Slug:    "2026-05-01",
		Subject: "GoDaily - May 1, 2026",
		Status:  digest.IssueStatusSent,
		SentAt:  time.Date(2026, time.May, 1, 8, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	s := items.New(db)

	base := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)

	// Linked items (in digest).
	linked := []news.Item{
		{Source: news.SourceGoBlog, Tag: news.TagRelease, Title: "Go 1.30 released", URL: "https://go.dev/1.30", Score: 0.9, Snippet: "release notes", Published: base.Add(3 * time.Hour)},
		{Source: news.SourceHN, Tag: news.TagDiscussion, Title: "HN thread on generics", URL: "https://news.ycombinator.com/item?id=10", Score: 0.5, Snippet: "generics chatter", Published: base.Add(2 * time.Hour)},
		{Source: news.SourceGoBlog, Tag: news.TagArticle, Title: "Profiling Go", URL: "https://go.dev/profiling", Score: 0.7, Snippet: "pprof tips", Published: base.Add(1 * time.Hour)},
	}
	for i, it := range linked {
		_, err := s.Create(ctx, &issue.ID, i, it)
		require.NoError(t, err)
	}

	// Unlinked items (not in digest).
	unlinked := []news.Item{
		{Source: news.SourceHN, Tag: news.TagTrending, Title: "Rust vs Go", URL: "https://news.ycombinator.com/item?id=20", Score: 0.4, Snippet: "language war", Published: base.Add(4 * time.Hour)},
		{Source: news.SourceGoBlog, Tag: news.TagTutorial, Title: "Context tutorial", URL: "https://go.dev/context", Score: 0.8, Snippet: "context.Context guide", Published: base.Add(5 * time.Hour)},
	}
	for i, it := range unlinked {
		_, err := s.Create(ctx, nil, i+100, it)
		require.NoError(t, err)
	}

	t.Run("InDigest filter", func(t *testing.T) {
		yes := true
		no := false

		gotYes, err := s.List(ctx, news.ItemListOptions{InDigest: &yes})
		require.NoError(t, err)
		assert.Len(t, gotYes, 3)
		for _, it := range gotYes {
			assert.True(t, it.InDigest, "expected InDigest=true for %q", it.Title)
		}

		gotNo, err := s.List(ctx, news.ItemListOptions{InDigest: &no})
		require.NoError(t, err)
		assert.Len(t, gotNo, 2)
		for _, it := range gotNo {
			assert.False(t, it.InDigest, "expected InDigest=false for %q", it.Title)
		}

		gotAll, err := s.List(ctx, news.ItemListOptions{})
		require.NoError(t, err)
		assert.Len(t, gotAll, 5)
	})

	t.Run("Source filter", func(t *testing.T) {
		got, err := s.List(ctx, news.ItemListOptions{Sources: []news.Source{news.SourceHN}})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		for _, it := range got {
			assert.Equal(t, news.SourceHN, it.Source)
		}

		got, err = s.List(ctx, news.ItemListOptions{Sources: []news.Source{news.SourceGoBlog, news.SourceHN}})
		require.NoError(t, err)
		assert.Len(t, got, 5)
	})

	t.Run("Tag filter", func(t *testing.T) {
		got, err := s.List(ctx, news.ItemListOptions{Tags: []news.Tag{news.TagRelease, news.TagTutorial}})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("Search filter", func(t *testing.T) {
		got, err := s.List(ctx, news.ItemListOptions{Search: "generics"})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "HN thread on generics", got[0].Title)

		// Matches summary as well.
		got, err = s.List(ctx, news.ItemListOptions{Search: "pprof"})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "Profiling Go", got[0].Title)
	})

	t.Run("Sort New orders by published DESC", func(t *testing.T) {
		got, err := s.List(ctx, news.ItemListOptions{Sort: news.ItemSortNew})
		require.NoError(t, err)
		require.Len(t, got, 5)
		for i := 1; i < len(got); i++ {
			assert.False(t, got[i].Published.After(got[i-1].Published),
				"expected published DESC at index %d", i)
		}
	})

	t.Run("Sort Top orders by score DESC", func(t *testing.T) {
		got, err := s.List(ctx, news.ItemListOptions{Sort: news.ItemSortTop})
		require.NoError(t, err)
		require.Len(t, got, 5)
		for i := 1; i < len(got); i++ {
			assert.LessOrEqual(t, got[i].Score, got[i-1].Score,
				"expected score DESC at index %d", i)
		}
	})

	t.Run("Sort Hot returns all rows", func(t *testing.T) {
		got, err := s.List(ctx, news.ItemListOptions{Sort: news.ItemSortHot})
		require.NoError(t, err)
		assert.Len(t, got, 5)
	})

	t.Run("Pagination", func(t *testing.T) {
		p1, err := s.List(ctx, news.ItemListOptions{Sort: news.ItemSortNew, Page: 1, PerPage: 2})
		require.NoError(t, err)
		assert.Len(t, p1, 2)

		p2, err := s.List(ctx, news.ItemListOptions{Sort: news.ItemSortNew, Page: 2, PerPage: 2})
		require.NoError(t, err)
		assert.Len(t, p2, 2)
		assert.NotEqual(t, p1[0].ID, p2[0].ID, "second page should not repeat first page rows")

		p3, err := s.List(ctx, news.ItemListOptions{Sort: news.ItemSortNew, Page: 3, PerPage: 2})
		require.NoError(t, err)
		assert.Len(t, p3, 1)

		// Page past the end is empty.
		p4, err := s.List(ctx, news.ItemListOptions{Sort: news.ItemSortNew, Page: 99, PerPage: 2})
		require.NoError(t, err)
		assert.Empty(t, p4)
	})

	t.Run("Count", func(t *testing.T) {
		got, err := s.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(5), got)
	})

	t.Run("SourceCounts", func(t *testing.T) {
		got, err := s.SourceCounts(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, got)
		total := int64(0)
		for _, sc := range got {
			total += sc.Count
		}
		assert.Equal(t, int64(5), total)
		// Ordered by count DESC.
		for i := 1; i < len(got); i++ {
			assert.LessOrEqual(t, got[i].Count, got[i-1].Count)
		}
	})

	t.Run("TagCounts", func(t *testing.T) {
		got, err := s.TagCounts(ctx)
		require.NoError(t, err)
		require.Len(t, got, 5) // five distinct tags above
		for i := 1; i < len(got); i++ {
			assert.LessOrEqual(t, got[i].Count, got[i-1].Count)
		}
	})

	t.Run("CountMatching", func(t *testing.T) {
		t.Log("No filters counts every row")
		{
			got, err := s.CountMatching(ctx, news.ItemListOptions{})
			require.NoError(t, err)
			assert.Equal(t, int64(5), got)
		}

		t.Log("Ignores pagination")
		{
			got, err := s.CountMatching(ctx, news.ItemListOptions{Page: 2, PerPage: 1})
			require.NoError(t, err)
			assert.Equal(t, int64(5), got)
		}

		t.Log("Applies the same WHERE as List")
		{
			yes := true
			got, err := s.CountMatching(ctx, news.ItemListOptions{InDigest: &yes})
			require.NoError(t, err)
			assert.Equal(t, int64(3), got)

			got, err = s.CountMatching(ctx, news.ItemListOptions{Sources: []news.Source{news.SourceHN}})
			require.NoError(t, err)
			assert.Equal(t, int64(2), got)

			got, err = s.CountMatching(ctx, news.ItemListOptions{Search: "generics"})
			require.NoError(t, err)
			assert.Equal(t, int64(1), got)
		}
	})

	t.Run("Combined filters", func(t *testing.T) {
		yes := true
		got, err := s.List(ctx, news.ItemListOptions{
			Sources:  []news.Source{news.SourceGoBlog},
			InDigest: &yes,
			Sort:     news.ItemSortTop,
		})
		require.NoError(t, err)
		require.Len(t, got, 2)
		for _, it := range got {
			assert.Equal(t, news.SourceGoBlog, it.Source)
			assert.True(t, it.InDigest)
		}
		assert.GreaterOrEqual(t, got[0].Score, got[1].Score)
	})
}
