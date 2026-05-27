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

func TestFreeCodeCamp_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/freecodecamp.xml")
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
				require.Len(t, items, 2)
				assert.Equal(t, news.Item{
					Source:    news.SourceFreeCodeCamp,
					Title:     "How to Build a REST API in Go",
					URL:       "https://www.freecodecamp.org/news/how-to-build-a-rest-api-in-go/",
					Author:    &news.Author{Name: "Quincy Larson"},
					Snippet:   "A practical walkthrough of building a REST API in Go using the standard library.",
					Tag:       news.TagTutorial,
					Score:     0.5,
					Published: time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
				}, items[0])
				assert.Equal(t, "Goroutines Explained", items[1].Title)
				assert.Equal(t, news.TagTutorial, items[1].Tag)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			got, err := FreeCodeCamp{url: s.URL}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}
