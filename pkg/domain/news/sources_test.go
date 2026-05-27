// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package news

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSource_String(t *testing.T) {
	input := SourceDevTo
	got := input.String()
	assert.IsType(t, got, "devto")
}

func TestSource_NiceName(t *testing.T) {
	t.Parallel()

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()

		input := Source("wrong")
		got := input.NiceName()
		assert.Empty(t, got)
	})

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		input := SourceDevTo
		got := input.NiceName()
		assert.IsType(t, got, "devto")
	})
}

func TestSource_Priority(t *testing.T) {
	t.Parallel()

	values := map[string]struct {
		source Source
		want   int
	}{
		"Go Releases":        {source: SourceGoRelease, want: 21},
		"Go Blog":            {source: SourceGoBlog, want: 20},
		"GitHub":             {source: SourceGitHub, want: 19},
		"GitHub Trending":    {source: SourceGitHubTrending, want: 18},
		"Hacker News":        {source: SourceHN, want: 17},
		"Lobsters":           {source: SourceLobsters, want: 16},
		"Reddit":             {source: SourceReddit, want: 15},
		"JetBrains":          {source: SourceJetBrains, want: 14},
		"Dev.to":             {source: SourceDevTo, want: 13},
		"GolangBridge":       {source: SourceGolangBridge, want: 12},
		"freeCodeCamp":       {source: SourceFreeCodeCamp, want: 11},
		"Meetup":             {source: SourceMeetup, want: 10},
		"go podcast()":       {source: SourceGoPodcast, want: 9},
		"Fallthrough":        {source: SourceFallthrough, want: 8},
		"Ardan Labs Podcast": {source: SourceArdanLabs, want: 7},
		"YouTube":            {source: SourceYouTube, want: 6},
		"Mastodon":           {source: SourceMastodon, want: 5},
		"Awesome Go":         {source: SourceAwesomeGo, want: 4},
		"Medium":             {source: SourceMedium, want: 3},
	}
	for name, test := range values {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, test.source.Priority())
		})
	}

	t.Run("All Sources Covered", func(t *testing.T) {
		t.Parallel()
		for _, s := range Sources {
			assert.Greater(t, s.Priority(), 0, "source %q must have a non-zero priority", s)
		}
	})

	t.Run("Orders Go Releases Above Medium", func(t *testing.T) {
		t.Parallel()
		assert.Greater(t, SourceGoRelease.Priority(), SourceGoBlog.Priority())
		assert.Greater(t, SourceGoBlog.Priority(), SourceGitHub.Priority())
		assert.Greater(t, SourceGitHub.Priority(), SourceGitHubTrending.Priority())
		assert.Greater(t, SourceGitHubTrending.Priority(), SourceHN.Priority())
		assert.Greater(t, SourceHN.Priority(), SourceLobsters.Priority())
		assert.Greater(t, SourceLobsters.Priority(), SourceReddit.Priority())
		assert.Greater(t, SourceReddit.Priority(), SourceJetBrains.Priority())
		assert.Greater(t, SourceJetBrains.Priority(), SourceDevTo.Priority())
		assert.Greater(t, SourceDevTo.Priority(), SourceGolangBridge.Priority())
		assert.Greater(t, SourceGolangBridge.Priority(), SourceFreeCodeCamp.Priority())
		assert.Greater(t, SourceFreeCodeCamp.Priority(), SourceMeetup.Priority())
		assert.Greater(t, SourceMeetup.Priority(), SourceGoPodcast.Priority())
		assert.Greater(t, SourceGoPodcast.Priority(), SourceFallthrough.Priority())
		assert.Greater(t, SourceFallthrough.Priority(), SourceArdanLabs.Priority())
		assert.Greater(t, SourceArdanLabs.Priority(), SourceYouTube.Priority())
		assert.Greater(t, SourceYouTube.Priority(), SourceMastodon.Priority())
		assert.Greater(t, SourceMastodon.Priority(), SourceAwesomeGo.Priority())
		assert.Greater(t, SourceAwesomeGo.Priority(), SourceMedium.Priority())
	})

	t.Run("All Priorities Are Unique", func(t *testing.T) {
		t.Parallel()
		seen := make(map[int]Source, len(Sources))
		for _, s := range Sources {
			p := s.Priority()
			if other, ok := seen[p]; ok {
				t.Errorf("priority %d is shared by %q and %q", p, other, s)
			}
			seen[p] = s
		}
	})
}
