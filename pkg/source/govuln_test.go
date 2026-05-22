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
	"encoding/json"
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

// goVulnTestWindow is wide enough that the static fixture timestamps (2024)
// are never dropped by the recency filter, regardless of when the test runs.
const goVulnTestWindow = 1000000 * time.Hour

// goVulnWithdrawnDetail is an OSV entry with a withdrawn timestamp — used to
// verify that ShouldInclude drops it.
const goVulnWithdrawnDetail = `{
  "id": "GO-2024-0001",
  "published": "2024-01-01T00:00:00Z",
  "modified": "2024-01-02T00:00:00Z",
  "summary": "Withdrawn advisory",
  "details": "This entry was withdrawn.",
  "withdrawn": "2024-01-03T00:00:00Z",
  "affected": []
}`

// goVulnNoSummaryDetail is an OSV entry missing a summary — should be filtered.
const goVulnNoSummaryDetail = `{
  "id": "GO-2024-0002",
  "published": "2024-02-01T00:00:00Z",
  "modified": "2024-02-02T00:00:00Z",
  "summary": "",
  "details": "Details without a summary.",
  "affected": []
}`

func TestGoVuln_Fetch(t *testing.T) {
	t.Parallel()

	indexFixture, err := os.ReadFile("testdata/govuln.json")
	require.NoError(t, err)

	detailFixture, err := os.ReadFile("testdata/govuln_detail.json")
	require.NoError(t, err)

	tt := map[string]struct {
		indexBody  []byte
		detailBody []byte
		indexCode  int
		detailCode int
		want       func(t *testing.T, items []news.Item, err error)
	}{
		"Bad Request": {
			indexCode: http.StatusBadRequest,
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			indexCode:  http.StatusOK,
			indexBody:  indexFixture,
			detailCode: http.StatusOK,
			detailBody: detailFixture,
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				// Index has 2 entries; both detail fetches succeed with the same fixture.
				require.Len(t, items, 2)

				var fixture vulnEntry
				require.NoError(t, json.Unmarshal(detailFixture, &fixture))

				assert.Equal(t, news.Item{
					Source:    news.SourceGoVuln,
					Title:     fixture.Summary,
					URL:       "https://pkg.go.dev/vuln/" + fixture.ID,
					Snippet:   fixture.Details,
					Tag:       news.TagSecurity,
					Score:     1.0, // SourceWeight 2.0 * constantNoSignal 0.5
					Published: fixture.Published,
				}, items[0])
			},
		},
		"Withdrawn": {
			indexCode:  http.StatusOK,
			indexBody:  []byte(`[{"id":"GO-2024-0001","modified":"2024-01-02T00:00:00Z"}]`),
			detailCode: http.StatusOK,
			detailBody: []byte(goVulnWithdrawnDetail),
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.NoError(t, err)
				assert.Empty(t, items)
			},
		},
		"Empty Summary": {
			indexCode:  http.StatusOK,
			indexBody:  []byte(`[{"id":"GO-2024-0002","modified":"2024-02-02T00:00:00Z"}]`),
			detailCode: http.StatusOK,
			detailBody: []byte(goVulnNoSummaryDetail),
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.NoError(t, err)
				assert.Empty(t, items)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/ID/") {
					w.WriteHeader(test.detailCode)
					if test.detailBody != nil {
						_, _ = w.Write(test.detailBody)
					}
					return
				}
				w.WriteHeader(test.indexCode)
				if test.indexBody != nil {
					_, _ = w.Write(test.indexBody)
				}
			}))
			defer s.Close()

			got, gotErr := GoVuln{
				indexURL:   s.URL + "/index/vulns.json",
				detailBase: s.URL + "/ID/",
				window:     goVulnTestWindow,
				limit:      10,
			}.Fetch(t.Context())
			test.want(t, got, gotErr)
		})
	}
}

func TestGoVuln_DetailErrorSkipped(t *testing.T) {
	t.Parallel()

	// Index has two entries; the detail server always returns 500.
	// Fetch should return an empty slice without error (errors are per-entry).
	indexBody := []byte(`[
		{"id":"GO-2024-3109","modified":"2024-12-01T10:00:00Z"},
		{"id":"GO-2024-3099","modified":"2024-11-28T08:30:00Z"}
	]`)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/ID/") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexBody)
	}))
	defer s.Close()

	got, err := GoVuln{
		indexURL:   s.URL + "/index/vulns.json",
		detailBase: s.URL + "/ID/",
		window:     goVulnTestWindow,
		limit:      10,
	}.Fetch(t.Context())
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestGoVuln_WindowFiltersOldEntries(t *testing.T) {
	t.Parallel()

	// Index has one entry modified 30 days ago; with a 7-day window it should
	// be filtered before any detail request is issued.
	old := time.Now().Add(-30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	indexBody := []byte(`[{"id":"GO-2024-OLD","modified":"` + old + `"}]`)

	detailCalled := false
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/ID/") {
			detailCalled = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexBody)
	}))
	defer s.Close()

	got, err := GoVuln{
		indexURL:   s.URL + "/index/vulns.json",
		detailBase: s.URL + "/ID/",
		window:     7 * 24 * time.Hour,
		limit:      10,
	}.Fetch(t.Context())
	assert.NoError(t, err)
	assert.Empty(t, got)
	assert.False(t, detailCalled, "detail endpoint must not be hit for out-of-window entries")
}
