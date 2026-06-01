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
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_DraftFeatured(t *testing.T) {
	t.Parallel()

	t.Run("No Posters Skip", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		res, err := f.service().DraftFeatured(t.Context(), social.PostOptions{Date: time.Now()})
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
		_, err := f.service().DraftFeatured(t.Context(), social.PostOptions{Date: date})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no digest")
	})

	t.Run("No Items Skips", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		f.posters = []platform.Poster{newMockPoster(f.ctrl, social.Bluesky)}

		f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
		f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().DraftFeatured(t.Context(), social.PostOptions{Date: date})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("AI Feature Fails", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		f.posters = []platform.Poster{newMockPoster(f.ctrl, social.Bluesky)}

		f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
		f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
		f.prompter.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("AI offline"))

		f.slack.EXPECT().MustSend(gomock.Any(), gomock.Any())

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		_, err := f.service().DraftFeatured(t.Context(), social.PostOptions{Date: date})
		require.Error(t, err)
	})

	t.Run("Dry Run Skips Persist", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)

		f.stubReframer(social.Bluesky, constReframer("dry-run text"))

		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		f.posters = []platform.Poster{bluesky}

		f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
		f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
		f.prompter.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(featureJSON(), nil)
		// posts.DeleteDraftsByIssue and posts.Create must NOT be called in dry-run.

		// Slack draft-preview ping still fires (drafts exist in memory).
		f.slack.EXPECT().MustSend(gomock.Any(), gomock.Any())

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().DraftFeatured(t.Context(), social.PostOptions{Date: date, DryRun: true})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, "dry-run text", res[0].Text)
	})

	t.Run("Happy Path Persists Draft", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		issue := sampleIssue()

		f.stubReframer(social.Bluesky, constReframer(
			"Go 1.30 lands generic inference improvements.\n\nhttps://go.dev/blog/go1.30\n#golang",
		))

		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		f.posters = []platform.Poster{bluesky}

		f.issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-20").Return(issue, nil)
		f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
		f.prompter.EXPECT().
			Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(featureJSON(), nil)

		f.posts.EXPECT().DeleteDraftsByIssue(gomock.Any(), int64(42)).Return(nil)
		f.posts.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, p social.Post) (social.Post, error) {
				require.NotNil(t, p.IssueID)
				assert.Equal(t, int64(42), *p.IssueID)
				assert.Equal(t, social.PostKindFeatured, p.Kind)
				assert.Equal(t, social.PostStatusDraft, p.Status)
				assert.Equal(t, "bluesky", p.Platform)
				assert.Equal(t, "go_release", p.MentionSource)
				assert.Contains(t, p.Text, "Go 1.30")
				assert.Empty(t, p.PostURL)
				p.ID = 1
				return p, nil
			})

		var slackMsg string
		f.slack.EXPECT().
			MustSend(gomock.Any(), gomock.Any()).
			Do(func(_ context.Context, req slack.Request) { slackMsg = flattenSlackRequest(req) })

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().DraftFeatured(t.Context(), social.PostOptions{Date: date})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, social.Bluesky, res[0].Platform)
		assert.Contains(t, res[0].Text, "Go 1.30")
		assert.Contains(t, slackMsg, "drafts")
		assert.Contains(t, slackMsg, "Bluesky")
	})

	t.Run("Platforms Filter", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)

		f.stubReframer(social.Mastodon, constReframer("mastodon text"))

		bluesky := newMockPoster(f.ctrl, social.Bluesky)
		mastodon := newMockPoster(f.ctrl, social.Mastodon)
		f.posters = []platform.Poster{bluesky, mastodon}

		f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
		f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
		f.prompter.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(featureJSON(), nil)
		f.posts.EXPECT().DeleteDraftsByIssue(gomock.Any(), gomock.Any()).Return(nil)
		f.posts.EXPECT().Create(gomock.Any(), gomock.Any()).Return(social.Post{}, nil)
		f.slack.EXPECT().MustSend(gomock.Any(), gomock.Any())

		date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		res, err := f.service().DraftFeatured(t.Context(), social.PostOptions{
			Date:      date,
			Platforms: []social.Platform{social.Mastodon},
		})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, social.Mastodon, res[0].Platform)
	})
}
