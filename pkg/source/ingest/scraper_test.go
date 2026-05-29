// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ingest

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
