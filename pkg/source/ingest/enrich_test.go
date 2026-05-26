// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ingest

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchPage(t *testing.T) {
	t.Parallel()

	const ogPage = `<html><head>
<meta property="og:description" content="OG wins">
<meta name="twitter:description" content="twitter loses">
<meta name="description" content="standard loses">
<meta property="og:image" content="https://cdn.example/img.jpg">
</head><body></body></html>`

	tt := map[string]struct {
		stub     http.HandlerFunc
		wantDesc string
		wantImg  string
		wantErr  bool
	}{
		"OG Tags Found": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, err := w.Write([]byte(ogPage))
				assert.NoError(t, err)
			},
			wantDesc: "OG wins",
			wantImg:  "https://cdn.example/img.jpg",
		},
		"Non-HTML Content-Type": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/pdf")
				_, err := w.Write([]byte("%PDF-1.4"))
				assert.NoError(t, err)
			},
			wantErr: true,
		},
		"Non-2xx": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			doc, base, err := fetchPage(t.Context(), s.URL)
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, doc)
			require.NotNil(t, base)
			assert.Equal(t, test.wantDesc, extractMeta(doc, metaDescriptionSelectors))
			assert.Equal(t, test.wantImg, extractMeta(doc, metaImageSelectors))
		})
	}
}

func TestFetchPage_UserAgent(t *testing.T) {
	t.Parallel()

	var gotUA string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/html")
		_, err := w.Write([]byte(`<html><head><meta name="description" content="x"></head></html>`))
		assert.NoError(t, err)
	}))
	defer s.Close()

	_, _, err := fetchPage(t.Context(), s.URL)
	require.NoError(t, err)
	assert.Equal(t, enrichUserAgent, gotUA)
}

func TestExtractMeta_DescriptionPriority(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		body string
		want string
	}{
		"OG Wins": {
			body: `<html><head>
<meta property="og:description" content="OG wins">
<meta name="twitter:description" content="twitter loses">
<meta name="description" content="standard loses">
</head></html>`,
			want: "OG wins",
		},
		"Twitter Fallback": {
			body: `<html><head>
<meta name="twitter:description" content="twitter wins">
<meta name="description" content="standard loses">
</head></html>`,
			want: "twitter wins",
		},
		"Standard Fallback": {
			body: `<html><head><meta name="description" content="standard wins"></head></html>`,
			want: "standard wins",
		},
		"Empty OG Skipped": {
			body: `<html><head>
<meta property="og:description" content="">
<meta name="description" content="standard wins">
</head></html>`,
			want: "standard wins",
		},
		"None": {
			body: `<html><head><title>x</title></head></html>`,
			want: "",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(test.body))
				assert.NoError(t, err)
			}))
			defer s.Close()

			doc, _, err := fetchPage(t.Context(), s.URL)
			require.NoError(t, err)
			assert.Equal(t, test.want, extractMeta(doc, metaDescriptionSelectors))
		})
	}
}

func TestExtractMeta_ImagePriority(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		body string
		want string
	}{
		"Secure URL Wins": {
			body: `<html><head>
<meta property="og:image:secure_url" content="https://secure.example/img.jpg">
<meta property="og:image" content="https://example/img.jpg">
<meta name="twitter:image" content="https://twitter.example/img.jpg">
</head></html>`,
			want: "https://secure.example/img.jpg",
		},
		"OG Image Fallback": {
			body: `<html><head>
<meta property="og:image" content="https://example/img.jpg">
<meta name="twitter:image" content="https://twitter.example/img.jpg">
</head></html>`,
			want: "https://example/img.jpg",
		},
		"Twitter Fallback": {
			body: `<html><head><meta name="twitter:image" content="https://twitter.example/img.jpg"></head></html>`,
			want: "https://twitter.example/img.jpg",
		},
		"None": {
			body: `<html><head></head></html>`,
			want: "",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(test.body))
				assert.NoError(t, err)
			}))
			defer s.Close()

			doc, _, err := fetchPage(t.Context(), s.URL)
			require.NoError(t, err)
			assert.Equal(t, test.want, extractMeta(doc, metaImageSelectors))
		})
	}
}

func TestResolveImageURL(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		base string
		raw  string
		want string
	}{
		"Absolute HTTPS": {base: "https://example.com/post", raw: "https://cdn.example/img.jpg", want: "https://cdn.example/img.jpg"},
		"Absolute HTTP":  {base: "https://example.com/post", raw: "http://cdn.example/img.jpg", want: "http://cdn.example/img.jpg"},
		"Relative Path":  {base: "https://example.com/articles/post", raw: "/images/hero.jpg", want: "https://example.com/images/hero.jpg"},
		"Relative Same":  {base: "https://example.com/articles/post", raw: "hero.jpg", want: "https://example.com/articles/hero.jpg"},
		"Data Scheme":    {base: "https://example.com/post", raw: "data:image/png;base64,iVBOR", want: ""},
		"FTP Scheme":     {base: "https://example.com/post", raw: "ftp://example.com/x.jpg", want: ""},
		"Unparseable":    {base: "https://example.com/post", raw: "://broken", want: ""},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			base, err := url.Parse(test.base)
			require.NoError(t, err)
			assert.Equal(t, test.want, resolveImageURL(base, test.raw))
		})
	}
}

