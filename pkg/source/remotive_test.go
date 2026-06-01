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

// remotiveFixture covers a Go-in-title job with salary, a tag-only Go match
// without salary, and a non-Go job that must be filtered out.
const remotiveFixture = `{
  "jobs": [
    {
      "id": 1,
      "url": "https://remotive.com/remote-jobs/1",
      "title": "Senior Go Engineer",
      "company_name": "Acme Corp",
      "candidate_required_location": "Worldwide",
      "salary": "$120k - $160k",
      "publication_date": "2024-12-30 10:00:00",
      "tags": ["golang", "backend"]
    },
    {
      "id": 2,
      "url": "https://remotive.com/remote-jobs/2",
      "title": "Backend Engineer",
      "company_name": "Polyglot Inc",
      "candidate_required_location": "",
      "salary": "",
      "publication_date": "2024-12-30 10:00:00",
      "tags": ["golang", "python"]
    },
    {
      "id": 3,
      "url": "https://remotive.com/remote-jobs/3",
      "title": "Rust Engineer",
      "company_name": "NoFun",
      "candidate_required_location": "Berlin",
      "salary": "",
      "publication_date": "2024-12-30 10:00:00",
      "tags": ["rust"]
    }
  ]
}`

func TestRemotive_Fetch(t *testing.T) {
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
				_, err := w.Write([]byte(remotiveFixture))
				require.NoError(t, err)
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 2) // Rust listing filtered out.

				goEng := items[0]
				assert.Equal(t, news.SourceRemotive, goEng.Source)
				assert.Equal(t, news.TagJobs, goEng.Tag)
				assert.Equal(t, "Acme Corp · Senior Go Engineer · Worldwide", goEng.Title)
				assert.Equal(t, "https://remotive.com/remote-jobs/1", goEng.URL)
				require.NotNil(t, goEng.Author)
				assert.Equal(t, "Acme Corp", goEng.Author.Name)
				assert.Equal(t, "$120k - $160k", goEng.Snippet)
				assert.Equal(t, fixedNow(), goEng.Published)
				// Go-in-title + salary + remote, fresh: full boost.
				assert.Greater(t, goEng.Score, 1.5)

				backend := items[1]
				// Empty location renders as "Remote".
				assert.Equal(t, "Polyglot Inc · Backend Engineer · Remote", backend.Title)
				assert.Empty(t, backend.Snippet)
				// No salary, no Go in title: lower than goEng.
				assert.Less(t, backend.Score, goEng.Score)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			got, err := Remotive{url: s.URL, now: fixedNow}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestRemotive_AgeDays(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

	tt := map[string]struct {
		published string
		want      int
	}{
		"Same day":       {published: "2024-01-15 00:00:00", want: 0},
		"Three days":     {published: "2024-01-12 00:00:00", want: 3},
		"RFC3339 form":   {published: "2024-01-12T00:00:00Z", want: 3},
		"Unparseable":    {published: "not a date", want: 0},
		"Future floored": {published: "2024-01-20 00:00:00", want: 0},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, remotiveAgeDays(now, test.published))
		})
	}
}
