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
		"Go Blog":         {source: SourceGoBlog, want: 10},
		"GitHub":          {source: SourceGitHub, want: 9},
		"GitHub Trending": {source: SourceGitHubTrending, want: 8},
		"Hacker News":     {source: SourceHN, want: 7},
		"Lobsters":        {source: SourceLobsters, want: 6},
		"Reddit":          {source: SourceReddit, want: 5},
		"Dev.to":          {source: SourceDevTo, want: 4},
		"GolangBridge":    {source: SourceGolangBridge, want: 3},
		"YouTube":         {source: SourceYouTube, want: 2},
		"Medium":          {source: SourceMedium, want: 1},
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

	t.Run("Orders Go Blog Above Medium", func(t *testing.T) {
		t.Parallel()
		assert.Greater(t, SourceGoBlog.Priority(), SourceGitHub.Priority())
		assert.Greater(t, SourceGitHub.Priority(), SourceGitHubTrending.Priority())
		assert.Greater(t, SourceGitHubTrending.Priority(), SourceHN.Priority())
		assert.Greater(t, SourceHN.Priority(), SourceLobsters.Priority())
		assert.Greater(t, SourceLobsters.Priority(), SourceReddit.Priority())
		assert.Greater(t, SourceReddit.Priority(), SourceDevTo.Priority())
		assert.Greater(t, SourceDevTo.Priority(), SourceGolangBridge.Priority())
		assert.Greater(t, SourceGolangBridge.Priority(), SourceYouTube.Priority())
		assert.Greater(t, SourceYouTube.Priority(), SourceMedium.Priority())
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