func TestEnrich(t *testing.T) {
	t.Parallel()

	t.Run("Fills Empty Snippet And ImageURL", func(t *testing.T) {
		t.Parallel()
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, err := w.Write([]byte(`<html><head>
<meta property="og:description" content="enriched body">
<meta property="og:image" content="https://cdn.example/img.jpg">
</head></html>`))
			assert.NoError(t, err)
		}))
		defer s.Close()

		items := []news.Item{{Title: "A"}, {Title: "B"}}
		enrich(t.Context(), []enrichTarget{
			{URL: s.URL, Item: &items[0]},
			{URL: s.URL, Item: &items[1]},
		})
		assert.Equal(t, "enriched body", items[0].Snippet)
		assert.Equal(t, "https://cdn.example/img.jpg", items[0].ImageURL)
		assert.Equal(t, "enriched body", items[1].Snippet)
		assert.Equal(t, "https://cdn.example/img.jpg", items[1].ImageURL)
	})

	t.Run("Preserves Existing Fields", func(t *testing.T) {
		t.Parallel()
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, err := w.Write([]byte(`<html><head>
<meta property="og:description" content="should not overwrite">
<meta property="og:image" content="https://cdn.example/new.jpg">
</head></html>`))
			assert.NoError(t, err)
		}))
		defer s.Close()

		item := news.Item{Snippet: "kept", ImageURL: ""}
		enrich(t.Context(), []enrichTarget{{URL: s.URL, Item: &item}})
		assert.Equal(t, "kept", item.Snippet, "existing snippet must not be overwritten")
		assert.Equal(t, "https://cdn.example/new.jpg", item.ImageURL)
	})

	t.Run("Skips When Both Fields Set", func(t *testing.T) {
		t.Parallel()
		var hits int32
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt32(&hits, 1)
		}))
		defer s.Close()

		item := news.Item{Snippet: "set", ImageURL: "https://x/y.jpg"}
		enrich(t.Context(), []enrichTarget{{URL: s.URL, Item: &item}})
		assert.Equal(t, int32(0), atomic.LoadInt32(&hits))
	})

	t.Run("Single Fetch For Both Fields", func(t *testing.T) {
		t.Parallel()
		var hits int32
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt32(&hits, 1)
			w.Header().Set("Content-Type", "text/html")
			_, err := w.Write([]byte(`<html><head>
<meta property="og:description" content="d">
<meta property="og:image" content="https://cdn.example/img.jpg">
</head></html>`))
			assert.NoError(t, err)
		}))
		defer s.Close()

		item := news.Item{}
		enrich(t.Context(), []enrichTarget{{URL: s.URL, Item: &item}})
		assert.Equal(t, int32(1), atomic.LoadInt32(&hits))
		assert.Equal(t, "d", item.Snippet)
		assert.Equal(t, "https://cdn.example/img.jpg", item.ImageURL)
	})

	t.Run("Resolves Relative Image URL", func(t *testing.T) {
		t.Parallel()
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, err := w.Write([]byte(`<html><head><meta property="og:image" content="/hero.jpg"></head></html>`))
			assert.NoError(t, err)
		}))
		defer s.Close()

		item := news.Item{}
		enrich(t.Context(), []enrichTarget{{URL: s.URL, Item: &item}})
		assert.Equal(t, s.URL+"/hero.jpg", item.ImageURL)
	})

	t.Run("Rejects Data URI Image", func(t *testing.T) {
		t.Parallel()
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, err := w.Write([]byte(`<html><head><meta property="og:image" content="data:image/png;base64,xxx"></head></html>`))
			assert.NoError(t, err)
		}))
		defer s.Close()

		item := news.Item{}
		enrich(t.Context(), []enrichTarget{{URL: s.URL, Item: &item}})
		assert.Empty(t, item.ImageURL)
	})

	t.Run("Tolerates Errors", func(t *testing.T) {
		t.Parallel()
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer s.Close()

		item := news.Item{}
		enrich(t.Context(), []enrichTarget{{URL: s.URL, Item: &item}})
		assert.Empty(t, item.Snippet)
		assert.Empty(t, item.ImageURL)
	})

	t.Run("Truncates Long Description", func(t *testing.T) {
		t.Parallel()
		long := strings.Repeat("x", maxSnippetLen+50)
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, err := w.Write([]byte(`<html><head><meta property="og:description" content="` + long + `"></head></html>`))
			assert.NoError(t, err)
		}))
		defer s.Close()

		item := news.Item{}
		enrich(t.Context(), []enrichTarget{{URL: s.URL, Item: &item}})
		assert.Len(t, item.Snippet, maxSnippetLen)
	})

	t.Run("Empty Slice", func(t *testing.T) {
		t.Parallel()
		enrich(t.Context(), nil)
	})
}
