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

func TestGoPodcast_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/gopodcast.xml")
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
				assert.Len(t, items, 1, "trailer episode should be filtered out")
				assert.Equal(t, news.Item{
					Source:    news.SourceGoPodcast,
					Title:     "082: Streaming, product updates, and marketing",
					URL:       "https://share.transistor.fm/s/e97475c0",
					Author:    &news.Author{Name: "Dominic St-Pierre"},
					ImageURL:  "https://img.transistorcdn.com/episode-082.jpg",
					Snippet:   "Hey we talk about streaming programming session, some updates on our products, and challenges related to marketing.",
					Tag:       news.TagPodcast,
					Score:     0.5, // weight 1.0 * constantNoSignal 0.5
					Published: time.Date(2026, time.April, 23, 9, 0, 0, 0, time.UTC),
				}, items[0])
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			got, err := GoPodcast{url: s.URL}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}
