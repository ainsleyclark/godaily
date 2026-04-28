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

package items_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/ainsleyclark/godaily/internal/store/internal/dbtest"
	"github.com/ainsleyclark/godaily/internal/store/issues"
	"github.com/ainsleyclark/godaily/internal/store/items"
)

func TestItems_Store(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	is := issues.New(db)
	issue, err := is.Create(ctx, news.Issue{
		Slug:     "2026-04-28",
		Subject:  "GoDaily - April 28, 2026",
		Status:   news.IssueStatusSent,
		HtmlBody: "<p>hi</p>",
		TextBody: "hi",
		SentAt:   time.Date(2026, time.April, 28, 8, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	s := items.New(db)

	itemWithAuthor := news.Item{
		Source:    news.SourceGoBlog,
		Title:     "Go 1.30 released",
		URL:       "https://go.dev/blog/go1.30",
		Snippet:   "Headline post",
		Score:     0.9,
		Author:    &news.Author{Name: "Ainsley", Username: "ainsleyclark"},
		Published: time.Date(2026, time.April, 28, 7, 0, 0, 0, time.UTC),
	}
	itemNoAuthor := news.Item{
		Source: news.SourceHN,
		Title:  "Trending discussion",
		URL:    "https://news.ycombinator.com/item?id=1",
		Score:  0.5,
	}

	var firstID int64

	t.Run("Create", func(t *testing.T) {
		t.Log("Happy path with author")
		{
			got, err := s.Create(ctx, issue.ID, 0, itemWithAuthor)
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
			got, err := s.Create(ctx, issue.ID, 1, itemNoAuthor)
			require.NoError(t, err)
			assert.NotZero(t, got.ID)
			assert.Nil(t, got.Author)
		}
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

	t.Run("ListByIssue", func(t *testing.T) {
		t.Log("Returns rows ordered by position")
		{
			got, err := s.ListByIssue(ctx, issue.ID)
			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.Equal(t, itemWithAuthor.Title, got[0].Title)
			assert.Equal(t, itemNoAuthor.Title, got[1].Title)
		}

		t.Log("Empty for unknown issue")
		{
			got, err := s.ListByIssue(ctx, 999_999)
			require.NoError(t, err)
			assert.Empty(t, got)
		}
	})

	t.Run("DeleteByIssue", func(t *testing.T) {
		require.NoError(t, s.DeleteByIssue(ctx, issue.ID))
		got, err := s.ListByIssue(ctx, issue.ID)
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

		t.Log("ListByIssue")
		{
			_, err := s.ListByIssue(ctx, 1)
			assert.Error(t, err)
		}

		t.Log("Create")
		{
			_, err := s.Create(ctx, 1, 0, news.Item{Source: news.SourceHN, Title: "x", URL: "x"})
			assert.Error(t, err)
		}

		t.Log("DeleteByIssue")
		{
			assert.Error(t, s.DeleteByIssue(ctx, 1))
		}
	})
}
