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

// wwrFixture covers a Go job named in the title (kept) and a role whose
// description merely contains the English word "go" (dropped — only the title
// is matched, to avoid prose false positives).
const wwrFixture = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Acme Corp: Senior Go Engineer</title>
      <link>https://weworkremotely.com/jobs/1</link>
      <description>We need a Go engineer. Salary $120k.</description>
      <pubDate>Mon, 30 Dec 2024 10:00:00 +0000</pubDate>
    </item>
    <item>
      <title>Beta Ltd: Platform Product Manager</title>
      <link>https://weworkremotely.com/jobs/2</link>
      <description>Ready to go further with us and grow your career.</description>
      <pubDate>Mon, 30 Dec 2024 10:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

func TestWeWorkRemotely_Fetch(t *testing.T) {
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
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(wwrFixture))
				require.NoError(t, err)
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				// Only the title-matched Go role survives; the "go further"
				// prose listing is dropped.
				require.Len(t, items, 1)

				goEng := items[0]
				assert.Equal(t, news.SourceWeWorkRemotely, goEng.Source)
				assert.Equal(t, news.TagJobs, goEng.Tag)
				// "Company: Role" split into "Company · Role".
				assert.Equal(t, "Acme Corp · Senior Go Engineer", goEng.Title)
				assert.Equal(t, "https://weworkremotely.com/jobs/1", goEng.URL)
				require.NotNil(t, goEng.Author)
				assert.Equal(t, "Acme Corp", goEng.Author.Name)
				assert.Equal(t, fixedNow(), goEng.Published)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			got, err := WeWorkRemotely{url: s.URL, now: fixedNow}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestWeWorkRemotely_CompanyRole(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		title       string
		wantCompany string
		wantRole    string
	}{
		"Standard convention": {title: "Acme: Go Engineer", wantCompany: "Acme", wantRole: "Go Engineer"},
		"Extra spacing":       {title: "  Acme :  Go Engineer ", wantCompany: "Acme", wantRole: "Go Engineer"},
		"No colon":            {title: "Go Engineer", wantCompany: "", wantRole: "Go Engineer"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			company, role := wwrCompanyRole(test.title)
			assert.Equal(t, test.wantCompany, company)
			assert.Equal(t, test.wantRole, role)
		})
	}
}
