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

func TestJetBrains_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/jetbrains.xml")
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
					_, _ = w.Write([]byte(body))
				}
			},
			want: func(t *testing.T, items []news.Item, err error, serverURL string) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 2)
				assert.Equal(t, news.Item{
					Source:    news.SourceJetBrains,
					Title:     "Code like a PIRATE with Junie and GoLand",
					URL:       serverURL,
					Author:    &news.Author{Name: "Anna Protsenko"},
					Snippet:   "This is a guest post from John Arundel, a Go writer and teacher who runs a free email course for Go learners.",
					Tag:       news.TagArticle,
					Score:     0.5, // weight 1.0 * constantNoSignal 0.5
					Published: time.Date(2026, time.April, 2, 13, 34, 24, 0, time.UTC),
				}, items[0])
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

			got, err := JetBrains{url: s.URL}.Fetch(t.Context())
			test.want(t, got, err, s.URL)
		})
	}
}

func TestJetBrains_UserAgent(t *testing.T) {
	t.Parallel()

	var ua string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<?xml version="1.0"?><rss><channel></channel></rss>`))
	}))
	defer s.Close()

	_, err := JetBrains{url: s.URL}.Fetch(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "godaily/1.0", ua)
}
