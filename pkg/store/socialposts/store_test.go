// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package socialposts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtest"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/socialposts"
)

func stringPtr(s string) *string { return &s }

func TestSocialPosts_Store(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	is := issues.New(db)
	issue, err := is.Create(ctx, digest.Issue{
		Slug:    "2026-05-20",
		Subject: "GoDaily - May 20, 2026",
		Status:  digest.IssueStatusSent,
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
			got, err := s.Create(ctx, social.Post{
				IssueID:  &issue.ID,
				Platform: "bluesky",
				Text:     "Go 1.30 released — generics finally land in the standard library",
				PostURL:  "https://bsky.app/profile/godaily.bsky.social/post/abc123",
				PostedAt: when,
			})
			require.NoError(t, err)
			assert.NotZero(t, got.ID)
			require.NotNil(t, got.IssueID)
			assert.Equal(t, issue.ID, *got.IssueID)
			assert.Equal(t, "bluesky", got.Platform)
			assert.Equal(t, social.PostKindFeatured, got.Kind, "Kind defaults to featured when empty")
			assert.Equal(t, "https://bsky.app/profile/godaily.bsky.social/post/abc123", got.PostURL)
			assert.True(t, got.PostedAt.Equal(when))
		}

		t.Log("Zero PostedAt defaults to now")
		{
			before := time.Now().UTC().Add(-time.Second)
			got, err := s.Create(ctx, social.Post{
				IssueID:  &issue.ID,
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

	t.Run("HasPostedBySubject and HasPostedKindSince", func(t *testing.T) {
		t.Log("Subject lookup misses before insert")
		{
			got, err := s.HasPostedBySubject(ctx, "spotlight:ardanlabs", "bluesky")
			require.NoError(t, err)
			assert.False(t, got)
		}

		t.Log("Create a rotation post and re-check")
		{
			when := time.Date(2026, time.May, 20, 15, 0, 0, 0, time.UTC)
			_, err := s.Create(ctx, social.Post{
				Kind:     social.PostKindSpotlight,
				Subject:  "spotlight:ardanlabs",
				Platform: "bluesky",
				Text:     "shout out to ardanlabs",
				PostedAt: when,
			})
			require.NoError(t, err)

			got, err := s.HasPostedBySubject(ctx, "spotlight:ardanlabs", "bluesky")
			require.NoError(t, err)
			assert.True(t, got, "should match after insert")

			gotOther, err := s.HasPostedBySubject(ctx, "spotlight:ardanlabs", "linkedin")
			require.NoError(t, err)
			assert.False(t, gotOther, "different platform must not match")

			gotSince, err := s.HasPostedKindSince(ctx, social.PostKindSpotlight, "bluesky", when.Add(-time.Hour))
			require.NoError(t, err)
			assert.True(t, gotSince)

			gotFuture, err := s.HasPostedKindSince(ctx, social.PostKindSpotlight, "bluesky", when.Add(time.Hour))
			require.NoError(t, err)
			assert.False(t, gotFuture, "since after the post must miss")
		}
	})

	t.Run("List by issue returns inserted rows", func(t *testing.T) {
		got, err := s.List(ctx, social.PostListOptions{IssueID: &issue.ID})
		require.NoError(t, err)
		require.Len(t, got, 2)
		platforms := []string{got[0].Platform, got[1].Platform}
		assert.ElementsMatch(t, []string{"bluesky", "linkedin"}, platforms)
	})

	t.Run("Find returns inserted row and ErrNotFound for missing id", func(t *testing.T) {
		t.Log("Existing row")
		{
			rows, err := s.List(ctx, social.PostListOptions{IssueID: &issue.ID})
			require.NoError(t, err)
			require.NotEmpty(t, rows)

			got, err := s.Find(ctx, rows[0].ID)
			require.NoError(t, err)
			assert.Equal(t, rows[0].ID, got.ID)
			assert.Equal(t, rows[0].Platform, got.Platform)
		}

		t.Log("Missing row")
		{
			_, err := s.Find(ctx, 99999)
			require.Error(t, err)
		}
	})

	t.Run("Update transitions draft to published with COALESCE preservation", func(t *testing.T) {
		when := time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC)
		created, err := s.Create(ctx, social.Post{
			IssueID:  &issue.ID,
			Platform: "mastodon",
			Text:     "draft body",
			Status:   social.PostStatusDraft,
			PostedAt: when,
		})
		require.NoError(t, err)

		t.Log("Edit text alone preserves status")
		{
			newText := "edited body"
			got, err := s.Update(ctx, created.ID, social.PostUpdate{Text: &newText})
			require.NoError(t, err)
			assert.Equal(t, "edited body", got.Text)
			assert.Equal(t, social.PostStatusDraft, got.Status, "text-only edit must keep status=draft")
		}

		t.Log("Publish: set status + published_at + post_url")
		{
			pub := social.PostStatusPublished
			now := time.Now().UTC()
			url := "https://mastodon.social/x/42"
			got, err := s.Update(ctx, created.ID, social.PostUpdate{
				Status:      &pub,
				PublishedAt: &now,
				PostURL:     &url,
			})
			require.NoError(t, err)
			assert.Equal(t, social.PostStatusPublished, got.Status)
			require.NotNil(t, got.PublishedAt)
			assert.Equal(t, url, got.PostURL)
		}

		t.Log("Cancel: status-only update preserves text and post_url")
		{
			cancel := social.PostStatusCancelled
			got, err := s.Update(ctx, created.ID, social.PostUpdate{Status: &cancel})
			require.NoError(t, err)
			assert.Equal(t, social.PostStatusCancelled, got.Status)
			assert.Equal(t, "edited body", got.Text)
			assert.Equal(t, "https://mastodon.social/x/42", got.PostURL)
		}
	})

	t.Run("HasPostedOrCancelledBySubject counts cancelled rows", func(t *testing.T) {
		// Create a draft, then cancel it. Subject lookup must report true.
		when := time.Date(2026, time.May, 22, 12, 0, 0, 0, time.UTC)
		created, err := s.Create(ctx, social.Post{
			Kind:     social.PostKindSpotlight,
			Subject:  "spotlight:gobyexample",
			Platform: "bluesky",
			Text:     "draft",
			Status:   social.PostStatusDraft,
			PostedAt: when,
		})
		require.NoError(t, err)

		cancel := social.PostStatusCancelled
		_, err = s.Update(ctx, created.ID, social.PostUpdate{Status: &cancel})
		require.NoError(t, err)

		gotPublished, err := s.HasPostedBySubject(ctx, "spotlight:gobyexample", "bluesky")
		require.NoError(t, err)
		assert.False(t, gotPublished, "cancelled rows must not count as published")

		gotEither, err := s.HasPostedOrCancelledBySubject(ctx, "spotlight:gobyexample", "bluesky")
		require.NoError(t, err)
		assert.True(t, gotEither, "cancelled rows must count when checking the combined predicate")
	})

	t.Run("DeleteDraftsByKind only removes status=draft rows with null issue_id", func(t *testing.T) {
		// Two rotation drafts of the same kind and one published row.
		when := time.Date(2026, time.May, 23, 12, 0, 0, 0, time.UTC)
		_, err := s.Create(ctx, social.Post{
			Kind:     social.PostKindCommunity,
			Subject:  "community:foo:2026",
			Platform: "bluesky",
			Text:     "d1",
			Status:   social.PostStatusDraft,
			PostedAt: when,
		})
		require.NoError(t, err)
		_, err = s.Create(ctx, social.Post{
			Kind:     social.PostKindCommunity,
			Subject:  "community:foo:2026",
			Platform: "linkedin",
			Text:     "d2",
			Status:   social.PostStatusDraft,
			PostedAt: when,
		})
		require.NoError(t, err)
		_, err = s.Create(ctx, social.Post{
			Kind:     social.PostKindCommunity,
			Subject:  "community:bar:2026",
			Platform: "bluesky",
			Text:     "published one",
			Status:   social.PostStatusPublished,
			PostedAt: when,
		})
		require.NoError(t, err)

		require.NoError(t, s.DeleteDraftsByKind(ctx, social.PostKindCommunity))

		draft := social.PostStatusDraft
		drafts, err := s.List(ctx, social.PostListOptions{Status: &draft})
		require.NoError(t, err)
		for _, d := range drafts {
			assert.NotEqual(t, social.PostKindCommunity, d.Kind, "community drafts must be gone")
		}

		// Published row of the same kind survives.
		published := social.PostStatusPublished
		pubRows, err := s.List(ctx, social.PostListOptions{Status: &published, Platform: stringPtr("bluesky")})
		require.NoError(t, err)
		var foundPublished bool
		for _, r := range pubRows {
			if r.Kind == social.PostKindCommunity {
				foundPublished = true
				break
			}
		}
		assert.True(t, foundPublished, "DeleteDraftsByKind must not touch non-draft rows")
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
			_, err := s.Create(ctx, social.Post{IssueID: &issue.ID, Platform: "x", Text: "y"})
			assert.Error(t, err)
		}

		t.Log("List")
		{
			_, err := s.List(ctx, social.PostListOptions{IssueID: &issue.ID})
			assert.Error(t, err)
		}
	})
}
