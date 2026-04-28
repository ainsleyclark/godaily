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

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const awesomeGoMergeResponse = `[
  {
    "sha": "deadbeef",
    "html_url": "https://github.com/avelino/awesome-go/commit/deadbeef",
    "commit": {
      "message": "Merge pull request #9999 from someuser/branch",
      "author": {"name": "Bot", "date": "2026-04-20T00:00:00Z"}
    }
  },
  {
    "sha": "feed1234",
    "html_url": "https://github.com/avelino/awesome-go/commit/feed1234",
    "commit": {
      "message": "Merge branch 'main' of github.com:avelino/awesome-go",
      "author": {"name": "Bot", "date": "2026-04-20T00:00:00Z"}
    }
  }
]`

func TestAwesomeGo_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/awesomego.json")
	require.NoError(t, err)

	tt := map[string]struct {
		body []byte
		want func(t *testing.T, items []news.Item, err error)
	}{
		"Bad Request": {
			body: nil,
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			body: fixture,
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 3)
				assert.Equal(t, news.Item{
					Source:    news.SourceAwesomeGo,
					Title:     "Add lynxdb to databases (#6282)",
					URL:       "https://github.com/avelino/awesome-go/commit/05a987517e34d76afbd22d3460f7c5359fbee3b1",
					Author:    "Evgenii Orlov",
					Tag:       news.TagArticle,
					Score:     0.5, // weight 1.0 * constantNoSignal 0.5
					Published: time.Date(2026, time.April, 27, 8, 30, 23, 0, time.UTC),
				}, items[0])
				// Multi-line commit body becomes the snippet on item 3.
				assert.NotEmpty(t, items[2].Snippet)
			},
		},
		"Merge Commits Dropped": {
			body: []byte(awesomeGoMergeResponse),
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, items)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if test.body == nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(test.body)
			}))
			defer s.Close()

			got, err := AwesomeGo{url: s.URL}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestAwesomeGo_AuthHeader(t *testing.T) {
	t.Parallel()

	var gotAuth string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer s.Close()

	_, err := AwesomeGo{url: s.URL, token: "test-token"}.Fetch(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Bearer test-token", gotAuth)
}

func TestSplitCommitMessage(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in        string
		wantTitle string
		wantBody  string
	}{
		"single line":  {in: "Add foo (#1)", wantTitle: "Add foo (#1)", wantBody: ""},
		"with body":    {in: "Subject\n\nbody line one\nbody line two", wantTitle: "Subject", wantBody: "body line one\nbody line two"},
		"trims spaces": {in: "  hello  \n  world  ", wantTitle: "hello", wantBody: "world"},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			gotTitle, gotBody := splitCommitMessage(tc.in)
			assert.Equal(t, tc.wantTitle, gotTitle)
			assert.Equal(t, tc.wantBody, gotBody)
		})
	}
}
