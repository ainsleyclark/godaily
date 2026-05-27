// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_Post(t *testing.T) {
	t.Parallel()

	t.Run("No Posters Skip", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		res, err := f.service().Post(t.Context(), social.PostOptions{Date: time.Now()})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Issue Not Found", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		f.posters = []platform.Poster{newMockPoster(f.ctrl, social.Bluesky)}

		f.issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-20").
			Return(digest.Issue{}, store.ErrNotFound)

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		_, err := f.service().Post(t.Context(), social.PostOptions{Date: date})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no digest")
	})

	t.Run("No Items Skips", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		f.posters = []platform.Poster{newMockPoster(f.ctrl, social.Bluesky)}

		issue := sampleIssue()
		f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(issue, nil)
		f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().Post(t.Context(), social.PostOptions{Date: date})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Skips Already Posted", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		// bluesky.Post must NOT be called when HasPosted returns true.
		f.posters = []platform.Poster{bluesky}

		f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
		f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
		f.prompter.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(featureJSON(), nil)

		f.posts.EXPECT().HasPosted(gomock.Any(), int64(42), "bluesky").Return(true, nil)

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().Post(t.Context(), social.PostOptions{Date: date})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.True(t, res[0].Skipped)
	})

	t.Run("Dry Run", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)

		f.stubReframer(social.Bluesky, constReframer("dry-run text"))

		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		// bluesky.Post must NOT be called in dry-run.
		f.posters = []platform.Poster{bluesky}

		f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
		f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
		f.prompter.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(featureJSON(), nil)
		// posts.HasPosted + posts.Create must NOT be called in dry-run.

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().Post(t.Context(), social.PostOptions{Date: date, DryRun: true})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, "dry-run text", res[0].Text)
		assert.Empty(t, res[0].PostURL)
	})

	t.Run("Poster Error Notifies Slack", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)

		f.stubReframer(social.Bluesky, constReframer("ok"))

		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		bluesky.EXPECT().Post(gomock.Any(), gomock.Any()).Return(platform.PostResponse{}, errors.New("API down"))
		f.posters = []platform.Poster{bluesky}

		f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
		f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
		f.prompter.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(featureJSON(), nil)
		f.posts.EXPECT().HasPosted(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)

		// Capture the Slack notification so we can assert on its content.
		var slackMsg string
		f.slack.EXPECT().
			MustSend(gomock.Any(), gomock.Any()).
			Do(func(_ context.Context, m string) { slackMsg = m })

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().Post(t.Context(), social.PostOptions{Date: date})
		require.Error(t, err)
		require.Len(t, res, 1)
		assert.Contains(t, res[0].Err.Error(), "API down")
		assert.Contains(t, slackMsg, "bluesky")
	})

	t.Run("Platforms Filter", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)

		f.stubReframer(social.Mastodon, constReframer("mastodon text"))

		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		mastodon := newMockPoster(f.ctrl, social.Mastodon)
		mastodon.EXPECT().Post(gomock.Any(), gomock.Any()).Return(
			platform.PostResponse{PostURL: "https://mastodon.social/@godaily/9"}, nil,
		)
		f.posters = []platform.Poster{bluesky, mastodon}

		f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
		f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
		f.prompter.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(featureJSON(), nil)
		f.posts.EXPECT().HasPosted(gomock.Any(), gomock.Any(), "mastodon").Return(false, nil)
		f.posts.EXPECT().Create(gomock.Any(), gomock.Any()).Return(social.Post{}, nil)

		// Wet run posts a single platform — one success Slack notification.
		f.slack.EXPECT().MustSend(gomock.Any(), gomock.Any())

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().Post(t.Context(), social.PostOptions{
			Date:      date,
			Platforms: []social.Platform{social.Mastodon},
		})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, social.Mastodon, res[0].Platform)
	})

	t.Run("Happy Path", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		issue := sampleIssue()

		f.stubReframer(social.Bluesky, constReframer(
			"Go 1.30 lands generic inference improvements.\n\nhttps://go.dev/blog/go1.30\n#golang",
		))

		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		bluesky.EXPECT().Post(gomock.Any(), gomock.Any()).Return(
			platform.PostResponse{PostURL: "https://bsky.app/profile/godaily/post/abc"}, nil,
		)
		f.posters = []platform.Poster{bluesky}

		f.issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-20").Return(issue, nil)
		f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
		f.prompter.EXPECT().
			Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(featureJSON(), nil)

		f.posts.EXPECT().
			HasPosted(gomock.Any(), int64(42), "bluesky").Return(false, nil)
		f.posts.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, p social.Post) (social.Post, error) {
				require.NotNil(t, p.IssueID)
				assert.Equal(t, int64(42), *p.IssueID)
				assert.Equal(t, social.PostKindFeatured, p.Kind)
				assert.Equal(t, "bluesky", p.Platform)
				assert.Contains(t, p.Text, "Go 1.30")
				assert.Equal(t, "https://bsky.app/profile/godaily/post/abc", p.PostURL)
				p.ID = 1
				return p, nil
			})

		// One success Slack notification expected, carrying the post URL.
		var successMsg string
		f.slack.EXPECT().
			MustSend(gomock.Any(), gomock.Any()).
			Do(func(_ context.Context, m string) { successMsg = m })

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().Post(t.Context(), social.PostOptions{Date: date})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, social.Bluesky, res[0].Platform)
		assert.Equal(t, "https://bsky.app/profile/godaily/post/abc", res[0].PostURL)
		assert.False(t, res[0].Skipped)
		assert.Contains(t, successMsg, "featured")
		assert.Contains(t, successMsg, "Bluesky")
		assert.Contains(t, successMsg, "https://bsky.app/profile/godaily/post/abc")
	})
}
