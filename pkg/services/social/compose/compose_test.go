// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compose

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

const longURL = "https://github.com/someone/a-very-long-repository-name/pull/12345"

func TestLength(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		platform social.Platform
		text     string
		want     int
	}{
		"Plain text counts runes": {
			platform: social.Mastodon,
			text:     "héllo",
			want:     5,
		},
		"Mastodon counts every URL as 23": {
			platform: social.Mastodon,
			text:     "See " + longURL,
			want:     4 + 23,
		},
		"Mastodon keeps trailing punctuation outside the URL": {
			platform: social.Mastodon,
			text:     longURL + ".",
			want:     23 + 1,
		},
		"Bluesky counts the shortened display form": {
			platform: social.Bluesky,
			text:     "See " + longURL,
			want:     4 + 30,
		},
		"LinkedIn counts raw runes": {
			platform: social.LinkedIn,
			text:     "See " + longURL,
			want:     4 + utf8.RuneCountInString(longURL),
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, Length(tc.platform, tc.text))
		})
	}
}

func TestShortenURL(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"Strips scheme and www":       {in: "https://www.go.dev/dl/", want: "go.dev/dl"},
		"Short URL kept whole":        {in: "https://go.dev/blog/go1.30", want: "go.dev/blog/go1.30"},
		"Long URL capped with …":      {in: longURL, want: "github.com/someone/a-very-lon…"},
		"UTM params hidden":           {in: "https://godaily.dev/?utm_source=x&utm_medium=y&utm_campaign=z", want: "godaily.dev"},
		"Non-UTM query kept":          {in: "https://youtube.com/watch?v=abc", want: "youtube.com/watch?v=abc"},
		"Unparseable input is capped": {in: "not a url at all but quite a long one indeed", want: "not a url at all but quite a …"},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ShortenURL(tc.in))
		})
	}
}

func TestSubscribeLine(t *testing.T) {
	t.Parallel()

	got := SubscribeLine(social.Mastodon, "recap")
	assert.True(t, strings.HasPrefix(got, "Subscribe: https://godaily.dev/?"))
	assert.Contains(t, got, "utm_source=social-mastodon")
	assert.Contains(t, got, "utm_medium=social")
	assert.Contains(t, got, "utm_campaign=recap")
}

func TestWithFooter(t *testing.T) {
	t.Parallel()

	for _, plat := range []social.Platform{social.Bluesky, social.LinkedIn, social.Mastodon} {
		t.Run(plat.String(), func(t *testing.T) {
			t.Parallel()

			footer := SubscribeLine(plat, "recap")

			t.Run("Short body is joined with a blank line", func(t *testing.T) {
				t.Parallel()
				got := WithFooter(plat, "Hello.", footer)
				assert.Equal(t, "Hello.\n\n"+footer, got)
			})

			t.Run("Long body is trimmed and the footer always survives", func(t *testing.T) {
				t.Parallel()
				body := strings.Repeat("A complete sentence about Go.\n", 80)
				got := WithFooter(plat, body, footer)
				assert.LessOrEqual(t, Length(plat, got), Limit(plat))
				assert.True(t, strings.HasSuffix(got, "\n\n"+footer))
				assert.NotContains(t, got, "…", "whole lines are dropped before any mid-sentence cut")
			})

			t.Run("Empty footer just fits the body", func(t *testing.T) {
				t.Parallel()
				got := WithFooter(plat, strings.Repeat("word ", 400), "")
				assert.LessOrEqual(t, Length(plat, got), Limit(plat))
			})
		})
	}
}

func TestFitTo(t *testing.T) {
	t.Parallel()

	t.Run("Fitting text is unchanged", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "abc", FitTo(social.LinkedIn, " abc ", 10))
	})

	t.Run("Drops trailing lines before cutting words", func(t *testing.T) {
		t.Parallel()
		got := FitTo(social.LinkedIn, "First line.\nSecond line.\nThird line.", 25)
		assert.Equal(t, "First line.\nSecond line.", got)
	})

	t.Run("Single long line is cut at a word boundary", func(t *testing.T) {
		t.Parallel()
		got := FitTo(social.LinkedIn, "one two three four five", 12)
		assert.Equal(t, "one two…", got)
	})

	t.Run("Respects platform URL weighting", func(t *testing.T) {
		t.Parallel()
		text := "Read this " + longURL
		// Raw rune count is well over 40, but Mastodon counts the URL as 23.
		assert.Equal(t, text, FitTo(social.Mastodon, text, 40))
	})
}
