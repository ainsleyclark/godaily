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

package ingest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldEnrich(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input string
		want  bool
	}{
		"Empty":             {input: "", want: false},
		"HN Permalink":      {input: "https://news.ycombinator.com/item?id=1", want: false},
		"Reddit Self Post":  {input: "https://www.reddit.com/r/golang/comments/abc/foo/", want: false},
		"Reddit Mixed Case": {input: "https://www.Reddit.com/R/Golang/comments/abc/", want: false},
		"External Article":  {input: "https://example.com/post", want: true},
		"GitHub Repo":       {input: "https://github.com/foo/bar", want: true},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, shouldEnrich(test.input))
		})
	}
}

func TestFetchMetaDescription(t *testing.T) {
	t.Parallel()

	const ogPage = `<html><head>
<meta property="og:description" content="OG wins">
<meta name="twitter:description" content="twitter loses">
<meta name="description" content="standard loses">
</head><body></body></html>`

	const twitterOnlyPage = `<html><head>
<meta name="twitter:description" content="twitter wins">
<meta name="description" content="standard loses">
</head></html>`

	const stdOnlyPage = `<html><head>
<meta name="description" content="standard wins">
</head></html>`

	const emptyOGPage = `<html><head>
<meta property="og:description" content="">
<meta name="description" content="standard wins">
</head></html>`

	const noMetaPage = `<html><head><title>Nothing</title></head><body><p>hi</p></body></html>`

	tt := map[string]struct {
		stub        http.HandlerFunc
		wantContent string
		wantErr     bool
	}{
		"OG Wins": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, err := w.Write([]byte(ogPage))
				assert.NoError(t, err)
			},
			wantContent: "OG wins",
		},
		"Twitter Fallback": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(twitterOnlyPage))
				assert.NoError(t, err)
			},
			wantContent: "twitter wins",
		},
		"Standard Fallback": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(stdOnlyPage))
				assert.NoError(t, err)
			},
			wantContent: "standard wins",
		},
		"Empty OG Skipped": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(emptyOGPage))
				assert.NoError(t, err)
			},
			wantContent: "standard wins",
		},
		"No Meta": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(noMetaPage))
				assert.NoError(t, err)
			},
			wantContent: "",
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

			got, err := fetchMetaDescription(t.Context(), s.URL)
			assert.Equal(t, test.wantErr, err != nil)
			assert.Equal(t, test.wantContent, got)
		})
	}
}

func TestFetchMetaDescription_UserAgent(t *testing.T) {
	t.Parallel()

	var gotUA string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/html")
		_, err := w.Write([]byte(`<html><head><meta name="description" content="x"></head></html>`))
		assert.NoError(t, err)
	}))
	defer s.Close()

	_, err := fetchMetaDescription(t.Context(), s.URL)
	require.NoError(t, err)
	assert.Equal(t, enrichUserAgent, gotUA)
}

func TestEnrichSnippets(t *testing.T) {
	t.Parallel()

	t.Run("Fills Empty Snippets", func(t *testing.T) {
		t.Parallel()
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, err := w.Write([]byte(`<html><head><meta property="og:description" content="enriched body"></head></html>`))
			assert.NoError(t, err)
		}))
		defer s.Close()

		items := []news.Item{
			{Title: "A", URL: s.URL, Snippet: ""},
			{Title: "B", URL: s.URL, Snippet: ""},
		}
		EnrichSnippets(t.Context(), items)
		assert.Equal(t, "enriched body", items[0].Snippet)
		assert.Equal(t, "enriched body", items[1].Snippet)
	})

	t.Run("Preserves Existing Snippet", func(t *testing.T) {
		t.Parallel()
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Errorf("server should not be called when snippet is already set")
		}))
		defer s.Close()

		items := []news.Item{{URL: s.URL, Snippet: "already populated"}}
		EnrichSnippets(t.Context(), items)
		assert.Equal(t, "already populated", items[0].Snippet)
	})

	t.Run("Skips Discussion URLs", func(t *testing.T) {
		t.Parallel()
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Errorf("server should not be called for skipped URL")
		}))
		defer s.Close()

		items := []news.Item{
			{URL: "https://news.ycombinator.com/item?id=1"},
			{URL: "https://www.reddit.com/r/golang/comments/x/y/"},
		}
		EnrichSnippets(t.Context(), items)
		assert.Empty(t, items[0].Snippet)
		assert.Empty(t, items[1].Snippet)
	})

	t.Run("Tolerates Errors", func(t *testing.T) {
		t.Parallel()
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer s.Close()

		items := []news.Item{{URL: s.URL}}
		EnrichSnippets(t.Context(), items)
		assert.Empty(t, items[0].Snippet)
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

		items := []news.Item{{URL: s.URL}}
		EnrichSnippets(t.Context(), items)
		assert.Len(t, items[0].Snippet, maxSnippetLen)
	})

	t.Run("Empty Slice", func(t *testing.T) {
		t.Parallel()
		EnrichSnippets(t.Context(), nil)
	})
}
