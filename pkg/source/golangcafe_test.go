// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// golangCafeFixture is a listing page carrying two JobPosting JSON-LD blocks
// (one with a salary range, one without) plus an unrelated WebSite block that
// must be ignored. The second posting is nested inside an ItemList to exercise
// the recursive walk.
const golangCafeFixture = `<!DOCTYPE html><html><head>
<script type="application/ld+json">
{"@context":"https://schema.org/","@type":"WebSite","name":"Golang Cafe","url":"https://golang.cafe"}
</script>
<script type="application/ld+json">
{"@context":"https://schema.org/","@type":"JobPosting","title":"Senior Go Engineer","datePosted":"2024-12-30","hiringOrganization":{"@type":"Organization","name":"Acme Corp"},"jobLocation":{"@type":"Place","address":{"@type":"PostalAddress","addressLocality":"London"}},"baseSalary":{"@type":"MonetaryAmount","currency":"GBP","value":{"@type":"QuantitativeValue","minValue":90000,"maxValue":120000,"unitText":"YEAR"}},"url":"https://golang.cafe/job/senior-go-engineer"}
</script>
<script type="application/ld+json">
{"@context":"https://schema.org/","@type":"ItemList","itemListElement":[{"@type":"ListItem","position":1,"item":{"@type":"JobPosting","title":"Backend Go Developer","datePosted":"2024-12-30","hiringOrganization":{"@type":"Organization","name":"Beta Ltd"},"jobLocationType":"TELECOMMUTE","applicantLocationRequirements":{"@type":"Country","name":"Worldwide"},"url":"https://golang.cafe/job/backend-go-developer"}}]}
</script>
</head><body></body></html>`

func TestGolangCafe_Fetch(t *testing.T) {
	t.Parallel()

	fixedNow := func() time.Time {
		return time.Date(2024, time.December, 31, 0, 0, 0, 0, time.UTC)
	}

	tt := map[string]struct {
		stub http.HandlerFunc
		want func(t *testing.T, items []news.Item, err error)
	}{
		"Bad Request": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(golangCafeFixture))
				require.NoError(t, err)
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				// Both JobPostings parsed; the WebSite block ignored.
				require.Len(t, items, 2)

				byTitle := map[string]news.Item{}
				for _, it := range items {
					assert.Equal(t, news.SourceGolangCafe, it.Source)
					assert.Equal(t, news.TagJobs, it.Tag)
					assert.Equal(t, fixedNow(), it.Published)
					byTitle[it.Title] = it
				}

				senior, ok := byTitle["Acme Corp · Senior Go Engineer · London"]
				require.True(t, ok, "salaried posting should be present")
				assert.Equal(t, "https://golang.cafe/job/senior-go-engineer", senior.URL)
				require.NotNil(t, senior.Author)
				assert.Equal(t, "Acme Corp", senior.Author.Name)
				assert.Equal(t, "£90k–£120k", senior.Snippet)

				// Nested-in-ItemList posting; remote via applicantLocationRequirements.
				backend, ok := byTitle["Beta Ltd · Backend Go Developer · Remote"]
				require.True(t, ok, "ItemList-nested posting should be present")
				assert.Empty(t, backend.Snippet) // no salary disclosed
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			got, err := GolangCafe{url: s.URL, now: fixedNow}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestGolangCafe_JSONLDSalary(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		base any
		want string
	}{
		"USD range": {
			base: map[string]any{"currency": "USD", "value": map[string]any{"minValue": 120000.0, "maxValue": 160000.0}},
			want: "$120k–$160k",
		},
		"EUR min only": {
			base: map[string]any{"currency": "EUR", "value": map[string]any{"minValue": 80000.0}},
			want: "€80k+",
		},
		"String numbers": {
			base: map[string]any{"currency": "GBP", "value": map[string]any{"minValue": "90,000", "maxValue": "120,000"}},
			want: "£90k–£120k",
		},
		"No value": {
			base: map[string]any{"currency": "USD"},
			want: "",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, jsonLDSalary(test.base))
		})
	}
}
