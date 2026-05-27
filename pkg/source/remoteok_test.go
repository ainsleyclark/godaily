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
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// remoteOKFixture is a trimmed real-shaped response: a metadata first element
// (filtered out) plus three jobs covering the salary-disclosed, salary-missing,
// and non-Go-but-tagged cases.
const remoteOKFixture = `[
  {"0":"Legal","legal":"All data..."},
  {
    "id":"go-1",
    "slug":"acme-senior-go-engineer",
    "epoch":1735603200,
    "company":"Acme Corp",
    "position":"Senior Go Engineer",
    "tags":["golang","backend"],
    "location":"Worldwide",
    "salary_min":120000,
    "salary_max":160000,
    "apply_url":"https://acme.example/apply",
    "url":"https://remoteok.com/remote-jobs/go-1"
  },
  {
    "id":"go-2",
    "slug":"polyglot-shop-backend-dev",
    "epoch":1735603200,
    "company":"Polyglot Inc",
    "position":"Backend Engineer",
    "tags":["golang","python"],
    "location":"",
    "salary_min":0,
    "salary_max":0,
    "apply_url":"",
    "url":"https://remoteok.com/remote-jobs/go-2"
  },
  {
    "id":"drop-1",
    "slug":"unrelated",
    "epoch":1735603200,
    "company":"NoFun",
    "position":"Rust Engineer",
    "tags":["rust"],
    "location":"Berlin",
    "url":"https://remoteok.com/remote-jobs/drop-1"
  }
]`

func TestRemoteOK_Fetch(t *testing.T) {
	t.Parallel()

	// Pin "now" to one day after the fixture epoch so age is deterministic.
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
				_, err := w.Write([]byte(remoteOKFixture))
				require.NoError(t, err)
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				// Metadata element and Rust listing are filtered out.
				require.Len(t, items, 2)

				goEng := items[0]
				assert.Equal(t, news.SourceRemoteOK, goEng.Source)
				assert.Equal(t, news.TagJobs, goEng.Tag)
				assert.Equal(t, "Senior Go Engineer", goEng.Title)
				assert.Equal(t, "https://acme.example/apply", goEng.URL)
				assert.Equal(t, "https://remoteok.com/remote-jobs/go-1", goEng.OriginalURL)
				require.NotNil(t, goEng.Author)
				assert.Equal(t, "Acme Corp", goEng.Author.Name)
				assert.Contains(t, goEng.Snippet, "Acme Corp")
				assert.Contains(t, goEng.Snippet, "Worldwide")
				assert.Contains(t, goEng.Snippet, "$120k")
				// Salary-disclosed + Go-in-title + remote, fresh: full boost.
				assert.Greater(t, goEng.Score, 1.5)

				backend := items[1]
				assert.Equal(t, "Backend Engineer", backend.Title)
				// No salary, no Go in title - lower score than goEng.
				assert.Less(t, backend.Score, goEng.Score)
				// Falls back to "Remote" when location is empty.
				assert.Contains(t, backend.Snippet, "Remote")
				// apply_url empty → URL falls back to the listing URL.
				assert.Equal(t, "https://remoteok.com/remote-jobs/go-2", backend.URL)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			got, err := RemoteOK{url: s.URL, now: fixedNow}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestRemoteOK_AgeDays(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

	tt := map[string]struct {
		epoch int64
		want  int
	}{
		"Same day":    {epoch: now.Unix(), want: 0},
		"Three days":  {epoch: now.Add(-72 * time.Hour).Unix(), want: 3},
		"Zero epoch":  {epoch: 0, want: 0},
		"Future date": {epoch: now.Add(48 * time.Hour).Unix(), want: 0},
		"Negative":    {epoch: -100, want: 0},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, remoteOKAgeDays(now, test.epoch))
		})
	}
}

func TestRemoteOK_FormatSalary(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		min, max float64
		want     string
	}{
		"Both bounds":  {min: 80000, max: 120000, want: "$80k–$120k"},
		"Min only":     {min: 100000, max: 0, want: "$100k+"},
		"Max only":     {min: 0, max: 90000, want: "up to $90k"},
		"Neither":      {min: 0, max: 0, want: ""},
		"Sub thousand": {min: 500, max: 800, want: "$500–$800"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, formatSalary(test.min, test.max))
		})
	}
}

func TestRemoteOK_ShouldInclude(t *testing.T) {
	t.Parallel()

	// 20 unrelated tags + golang — the keyword-spam pattern we want to drop.
	spamTags := []string{
		"golang", "swift", "mongo", "design", "recruiter", "marketing",
		"finance", "medical", "robotics", "education", "dev", "mobile",
		"digital nomad", "exec", "part time", "travel", "ops", "hr",
		"technical", "coordinator", "admin",
	}

	tt := map[string]struct {
		job  remoteOKJob
		want bool
	}{
		"Go in position passes": {
			job:  remoteOKJob{Position: "Senior Go Engineer", URL: "https://x", Tags: []string{"go"}},
			want: true,
		},
		"Tag-only match with sane tag count passes": {
			job:  remoteOKJob{Position: "Backend Engineer", URL: "https://x", Tags: []string{"golang", "backend", "remote"}},
			want: true,
		},
		"Spam tag set dropped": {
			job:  remoteOKJob{Position: "The perfect role not posted yet", URL: "https://x", Tags: spamTags},
			want: false,
		},
		"Empty position rejected": {
			job:  remoteOKJob{Position: "", URL: "https://x", Tags: []string{"golang"}},
			want: false,
		},
		"Empty URL rejected": {
			job:  remoteOKJob{Position: "Senior Go Engineer", URL: "", Tags: []string{"golang"}},
			want: false,
		},
		"Non-Go listing rejected": {
			job:  remoteOKJob{Position: "Rust Engineer", URL: "https://x", Tags: []string{"rust"}},
			want: false,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, test.job.ShouldInclude())
		})
	}
}

func TestRemoteOK_TrimsLocationPunctuation(t *testing.T) {
	t.Parallel()

	// API returned a half-typed location: "Reston, " with trailing space and
	// dangling comma. The snippet must trim both so it doesn't read like a
	// truncated sentence.
	job := remoteOKJob{
		Company:  "The Group, LLC",
		Location: "Reston, ",
	}
	got := buildRemoteOKSnippet(job)
	assert.Equal(t, "The Group, LLC · Reston", got)
}
