// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

func TestProfile_Mention(t *testing.T) {
	t.Run("Returns the platform-specific handle when present", func(t *testing.T) {
		p := social.Profile{
			DisplayName: "Ardan Labs",
			Mentions: map[string]string{
				"bluesky": "@ardanlabs.com",
			},
		}
		assert.Equal(t, "@ardanlabs.com", p.Mention("bluesky"))
	})

	t.Run("Falls back to DisplayName when the platform has no handle", func(t *testing.T) {
		p := social.Profile{
			DisplayName: "Ardan Labs",
			Mentions: map[string]string{
				"bluesky": "@ardanlabs.com",
			},
		}
		assert.Equal(t, "Ardan Labs", p.Mention("linkedin"))
	})

	t.Run("Falls back to DisplayName when the handle is empty", func(t *testing.T) {
		p := social.Profile{
			DisplayName: "Ardan Labs",
			Mentions:    map[string]string{"mastodon": ""},
		}
		assert.Equal(t, "Ardan Labs", p.Mention("mastodon"))
	})

	t.Run("Empty Mentions map falls back to DisplayName", func(t *testing.T) {
		p := social.Profile{DisplayName: "Anonymous Coder"}
		assert.Equal(t, "Anonymous Coder", p.Mention("bluesky"))
	})
}

func TestProfileFor(t *testing.T) {
	t.Run("Known source returns a profile", func(t *testing.T) {
		p, ok := social.ProfileFor(news.SourceArdanLabs)
		assert.True(t, ok)
		assert.Equal(t, news.SourceArdanLabs, p.Source)
		assert.NotEmpty(t, p.DisplayName)
	})

	t.Run("Unknown source returns the zero value and false", func(t *testing.T) {
		p, ok := social.ProfileFor(news.Source("nonexistent"))
		assert.False(t, ok)
		assert.Equal(t, social.Profile{}, p)
	})
}

func TestProfiles_CuratedEntriesAreWellFormed(t *testing.T) {
	for src, p := range social.Profiles {
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
