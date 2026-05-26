// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanetGolang_Fetch(t *testing.T) {
	t.Parallel()

	// Real-format fixture captured from planetgolang.dev/index.xml — each
	// item's <link> is replaced with __SERVER_URL__ so enrichment requests
	// land on the test server rather than the live internet.
	fixture, err := os.ReadFile("testdata/planet_golang.xml")
	require.NoError(t, err)

	tt := map[string]struct {
		stub func(serverURL string) http.HandlerFunc
		want func(t *testing.T, items []news.Item, err error, serverURL string)
	}{
		"Bad Request": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			stub: func(serverURL string) http.HandlerFunc {
				body := strings.ReplaceAll(string(fixture), "__SERVER_URL__", serverURL)
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(body))
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, serverURL string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 2)
				assert.Equal(t, news.Item{
					Source:    news.SourcePlanetGolang,
					Title:     "Understanding Go's Memory Model",
					URL:       serverURL,
					Author:    &news.Author{Name: "Alex Writer"},
					Snippet:   "A deep dive into how Go manages memory and goroutine synchronisation.",
					Tag:       news.TagArticle,
					Score:     0.5, // no signal: weight 1.0 * constant 0.5
					Published: time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC),
				}, items[0])
				// Second entry has no <author>; Author should be nil, not an empty struct.
				assert.Nil(t, items[1].Author)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var serverURL string
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				test.stub(serverURL)(w, r)
			}))
			defer s.Close()
			serverURL = s.URL

			got, err := PlanetGolang{url: s.URL}.Fetch(t.Context())
			test.want(t, got, err, s.URL)
		})
	}
}
