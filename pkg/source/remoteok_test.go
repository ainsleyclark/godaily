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
				// Title is now "Company · Role · Location" for scannability.
				assert.Equal(t, "Acme Corp · Senior Go Engineer · Worldwide", goEng.Title)
				assert.Equal(t, "https://acme.example/apply", goEng.URL)
				assert.Equal(t, "https://remoteok.com/remote-jobs/go-1", goEng.OriginalURL)
				require.NotNil(t, goEng.Author)
				assert.Equal(t, "Acme Corp", goEng.Author.Name)
				// Snippet is the salary range only — company and location are in the title.
				assert.Equal(t, "$120k–$160k", goEng.Snippet)
				// Salary-disclosed + Go-in-title + remote, fresh: full boost.
				assert.Greater(t, goEng.Score, 1.5)

				backend := items[1]
				// Empty location renders as "Remote" in the title.
				assert.Equal(t, "Polyglot Inc · Backend Engineer · Remote", backend.Title)
				// No salary disclosed → snippet is empty (template skips it silently).
				assert.Empty(t, backend.Snippet)
				// No salary, no Go in title - lower score than goEng.
				assert.Less(t, backend.Score, goEng.Score)
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
		"Mid-count spam (13 tags) dropped": {
			// The US Army Corps "Interdisciplinary" listing pattern: 13 tags
			// spanning unrelated industries. Just over the legitimate-listing
			// max (~4–8 tags) and easily caught by the cap.
			job: remoteOKJob{
				Position: "Interdisciplinary",
				URL:      "https://x",
				Tags: []string{
					"customer support", "engineer", "marketing", "finance",
					"medical", "recruiter", "full time", "robotics",
					"education", "dev", "mobile", "golang", "digital nomad",
				},
			},
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

func TestRemoteOK_DisplayLocation(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"Specific country":         {in: "Italy", want: "Italy"},
		"City and country":         {in: "London, UK", want: "London, UK"},
		"Trailing comma trimmed":   {in: "Reston, ", want: "Reston"},
		"Multiple trailing commas": {in: "London,,", want: "London"},
		"Empty falls back":         {in: "", want: "Remote"},
		"Whitespace only":          {in: "  ", want: "Remote"},
		"Portuguese remoto":        {in: "Remoto", want: "Remote"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, remoteOKDisplayLocation(test.in))
		})
	}
}

func TestRemoteOK_BuildTitle(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		job  remoteOKJob
		want string
	}{
		"Company, position and specific location": {
			job:  remoteOKJob{Company: "Acme", Position: "Senior Go Engineer", Location: "Italy"},
			want: "Acme · Senior Go Engineer · Italy",
		},
		"Empty location renders as Remote": {
			job:  remoteOKJob{Company: "Acme", Position: "Senior Go Engineer", Location: ""},
			want: "Acme · Senior Go Engineer · Remote",
		},
		"Malformed location is sanitised": {
			job:  remoteOKJob{Company: "Acme", Position: "Backend Engineer", Location: "Reston, "},
			want: "Acme · Backend Engineer · Reston",
		},
		"Missing company falls back to role · location": {
			job:  remoteOKJob{Company: "", Position: "Senior Go Engineer", Location: "Italy"},
			want: "Senior Go Engineer · Italy",
		},
		"Empty position returns empty": {
			job:  remoteOKJob{Company: "Acme", Position: "", Location: "Italy"},
			want: "",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, buildRemoteOKTitle(test.job))
		})
	}
}
