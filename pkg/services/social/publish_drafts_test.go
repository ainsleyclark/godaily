// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_PublishDrafts(t *testing.T) {
	t.Parallel()

	t.Run("No Posters Skip", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		res, err := f.service().PublishDrafts(t.Context(), social.PostOptions{Date: time.Now()})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("No Drafts Returns Empty", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		f.posters = []platform.Poster{newMockPoster(f.ctrl, social.Bluesky)}

		f.posts.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().PublishDrafts(t.Context(), social.PostOptions{Date: date})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Dry Run Skips Poster", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		// poster.Post and posts.Update must NOT be called.
		f.posters = []platform.Poster{bluesky}

		f.posts.EXPECT().List(gomock.Any(), gomock.Any()).Return([]social.Post{
			{ID: 1, Platform: "bluesky", Text: "draft text", Status: social.PostStatusDraft},
		}, nil)

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().PublishDrafts(t.Context(), social.PostOptions{Date: date, DryRun: true})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Empty(t, res[0].PostURL)
	})

	t.Run("Poster Error Marks Errored", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		bluesky.EXPECT().Post(gomock.Any(), gomock.Any()).Return(platform.PostResponse{}, errors.New("API down"))
		f.posters = []platform.Poster{bluesky}

		f.posts.EXPECT().List(gomock.Any(), gomock.Any()).Return([]social.Post{
			{ID: 7, Platform: "bluesky", Text: "draft", Status: social.PostStatusDraft},
		}, nil)
		f.posts.EXPECT().
			Update(gomock.Any(), int64(7), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, u social.PostUpdate) (social.Post, error) {
				require.NotNil(t, u.Status)
				assert.Equal(t, social.PostStatusError, *u.Status)
				assert.Nil(t, u.PublishedAt)
				assert.Nil(t, u.PostURL)
				return social.Post{}, nil
			})

		var slackMsg string
		f.slack.EXPECT().
			MustSend(gomock.Any(), gomock.Any()).
			Do(func(_ context.Context, req slack.Request) { slackMsg = flattenSlackRequest(req) })

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().PublishDrafts(t.Context(), social.PostOptions{Date: date})
		require.Error(t, err)
		require.Len(t, res, 1)
		assert.Contains(t, res[0].Err.Error(), "API down")
		assert.Contains(t, slackMsg, "Bluesky")
	})

	t.Run("Happy Path Publishes", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)

		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		bluesky.EXPECT().Post(gomock.Any(), gomock.Any()).Return(
			platform.PostResponse{PostURL: "https://bsky.app/profile/godaily/post/abc"}, nil,
		)
		f.posters = []platform.Poster{bluesky}

		f.posts.EXPECT().List(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, opts social.PostListOptions) ([]social.Post, error) {
				require.NotNil(t, opts.Status)
				assert.Equal(t, social.PostStatusDraft, *opts.Status)
				assert.Nil(t, opts.IssueID, "PublishDrafts must not scope by issue")
				return []social.Post{
					{ID: 11, Platform: "bluesky", Text: "publish me", Status: social.PostStatusDraft},
				}, nil
			})

		f.posts.EXPECT().
			Update(gomock.Any(), int64(11), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, u social.PostUpdate) (social.Post, error) {
				require.NotNil(t, u.Status)
				assert.Equal(t, social.PostStatusPublished, *u.Status)
				require.NotNil(t, u.PublishedAt)
				assert.WithinDuration(t, time.Now().UTC(), *u.PublishedAt, 5*time.Second)
				require.NotNil(t, u.PostURL)
				assert.Equal(t, "https://bsky.app/profile/godaily/post/abc", *u.PostURL)
				return social.Post{}, nil
			})

		var successMsg string
		f.slack.EXPECT().
			MustSend(gomock.Any(), gomock.Any()).
			Do(func(_ context.Context, req slack.Request) { successMsg = flattenSlackRequest(req) })

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().PublishDrafts(t.Context(), social.PostOptions{Date: date})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, "https://bsky.app/profile/godaily/post/abc", res[0].PostURL)
		assert.Contains(t, successMsg, "Bluesky")
		assert.Contains(t, successMsg, "https://bsky.app/profile/godaily/post/abc")
	})

	t.Run("Strips Em Dashes Before Posting", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)

		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		bluesky.EXPECT().
			Post(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req platform.PostRequest) (platform.PostResponse, error) {
				assert.NotContains(t, req.Text, "—", "em dash must be stripped before the platform API")
				assert.Equal(t, "GDG Berlin (Berlin) - Go meetup", req.Text)
				return platform.PostResponse{PostURL: "https://bsky.app/x/1"}, nil
			})
		f.posters = []platform.Poster{bluesky}

		f.posts.EXPECT().List(gomock.Any(), gomock.Any()).Return([]social.Post{
			{ID: 3, Platform: "bluesky", Text: "GDG Berlin (Berlin) — Go meetup", Status: social.PostStatusDraft},
		}, nil)
		f.posts.EXPECT().
			Update(gomock.Any(), int64(3), gomock.Any()).
			Return(social.Post{}, nil)
		f.slack.EXPECT().MustSend(gomock.Any(), gomock.Any())

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().PublishDrafts(t.Context(), social.PostOptions{Date: date})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.NotContains(t, res[0].Text, "—")
	})

	t.Run("Skips Unwired Platform", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		// Only bluesky is wired; the draft is for mastodon.
		f.posters = []platform.Poster{newMockPoster(f.ctrl, social.Bluesky)}

		f.posts.EXPECT().List(gomock.Any(), gomock.Any()).Return([]social.Post{
			{ID: 9, Platform: "mastodon", Text: "orphan", Status: social.PostStatusDraft},
		}, nil)

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().PublishDrafts(t.Context(), social.PostOptions{Date: date})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Platforms Filter", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		// Both posters wired, drafts for both, but caller restricts to mastodon.
		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		mastodon := newMockPoster(f.ctrl, social.Mastodon)
		mastodon.EXPECT().Post(gomock.Any(), gomock.Any()).Return(
			platform.PostResponse{PostURL: "https://mastodon.social/x/1"}, nil,
		)
		f.posters = []platform.Poster{bluesky, mastodon}

		f.posts.EXPECT().List(gomock.Any(), gomock.Any()).Return([]social.Post{
			{ID: 1, Platform: "bluesky", Text: "bsky", Status: social.PostStatusDraft},
			{ID: 2, Platform: "mastodon", Text: "masto", Status: social.PostStatusDraft},
		}, nil)
		f.posts.EXPECT().
			Update(gomock.Any(), int64(2), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, u social.PostUpdate) (social.Post, error) {
				require.NotNil(t, u.Status)
				assert.Equal(t, social.PostStatusPublished, *u.Status)
				return social.Post{}, nil
			})

		f.slack.EXPECT().MustSend(gomock.Any(), gomock.Any())

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().PublishDrafts(t.Context(), social.PostOptions{
			Date:      date,
			Platforms: []social.Platform{social.Mastodon},
		})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, social.Mastodon, res[0].Platform)
	})
}
