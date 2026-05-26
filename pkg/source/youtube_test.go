// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYouTube_Fetch(t *testing.T) {
	t.Parallel()

	// YouTube Data API v3 sample response shapes. The search endpoint returns
	// snippet-only data; a second call to videos.list enriches each result
	// with view count statistics for quality-based scoring.
	fixture, err := os.ReadFile("testdata/youtube.json")
	require.NoError(t, err)
	statsFixture, err := os.ReadFile("testdata/youtube-stats.json")
	require.NoError(t, err)

	tt := map[string]struct {
		key  string
		stub http.HandlerFunc
		want func([]news.Item, error)
	}{
		"Missing API Key": {
			key:  "",
			stub: nil,
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Nil(t, items)
			},
		},
		"Bad Request": {
			key: "test-key",
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			want: func(items []news.Item, err error) {
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK - Stats Unavailable": {
			key: "test-key",
			stub: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("part") == "statistics" {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write(fixture)
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, float64(0.5), items[0].Score) // flat score: no stats
			},
		},
		"OK": {
			key: "test-key",
			stub: func(w http.ResponseWriter, r *http.Request) {
				var body []byte
				if r.URL.Query().Get("part") == "statistics" {
					body = statsFixture
				} else {
					body = fixture
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write(body)
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, news.Item{
					Source:    news.SourceYouTube,
					Title:     "Go Concurrency Patterns",
					URL:       "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
					ImageURL:  "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
					Author:    &news.Author{Name: "GopherCon", Username: "UCx0L2ZdYfiq-tsAXb8IXpQg", ProfileURL: "https://www.youtube.com/channel/UCx0L2ZdYfiq-tsAXb8IXpQg"},
					Snippet:   "An introduction to concurrency patterns in Go.",
					Tag:       news.TagVideo,
					Score:     news.ScoreOf(news.SourceYouTube, news.TagVideo, 2500, true),
					Published: time.Date(2024, 4, 25, 14, 0, 0, 0, time.UTC),
				}, items[0])
			},
		},
		// Two videos returned from search; stats only covers one of them.
		// The video with stats gets a view-count score; the other falls back
		// to the flat 0.5 score rather than erroring.
		"OK - Partial Stats": {
			key: "test-key",
			stub: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if r.URL.Query().Get("part") == "statistics" {
					// Only vid-aaa has stats; vid-bbb is absent.
					_, _ = w.Write([]byte(`{"items":[{"id":"vid-aaa","statistics":{"viewCount":"3000"}}]}`))
				} else {
					_, _ = w.Write([]byte(`{"items":[` +
						`{"id":{"videoId":"vid-aaa"},"snippet":{"publishedAt":"2024-04-25T14:00:00Z","channelId":"c1","title":"A","description":"","channelTitle":"Ch1"}},` +
						`{"id":{"videoId":"vid-bbb"},"snippet":{"publishedAt":"2024-04-25T13:00:00Z","channelId":"c2","title":"B","description":"","channelTitle":"Ch2"}}` +
						`]}`))
				}
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 2)
				assert.Equal(t, news.ScoreOf(news.SourceYouTube, news.TagVideo, 3000, true), items[0].Score)
				assert.Equal(t, float64(0.5), items[1].Score) // no stats → flat
			},
		},
		// A video whose title is entirely Cyrillic must be dropped by the shared
		// isEnglishTitle filter in TransformAll regardless of view count.
		"Filters Cyrillic title": {
			key: "test-key",
			stub: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if r.URL.Query().Get("part") == "statistics" {
					_, _ = w.Write([]byte(`{"items":[{"id":"vid-ru","statistics":{"viewCount":"5000"}}]}`))
				} else {
					_, _ = w.Write([]byte(`{"items":[{"id":{"videoId":"vid-ru"},"snippet":{"publishedAt":"2024-04-25T14:00:00Z","channelId":"c1","title":"Сравнимые типы данных в Go #shorts","description":"","channelTitle":"RuChannel"}}]}`))
				}
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Empty(t, items)
			},
		},
		// Two videos with different view counts must produce distinct scores
		// so the higher-viewed video sorts above the lower one.
		"OK - Scores Reflect View Count": {
			key: "test-key",
			stub: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if r.URL.Query().Get("part") == "statistics" {
					_, _ = w.Write([]byte(`{"items":[` +
						`{"id":"vid-low","statistics":{"viewCount":"100"}},` +
						`{"id":"vid-high","statistics":{"viewCount":"4000"}}` +
						`]}`))
				} else {
					_, _ = w.Write([]byte(`{"items":[` +
						`{"id":{"videoId":"vid-low"},"snippet":{"publishedAt":"2024-04-25T14:00:00Z","channelId":"c1","title":"Low","description":"","channelTitle":"Ch1"}},` +
						`{"id":{"videoId":"vid-high"},"snippet":{"publishedAt":"2024-04-25T13:00:00Z","channelId":"c2","title":"High","description":"","channelTitle":"Ch2"}}` +
						`]}`))
				}
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 2)
				lowScore := news.ScoreOf(news.SourceYouTube, news.TagVideo, 100, true)
				highScore := news.ScoreOf(news.SourceYouTube, news.TagVideo, 4000, true)
				assert.Greater(t, highScore, lowScore)
				assert.Equal(t, lowScore, items[0].Score)
				assert.Equal(t, highScore, items[1].Score)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			searchURL := "http://unused"
			statsURL := "http://unused-stats"
			if test.stub != nil {
				s := httptest.NewServer(test.stub)
				defer s.Close()
				searchURL = s.URL + "?part=snippet"
				statsURL = s.URL + "?part=statistics"
			}
			got, err := YouTube{url: searchURL, videosURL: statsURL, key: test.key}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}
