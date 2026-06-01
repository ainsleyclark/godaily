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

const golangProjectsFixture = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Senior Golang Developer @ TRE ALTAMIRA Srl - Remote</title>
      <link>https://www.golangprojects.com/job/1</link>
      <description>Remote Go role, salary €90k.</description>
      <pubDate>Mon, 30 Dec 2024 10:00:00 +0000</pubDate>
    </item>
    <item>
      <title>Platform Engineer @ Delta</title>
      <link>https://www.golangprojects.com/job/2</link>
      <description>On-site platform work.</description>
      <pubDate>Mon, 30 Dec 2024 10:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

func TestGolangProjects_Fetch(t *testing.T) {
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
				_, err := w.Write([]byte(golangProjectsFixture))
				require.NoError(t, err)
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 2)

				dev := items[0]
				assert.Equal(t, news.SourceGolangProjects, dev.Source)
				assert.Equal(t, news.TagJobs, dev.Tag)
				// "Role @ Company - Location" → "Company · Role", suffix trimmed.
				assert.Equal(t, "TRE ALTAMIRA Srl · Senior Golang Developer", dev.Title)
				assert.Equal(t, "https://www.golangprojects.com/job/1", dev.URL)
				require.NotNil(t, dev.Author)
				assert.Equal(t, "TRE ALTAMIRA Srl", dev.Author.Name)
				assert.Equal(t, fixedNow(), dev.Published)

				platform := items[1]
				assert.Equal(t, "Delta · Platform Engineer", platform.Title)
				// Remote + salary disclosed listing outranks the on-site one.
				assert.Less(t, platform.Score, dev.Score)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			got, err := GolangProjects{url: s.URL, now: fixedNow}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestJobRoleAt(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		title       string
		wantCompany string
		wantRole    string
	}{
		"Role at company":       {title: "Go Developer at Acme", wantCompany: "Acme", wantRole: "Go Developer"},
		"At sign separator":     {title: "Go Developer @ Acme", wantCompany: "Acme", wantRole: "Go Developer"},
		"Non-breaking @ spaces": {title: "Senior Go Developer\u00a0@\u00a0Acme Srl", wantCompany: "Acme Srl", wantRole: "Senior Go Developer"},
		"Trailing dash suffix":  {title: "Go Developer @ Acme - Remote", wantCompany: "Acme", wantRole: "Go Developer"},
		"Trailing paren suffix": {title: "Go Developer at Acme (Berlin)", wantCompany: "Acme", wantRole: "Go Developer"},
		"No separator":          {title: "Go Developer", wantCompany: "", wantRole: "Go Developer"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			company, role := jobRoleAt(test.title)
			assert.Equal(t, test.wantCompany, company)
			assert.Equal(t, test.wantRole, role)
		})
	}
}
