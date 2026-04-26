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
	"strings"
	"testing"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHub_Fetch(t *testing.T) {
	t.Parallel()

	issueJSON := func(url, milestoneJSON string) string {
		return `[{"title":"Test Proposal","html_url":"` + url + `","body":"Some body text","user":{"login":"gopher"},"comments":5,"reactions":{"+1":10},"created_at":"2024-01-01T00:00:00Z","milestone":` + milestoneJSON + `}]`
	}

	tt := map[string]struct {
		setup func() ([]ghEndpoint, func())
		want  func([]news.Item, error)
	}{
		"Bad Request": {
			setup: func() ([]ghEndpoint, func()) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				}))
				return []ghEndpoint{{url: s.URL, tag: news.TagProposal}}, s.Close
			},
			want: func(items []news.Item, err error) {
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"Version milestone in snippet": {
			setup: func() ([]ghEndpoint, func()) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(issueJSON("https://github.com/golang/go/issues/1", `{"title":"Go1.27"}`)))
				}))
				return []ghEndpoint{{url: s.URL, tag: news.TagProposalAccepted}}, s.Close
			},
			want: func(items []news.Item, err error) {
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.True(t, strings.HasPrefix(items[0].Snippet, "Targeting Go 1.27 \u2014 "), "snippet: %q", items[0].Snippet)
				assert.Equal(t, news.TagProposalAccepted, items[0].Tag)
				assert.Equal(t, 10, items[0].Score)
				assert.Equal(t, 5, items[0].Comments)
				assert.Equal(t, "gopher", items[0].Author)
			},
		},
		"Backlog milestone omitted from snippet": {
			setup: func() ([]ghEndpoint, func()) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(issueJSON("https://github.com/golang/go/issues/2", `{"title":"Backlog"}`)))
				}))
				return []ghEndpoint{{url: s.URL, tag: news.TagProposal}}, s.Close
			},
			want: func(items []news.Item, err error) {
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.False(t, strings.Contains(items[0].Snippet, "Targeting"), "snippet should have no targeting prefix: %q", items[0].Snippet)
			},
		},
		"No milestone omitted from snippet": {
			setup: func() ([]ghEndpoint, func()) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(issueJSON("https://github.com/golang/go/issues/3", "null")))
				}))
				return []ghEndpoint{{url: s.URL, tag: news.TagProposal}}, s.Close
			},
			want: func(items []news.Item, err error) {
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.False(t, strings.Contains(items[0].Snippet, "Targeting"), "snippet should have no targeting prefix: %q", items[0].Snippet)
			},
		},
		"Deduplication across endpoints": {
			setup: func() ([]ghEndpoint, func()) {
				// Both endpoints return the same html_url — it should appear only once.
				const sharedURL = "https://github.com/golang/go/issues/99"
				body := issueJSON(sharedURL, "null")
				s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(body))
				}))
				s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(body))
				}))
				eps := []ghEndpoint{
					{url: s1.URL, tag: news.TagProposalAccepted},
					{url: s2.URL, tag: news.TagProposal},
				}
				return eps, func() { s1.Close(); s2.Close() }
			},
			want: func(items []news.Item, err error) {
				require.NoError(t, err)
				require.Len(t, items, 1)
				// First endpoint's tag wins.
				assert.Equal(t, news.TagProposalAccepted, items[0].Tag)
			},
		},
		"Auth header sent when token set": {
			setup: func() ([]ghEndpoint, func()) {
				var gotAuth string
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					gotAuth = r.Header.Get("Authorization")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("[]"))
				}))
				// Inject a verification closure by capturing gotAuth via a custom endpoint.
				// We abuse the want func below to check after the fact.
				_ = gotAuth
				t.Cleanup(func() {
					assert.Equal(t, "Bearer test-token", gotAuth)
				})
				return []ghEndpoint{{url: s.URL, tag: news.TagProposal}}, s.Close
			},
			want: func(items []news.Item, err error) {
				require.NoError(t, err)
				assert.Empty(t, items)
			},
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			eps, cleanup := tc.setup()
			defer cleanup()

			token := ""
			if name == "Auth header sent when token set" {
				token = "test-token"
			}

			got, err := GitHub{endpoints: eps, token: token}.Fetch(t.Context())
			tc.want(got, err)
		})
	}
}

func TestGhSnippet(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		body      string
		milestone *ghMilestone
		wantPfx   string
	}{
		"version milestone": {body: "body", milestone: &ghMilestone{"Go1.27"}, wantPfx: "Targeting Go 1.27 \u2014 "},
		"patch version":     {body: "body", milestone: &ghMilestone{"Go1.27.1"}, wantPfx: "Targeting Go 1.27.1 \u2014 "},
		"backlog":           {body: "body", milestone: &ghMilestone{"Backlog"}, wantPfx: "body"},
		"nil milestone":     {body: "body", milestone: nil, wantPfx: "body"},
		"markdown stripped": {body: "## Title\n**bold** `code`", milestone: nil, wantPfx: "Title bold code"},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := ghSnippet(tc.body, tc.milestone)
			assert.True(t, strings.HasPrefix(got, tc.wantPfx), "ghSnippet(%q, %v) = %q, want prefix %q", tc.body, tc.milestone, got, tc.wantPfx)
		})
	}
}
