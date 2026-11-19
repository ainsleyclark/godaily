// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ingest

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScraperURL(t *testing.T) {
	t.Parallel()

	target := "https://www.reddit.com/r/golang/new.json"
	proxy := func(key string) string {
		return fmt.Sprintf("%s?api_key=%s&premium=true&url=%s", scraperAPIBase, key, url.QueryEscape(target))
	}

	tt := map[string]struct {
		keys []string
		opts []ScraperOption
		want string
	}{
		"No keys returns target unchanged": {
			keys: nil,
			want: target,
		},
		"Single key always proxies": {
			keys: []string{"key1"},
			want: proxy("key1"),
		},
		"Two keys selects by day parity": {
			keys: []string{"key_even", "key_odd"},
			want: proxy([]string{"key_even", "key_odd"}[time.Now().UTC().Day()%2]),
		},
		"WithKeepHeaders adds keep_headers param": {
			keys: []string{"key1"},
			opts: []ScraperOption{WithKeepHeaders()},
			want: fmt.Sprintf("%s?api_key=key1&keep_headers=true&premium=true&url=%s", scraperAPIBase, url.QueryEscape(target)),
		},
		"Options ignored when no keys": {
			keys: nil,
			opts: []ScraperOption{WithKeepHeaders()},
			want: target,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, ScraperURL(test.keys, target, test.opts...))
		})
	}
}

func TestRedactScraperKey(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input string
		want  string
	}{
		"Redacts api_key": {
			input: "http://api.scraperapi.com?api_key=secret&premium=true&url=https%3A%2F%2Fexample.com",
			want:  "http://api.scraperapi.com?api_key=REDACTED&premium=true&url=https%3A%2F%2Fexample.com",
		},
		"No key unchanged": {
			input: "https://arctic-shift.photon-reddit.com/api/posts/search?subreddit=golang",
			want:  "https://arctic-shift.photon-reddit.com/api/posts/search?subreddit=golang",
		},
		"Unparseable unchanged": {
			input: "://bad url",
			want:  "://bad url",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, redactScraperKey(test.input))
		})
	}
}

func TestFetch_RedactsScraperKeyInErrors(t *testing.T) {
	t.Parallel()

	// Nothing listens on port 1, so the request fails in the transport and
	// net/http embeds the full URL in the error.
	target := ScraperURL([]string{"supersecretkey"}, "https://www.reddit.com/r/golang/new.json")
	target = "http://127.0.0.1:1/" + target[len(scraperAPIBase):]

	_, err := Fetch[map[string]any](t.Context(), target, "reddit", json.Unmarshal)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "supersecretkey")
	assert.Contains(t, err.Error(), "api_key=REDACTED")

	_, err = FetchHTML(t.Context(), target, "reddit")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "supersecretkey")
}
