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
)

var (
	spotNow = time.Date(2026, 5, 19, 15, 0, 0, 0, time.UTC)

	// Two stub profiles. Names are deliberately ordered so we know
	// alphabetical iteration picks "alpha_source" first.
	alphaProfile = social.Profile{
		Source:         news.Source("alpha_source"),
		DisplayName:    "Alpha",
		SourceURL:      "https://alpha.example",
		SpotlightBlurb: "alpha blurb",
		Mentions: map[string]string{
			"bluesky": "@alpha.example",
		},
	}
	bravoProfile = social.Profile{
		Source:         news.Source("bravo_source"),
		DisplayName:    "Bravo",
		SourceURL:      "https://bravo.example",
		SpotlightBlurb: "bravo blurb",
	}
)

func TestSpotlight_Kind(t *testing.T) {
	c := candidates.NewSpotlight(nil, nil)
	assert.Equal(t, social.PostKindSpotlight, c.Kind())
}

func TestSpotlight_Eligible(t *testing.T) {
	t.Run("Empty profile map is not eligible", func(t *testing.T) {
		c := candidates.NewSpotlight(map[news.Source]social.Profile{}, nil)
		_, ok, err := c.Eligible(context.Background(), spotNow)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("Picks first unposted source alphabetically", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)

		profiles := map[news.Source]social.Profile{
			alphaProfile.Source: alphaProfile,
			bravoProfile.Source: bravoProfile,
		}

		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), "spotlight:alpha_source", "bluesky").
			Return(false, nil)

		c := candidates.NewSpotlight(profiles, posts)
		cctx, ok, err := c.Eligible(context.Background(), spotNow)
		require.NoError(t, err)
		require.True(t, ok)

		assert.Equal(t, social.PostKindSpotlight, cctx.Kind)
		assert.Equal(t, "spotlight:alpha_source", cctx.Subject)
		assert.Equal(t, "https://alpha.example", cctx.URL)

		profile, ok := cctx.Payload.(social.Profile)
		require.True(t, ok, "Payload must be a social.Profile")
		assert.Equal(t, "Alpha", profile.DisplayName)
		assert.Equal(t, "@alpha.example", cctx.Mentions[social.Bluesky])
	})

	t.Run("Rotates past already-covered source", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)

		profiles := map[news.Source]social.Profile{
			alphaProfile.Source: alphaProfile,
			bravoProfile.Source: bravoProfile,
		}

		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), "spotlight:alpha_source", "bluesky").
			Return(true, nil)
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), "spotlight:bravo_source", "bluesky").
			Return(false, nil)

		c := candidates.NewSpotlight(profiles, posts)
		cctx, ok, err := c.Eligible(context.Background(), spotNow)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, "spotlight:bravo_source", cctx.Subject)
	})

	t.Run("All sources covered is not eligible", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)

		profiles := map[news.Source]social.Profile{
			alphaProfile.Source: alphaProfile,
		}

		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), "spotlight:alpha_source", "bluesky").
			Return(true, nil)

		c := candidates.NewSpotlight(profiles, posts)
		_, ok, err := c.Eligible(context.Background(), spotNow)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}
