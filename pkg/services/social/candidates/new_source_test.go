// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package candidates_test

import (
	"context"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidates"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/rotation"
)

var nsNow = time.Date(2026, 5, 19, 15, 0, 0, 0, time.UTC)

func nsProfile(name string, announceable bool) social.Profile {
	return social.Profile{
		Source:         news.Source(name),
		DisplayName:    name,
		SourceURL:      "https://" + name + ".example",
		SpotlightBlurb: name + " blurb",
		Announceable:   announceable,
		Mentions: []social.Mention{
			{Platform: social.Bluesky, Handle: "@" + name},
		},
	}
}

func TestNewSource_Kind(t *testing.T) {
	c := candidates.NewNewSource(nil, nil)
	assert.Equal(t, social.PostKindNewSource, c.Kind())
}

func TestNewSource_Eligible(t *testing.T) {
	t.Run("Empty profile map is not eligible", func(t *testing.T) {
		c := candidates.NewNewSource(map[news.Source]social.Profile{}, nil)
		_, ok, err := c.Eligible(context.Background(), nsNow)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("Picks first un-announced Announceable source alphabetically", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)

		profiles := map[news.Source]social.Profile{
			"alpha": nsProfile("alpha", true),
			"bravo": nsProfile("bravo", true),
			"zulu":  nsProfile("zulu", false), // not announceable, skipped entirely
		}

		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), "new_source:alpha", "bluesky").
			Return(false, nil)

		c := candidates.NewNewSource(profiles, posts)
		cctx, ok, err := c.Eligible(context.Background(), nsNow)
		require.NoError(t, err)
		require.True(t, ok)

		assert.Equal(t, social.PostKindNewSource, cctx.Kind)
		assert.Equal(t, "new_source:alpha", cctx.Subject)
		assert.Equal(t, "https://alpha.example", cctx.URL)
	})

	t.Run("Skips already-announced and picks next", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)

		profiles := map[news.Source]social.Profile{
			"alpha": nsProfile("alpha", true),
			"bravo": nsProfile("bravo", true),
		}

		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), "new_source:alpha", "bluesky").
			Return(true, nil)
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), "new_source:bravo", "bluesky").
			Return(false, nil)

		c := candidates.NewNewSource(profiles, posts)
		cctx, ok, err := c.Eligible(context.Background(), nsNow)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, "new_source:bravo", cctx.Subject)
	})

	t.Run("All announceable sources covered is not eligible", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)

		profiles := map[news.Source]social.Profile{
			"alpha": nsProfile("alpha", true),
		}

		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), "new_source:alpha", "bluesky").
			Return(true, nil)

		c := candidates.NewNewSource(profiles, posts)
		_, ok, err := c.Eligible(context.Background(), nsNow)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("Non-Announceable sources are skipped silently", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)

		profiles := map[news.Source]social.Profile{
			"silent": nsProfile("silent", false), // not eligible, no DB call expected
		}

		c := candidates.NewNewSource(profiles, posts)
		_, ok, err := c.Eligible(context.Background(), nsNow)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestNewSource_PayloadShape(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := mocksocial.NewMockPostRepository(ctrl)

	profiles := map[news.Source]social.Profile{
		"alpha": nsProfile("alpha", true),
	}

	posts.EXPECT().
		HasPostedBySubject(gomock.Any(), "new_source:alpha", "bluesky").
		Return(false, nil)

	c := candidates.NewNewSource(profiles, posts)
	cctx, ok, err := c.Eligible(context.Background(), nsNow)
	require.NoError(t, err)
	require.True(t, ok)

	profile, ok := cctx.Payload.(social.Profile)
	require.True(t, ok, "Payload must be a social.Profile")
	assert.Equal(t, "alpha", profile.DisplayName)

	// The mentions map carried on the CandidateContext is the typed version
	// used by the publish loop, ensuring socialgw.Platform keys survive.
	assert.NotEmpty(t, cctx.Mentions, "Mentions should be populated for announceable profiles")

	// Sanity-check the payload the prompt is going to receive.
	pl := rotation.NewSourcePayload{
		DisplayName: profile.DisplayName,
		Mention:     profile.Mention("bluesky"),
		Blurb:       profile.SpotlightBlurb,
		URL:         profile.SourceURL,
	}
	assert.Equal(t, "@alpha", pl.Mention)
	assert.Equal(t, "alpha blurb", pl.Blurb)
}
