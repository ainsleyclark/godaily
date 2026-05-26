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

package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
)

func appWithCreds(cfg env.Config) *godaily.App {
	return &godaily.App{Config: &cfg}
}

func TestPostersForFlags(t *testing.T) {
	t.Parallel()

	blueskyApp := appWithCreds(env.Config{
		BlueskyHandle:      "godaily.bsky.social",
		BlueskyAppPassword: "app-pass",
	})
	linkedInApp := appWithCreds(env.Config{
		LinkedInOAuthToken: "tok",
		LinkedInOrgURN:     "urn:li:organization:1",
	})
	mastodonApp := appWithCreds(env.Config{
		MastodonServer:   "https://mastodon.social",
		MastodonAppToken: "token",
	})
	allApp := appWithCreds(env.Config{
		BlueskyHandle:      "godaily.bsky.social",
		BlueskyAppPassword: "app-pass",
		LinkedInOAuthToken: "tok",
		LinkedInOrgURN:     "urn:li:organization:1",
		MastodonServer:     "https://mastodon.social",
		MastodonAppToken:   "token",
	})

	t.Run("No platforms returns all configured", func(t *testing.T) {
		t.Parallel()
		got, err := postersForFlags(allApp, nil)
		require.NoError(t, err)
		assert.Len(t, got, 3)
	})

	t.Run("Empty app returns empty slice", func(t *testing.T) {
		t.Parallel()
		got, err := postersForFlags(appWithCreds(env.Config{}), nil)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("Platform filter selects only requested", func(t *testing.T) {
		t.Parallel()
		got, err := postersForFlags(allApp, []string{"bluesky"})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, social.Bluesky, got[0].Platform())
	})

	t.Run("LinkedIn creds", func(t *testing.T) {
		t.Parallel()
		got, err := postersForFlags(linkedInApp, []string{"linkedin"})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, social.LinkedIn, got[0].Platform())
	})

	t.Run("Mastodon creds", func(t *testing.T) {
		t.Parallel()
		got, err := postersForFlags(mastodonApp, []string{"mastodon"})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, social.Mastodon, got[0].Platform())
	})

	t.Run("Requesting unconfigured platform errors", func(t *testing.T) {
		t.Parallel()
		_, err := postersForFlags(blueskyApp, []string{"linkedin"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "linkedin")
	})

	t.Run("Unknown platform name errors", func(t *testing.T) {
		t.Parallel()
		_, err := postersForFlags(allApp, []string{"twitter"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "twitter")
	})
}

func TestPrintResults(t *testing.T) {
	// Not parallel — subtests swap os.Stdout and cannot run concurrently.

	capture := func(f func()) string {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		f()
		_ = w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		return buf.String()
	}

	t.Run("Empty results", func(t *testing.T) {
		out := capture(func() { printResults(nil) })
		assert.Contains(t, out, "no posts produced")
	})

	t.Run("Error result", func(t *testing.T) {
		out := capture(func() {
			printResults([]social.PostResult{{
				Platform: social.LinkedIn,
				Err:      errors.New("bad token"),
			}})
		})
		assert.Contains(t, out, "linkedin")
		assert.Contains(t, out, "bad token")
	})

	t.Run("Skipped result", func(t *testing.T) {
		out := capture(func() {
			printResults([]social.PostResult{{
				Platform: social.Bluesky,
				Skipped:  true,
			}})
		})
		assert.Contains(t, out, "skipped")
	})

	t.Run("Dry-run result prints text", func(t *testing.T) {
		out := capture(func() {
			printResults([]social.PostResult{{
				Platform: social.Mastodon,
				Text:     "Hello world",
			}})
		})
		assert.Contains(t, out, "dry-run")
		assert.Contains(t, out, "Hello world")
	})

	t.Run("Successful post shows URL", func(t *testing.T) {
		url := "https://bsky.app/profile/godaily.bsky.social/post/abc"
		out := capture(func() {
			printResults([]social.PostResult{{
				Platform: social.Bluesky,
				PostURL:  url,
			}})
		})
		assert.Contains(t, out, "posted")
		assert.Contains(t, out, url)
	})
}

func TestParsePlatforms(t *testing.T) {
	t.Parallel()

	t.Run("Empty input yields nil", func(t *testing.T) {
		t.Parallel()
		got, err := parsePlatforms(nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("All three platforms parsed in order", func(t *testing.T) {
		t.Parallel()
		got, err := parsePlatforms([]string{"bluesky", "linkedin", "mastodon"})
		require.NoError(t, err)
		assert.Equal(t, []social.Platform{
			social.Bluesky,
			social.LinkedIn,
			social.Mastodon,
		}, got)
	})

	t.Run("Whitespace + casing tolerated", func(t *testing.T) {
		t.Parallel()
		got, err := parsePlatforms([]string{"  Bluesky  ", "LINKEDIN"})
		require.NoError(t, err)
		assert.Equal(t, []social.Platform{
			social.Bluesky,
			social.LinkedIn,
		}, got)
	})

	t.Run("Unknown platform errors", func(t *testing.T) {
		t.Parallel()
		_, err := parsePlatforms([]string{"twitter"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "twitter")
	})
}
