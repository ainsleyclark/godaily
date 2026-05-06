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

package source

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArdanLabs_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/ardanlabs.xml")
	require.NoError(t, err)

	tt := map[string]struct {
		stub http.HandlerFunc
		want func([]news.Item, error)
	}{
		"Bad Request": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			want: func(items []news.Item, err error) {
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write(fixture)
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, news.Item{
					Source:    news.SourceArdanLabs,
					Title:     "Creativity, AI, and Supreme Robot with Victor Varnado",
					URL:       "https://www.buzzsprout.com/1466944/episodes/18828808-creativity-ai-and-supreme-robot-with-victor-varnado",
					Author:    &news.Author{Name: "Victor Varnado"},
					ImageURL:  "https://storage.buzzsprout.com/show.jpg",
					Snippet:   "Ale Kennedy talks with Victor Varnado about creativity and tech.",
					Tag:       news.TagPodcast,
					Score:     0.5,
					Published: time.Date(2026, time.March, 11, 14, 0, 0, 0, time.UTC),
				}, items[0])
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			got, err := ArdanLabs{url: s.URL}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}

func TestBuzzsproutEpisodeURL(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"Standard MP3":     {"https://www.buzzsprout.com/1/episodes/2-foo.mp3", "https://www.buzzsprout.com/1/episodes/2-foo"},
		"Already Stripped": {"https://www.buzzsprout.com/1/episodes/2-foo", "https://www.buzzsprout.com/1/episodes/2-foo"},
		"Empty":            {"", ""},
	}
	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, buzzsproutEpisodeURL(test.in))
		})
	}
}
