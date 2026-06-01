// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// golangCafeFixture has one listing with salary/remote markers and one bare
// listing whose title doesn't even mention Go — on a Go-only board it is still
// kept (no keyword filter).
const golangCafeFixture = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Senior Go Engineer at Acme Corp - Remote ($120k - $160k)</title>
      <link>https://golang.cafe/job/1</link>
      <description>Build distributed systems in Go.</description>
      <pubDate>Mon, 30 Dec 2024 10:00:00 +0000</pubDate>
    </item>
    <item>
      <title>Backend Developer at Beta Ltd</title>
      <link>https://golang.cafe/job/2</link>
      <description>Join our backend team.</description>
      <pubDate>Mon, 30 Dec 2024 10:00:00 +0000</pubDate>
    </item>
    <item>
      <title>Missing link is dropped</title>
      <link></link>
      <description>No link.</description>
      <pubDate>Mon, 30 Dec 2024 10:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

func TestGolangCafe_Fetch(t *testing.T) {
	t.Parallel()

	fixedNow := func() time.Time {
		return time.Date(2024, time.December, 31, 0, 0, 0, 0, time.UTC)
	}

	tt := map[string]struct {
		stub http.HandlerFunc
		want func(t *testing.T, items []news.Item, err error)
	}{
		"Bad Request": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(golangCafeFixture))
				require.NoError(t, err)
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 2) // Listing with empty link dropped.

				senior := items[0]
				assert.Equal(t, news.SourceGolangCafe, senior.Source)
				assert.Equal(t, news.TagJobs, senior.Tag)
				assert.Equal(t, "Acme Corp · Senior Go Engineer", senior.Title)
				assert.Equal(t, "https://golang.cafe/job/1", senior.URL)
				require.NotNil(t, senior.Author)
				assert.Equal(t, "Acme Corp", senior.Author.Name)
				assert.Equal(t, fixedNow(), senior.Published)

				// Go-only board: a non-Go title is still kept and ranked.
				backend := items[1]
				assert.Equal(t, "Beta Ltd · Backend Developer", backend.Title)
				assert.Less(t, backend.Score, senior.Score)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			got, err := GolangCafe{url: s.URL, now: fixedNow}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}
