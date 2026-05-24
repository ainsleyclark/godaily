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

package news_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

func TestSocialProfile_Mention(t *testing.T) {
	t.Run("Returns the platform-specific handle when present", func(t *testing.T) {
		p := news.SocialProfile{
			DisplayName: "Ardan Labs",
			Mentions: map[string]string{
				"bluesky": "@ardanlabs.com",
			},
		}
		assert.Equal(t, "@ardanlabs.com", p.Mention("bluesky"))
	})

	t.Run("Falls back to DisplayName when the platform has no handle", func(t *testing.T) {
		p := news.SocialProfile{
			DisplayName: "Ardan Labs",
			Mentions: map[string]string{
				"bluesky": "@ardanlabs.com",
			},
		}
		assert.Equal(t, "Ardan Labs", p.Mention("linkedin"))
	})

	t.Run("Falls back to DisplayName when the handle is empty", func(t *testing.T) {
		p := news.SocialProfile{
			DisplayName: "Ardan Labs",
			Mentions:    map[string]string{"mastodon": ""},
		}
		assert.Equal(t, "Ardan Labs", p.Mention("mastodon"))
	})

	t.Run("Empty Mentions map falls back to DisplayName", func(t *testing.T) {
		p := news.SocialProfile{DisplayName: "Anonymous Coder"}
		assert.Equal(t, "Anonymous Coder", p.Mention("bluesky"))
	})
}

func TestSocialProfileFor(t *testing.T) {
	t.Run("Known source returns a profile", func(t *testing.T) {
		p, ok := news.SocialProfileFor(news.SourceArdanLabs)
		assert.True(t, ok)
		assert.Equal(t, news.SourceArdanLabs, p.Source)
		assert.NotEmpty(t, p.DisplayName)
	})

	t.Run("Unknown source returns the zero value and false", func(t *testing.T) {
		p, ok := news.SocialProfileFor(news.Source("nonexistent"))
		assert.False(t, ok)
		assert.Equal(t, news.SocialProfile{}, p)
	})
}

func TestSocialProfiles_CuratedEntriesAreWellFormed(t *testing.T) {
	// Every curated profile must be addressable, have a blurb and a
	// source URL — these are the inputs the AI prompts depend on. The
	// loop catches typos and missing fields when sources are added.
	for src, p := range news.SocialProfiles {
		src, p := src, p
		t.Run(string(src), func(t *testing.T) {
			assert.NotEmpty(t, p.DisplayName, "DisplayName is required")
			assert.NotEmpty(t, p.SpotlightBlurb, "SpotlightBlurb is required")
			assert.NotEmpty(t, p.SourceURL, "SourceURL is required")
			assert.Equal(t, src, p.Source, "map key and Source field must agree")

			found := false
			for _, known := range news.Sources {
				if known == src {
					found = true
					break
				}
			}
			assert.True(t, found, "%s: not in news.Sources", src)
		})
	}
}
