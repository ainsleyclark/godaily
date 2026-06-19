// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoRelease_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/gorelease.json")
	require.NoError(t, err)

	tt := map[string]struct {
		stub func() http.HandlerFunc
		want func(t *testing.T, items []news.Item, err error)
	}{
		"Bad Request": {
			stub: func() http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				}
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			stub: func() http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write(fixture)
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				// Fixture contains 3 entries: 1 stable, 1 unstable RC, 1 stable.
				// Pre-releases are now included, so all 3 survive.
				require.Len(t, items, 3)
				assert.Equal(t, news.Item{
					Source:  news.SourceGoRelease,
					Title:   "Go 1.26.2 released",
					URL:     "https://go.dev/doc/devel/release#go1.26.2",
					Snippet: "Stable Go release. See release notes for changes.",
					Tag:     news.TagRelease,
					Score:   1.0, // weight 2.0 * constantNoSignal 0.5
				}, items[0])
				assert.Equal(t, news.Item{
					Source:  news.SourceGoRelease,
					Title:   "Go 1.26 RC3 released",
					URL:     "https://go.dev/doc/devel/release#go1.26rc3",
					Snippet: "Go pre-release — try it in dev and prod, and file bugs.",
					Tag:     news.TagRelease,
					Score:   1.0,
				}, items[1])
				assert.Equal(t, "Go 1.25.9 released", items[2].Title)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub())
			defer s.Close()

			got, err := GoRelease{url: s.URL, limit: 5}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestGoRelease_LimitTrims(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/gorelease.json")
	require.NoError(t, err)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer s.Close()

	// limit=1 keeps only the first entry from the upstream slice (which is stable),
	// so the RC and the older stable release never reach Transform.
	got, err := GoRelease{url: s.URL, limit: 1}.Fetch(t.Context())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Go 1.26.2 released", got[0].Title)
}

func TestGoRelease_TitleAndSnippet(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		version     string
		stable      bool
		wantTitle   string
		wantSnippet string
	}{
		"Stable patch": {
			version:     "go1.26.2",
			stable:      true,
			wantTitle:   "Go 1.26.2 released",
			wantSnippet: "Stable Go release. See release notes for changes.",
		},
		"Release candidate": {
			version:     "go1.27rc1",
			stable:      false,
			wantTitle:   "Go 1.27 RC1 released",
			wantSnippet: "Go pre-release — try it in dev and prod, and file bugs.",
		},
		"Beta": {
			version:     "go1.27beta2",
			stable:      false,
			wantTitle:   "Go 1.27 Beta2 released",
			wantSnippet: "Go pre-release — try it in dev and prod, and file bugs.",
		},
		"Unstable without marker": {
			version:     "go1.27",
			stable:      false,
			wantTitle:   "Go 1.27 released",
			wantSnippet: "Go pre-release — try it in dev and prod, and file bugs.",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			title, snippet := goRelease{Version: test.version, Stable: test.stable}.titleAndSnippet()
			assert.Equal(t, test.wantTitle, title)
			assert.Equal(t, test.wantSnippet, snippet)
		})
	}
}

func TestGoRelease_DateLookup(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/gorelease.json")
	require.NoError(t, err)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer s.Close()

	want := time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)
	var gotURLs []string
	got, err := GoRelease{
		url:    s.URL,
		dlBase: "https://go.dev/dl/",
		limit:  5,
		dateFor: func(_ context.Context, fileURL string) time.Time {
			gotURLs = append(gotURLs, fileURL)
			return want
		},
	}.Fetch(t.Context())
	require.NoError(t, err)

	// Every release (stable and pre-release) gets the looked-up date.
	require.Len(t, got, 3)
	assert.Equal(t, want, got[0].Published)
	assert.Equal(t, want, got[1].Published)
	assert.Equal(t, want, got[2].Published)
	// All three input releases had their date queried.
	assert.Equal(t, []string{
		"https://go.dev/dl/go1.26.2.src.tar.gz",
		"https://go.dev/dl/go1.26rc3.src.tar.gz",
		"https://go.dev/dl/go1.25.9.src.tar.gz",
	}, gotURLs)
}
