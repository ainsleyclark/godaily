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

package socialposts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtest"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/socialposts"
)

func TestSocialPosts_Store(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	is := issues.New(db)
	issue, err := is.Create(ctx, news.Issue{
		Slug:    "2026-05-20",
		Subject: "GoDaily - May 20, 2026",
		Status:  news.IssueStatusSent,
		SentAt:  time.Date(2026, time.May, 20, 8, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	s := socialposts.New(db)

	t.Run("HasPosted false before create", func(t *testing.T) {
		got, err := s.HasPosted(ctx, issue.ID, "bluesky")
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("Create persists row and sets PostedAt", func(t *testing.T) {
		t.Log("With explicit PostedAt")
		{
			when := time.Date(2026, time.May, 20, 11, 30, 0, 0, time.UTC)
			got, err := s.Create(ctx, news.SocialPost{
				IssueID:  issue.ID,
				Platform: "bluesky",
				Text:     "Go 1.30 released — generics finally land in the standard library",
				PostURL:  "https://bsky.app/profile/godaily.bsky.social/post/abc123",
				PostedAt: when,
			})
			require.NoError(t, err)
			assert.NotZero(t, got.ID)
			assert.Equal(t, issue.ID, got.IssueID)
			assert.Equal(t, "bluesky", got.Platform)
			assert.Equal(t, "https://bsky.app/profile/godaily.bsky.social/post/abc123", got.PostURL)
			assert.True(t, got.PostedAt.Equal(when))
		}

		t.Log("Zero PostedAt defaults to now")
		{
			before := time.Now().UTC().Add(-time.Second)
			got, err := s.Create(ctx, news.SocialPost{
				IssueID:  issue.ID,
				Platform: "linkedin",
				Text:     "professional rendering of the same item",
			})
			require.NoError(t, err)
			assert.False(t, got.PostedAt.IsZero())
			assert.True(t, got.PostedAt.After(before), "expected default PostedAt to be ~now")
			assert.Empty(t, got.PostURL)
		}
	})

	t.Run("HasPosted true after create", func(t *testing.T) {
		t.Log("Match")
		{
			got, err := s.HasPosted(ctx, issue.ID, "bluesky")
			require.NoError(t, err)
			assert.True(t, got)
		}

		t.Log("Different platform same issue is unaffected")
		{
			got, err := s.HasPosted(ctx, issue.ID, "mastodon")
			require.NoError(t, err)
			assert.False(t, got)
		}
	})

	t.Run("Unique constraint prevents duplicates", func(t *testing.T) {
		_, err := s.Create(ctx, news.SocialPost{
			IssueID:  issue.ID,
			Platform: "bluesky",
			Text:     "duplicate",
		})
		assert.Error(t, err)
	})

	t.Run("List by issue returns inserted rows", func(t *testing.T) {
		got, err := s.List(ctx, news.SocialPostListOptions{IssueID: &issue.ID})
		require.NoError(t, err)
		require.Len(t, got, 2)
		platforms := []string{got[0].Platform, got[1].Platform}
		assert.ElementsMatch(t, []string{"bluesky", "linkedin"}, platforms)
	})

	// MUST be last: closing the DB makes every subsequent query fail.
	t.Run("Query Error On Closed DB", func(t *testing.T) {
		require.NoError(t, db.Close())

		t.Log("HasPosted")
		{
			_, err := s.HasPosted(ctx, issue.ID, "bluesky")
			assert.Error(t, err)
		}

		t.Log("Create")
		{
			_, err := s.Create(ctx, news.SocialPost{IssueID: issue.ID, Platform: "x", Text: "y"})
			assert.Error(t, err)
		}

		t.Log("List")
		{
			_, err := s.List(ctx, news.SocialPostListOptions{IssueID: &issue.ID})
			assert.Error(t, err)
		}
	})
}
