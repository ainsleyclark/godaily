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

package social_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	"github.com/ainsleyclark/godaily/pkg/services/social"
)

func TestSourceProfile_Mention_FallsBackToDisplayName(t *testing.T) {
	p := social.SourceProfile{
		DisplayName: "Ardan Labs",
		Mentions: map[socialgw.Platform]string{
			socialgw.PlatformBluesky: "@ardanlabs.com",
		},
	}

	assert.Equal(t, "@ardanlabs.com", p.Mention(socialgw.PlatformBluesky),
		"Bluesky mention should come from the map")
	assert.Equal(t, "Ardan Labs", p.Mention(socialgw.PlatformLinkedIn),
		"missing platforms must fall back to DisplayName")
	assert.Equal(t, "Ardan Labs", p.Mention(socialgw.PlatformMastodon),
		"empty Mastodon mention must fall back to DisplayName")
}

func TestSourceProfile_Mention_EmptyMapFallsBack(t *testing.T) {
	p := social.SourceProfile{DisplayName: "Anonymous Coder"}
	assert.Equal(t, "Anonymous Coder", p.Mention(socialgw.PlatformBluesky))
}

func TestSourceProfiles_ConfigSanityChecks(t *testing.T) {
	// Every curated profile should be addressable, have a blurb, and
	// have a source URL. These are the inputs the AI prompt depends on.
	for src, p := range social.SourceProfiles {
		t.Run(string(src), func(t *testing.T) {
			assert.NotEmpty(t, p.DisplayName, "%s: DisplayName is required", src)
			assert.NotEmpty(t, p.SpotlightBlurb, "%s: SpotlightBlurb is required", src)
			assert.NotEmpty(t, p.SourceURL, "%s: SourceURL is required", src)
			assert.Equal(t, src, p.Source, "%s: Source key/value mismatch", src)

			// Profile source must be a known news.Source — sanity check
			// against typos like a renamed constant.
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
