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

// golangCafeFixture mimics the real listing page: a SvelteKit hydration script
// carrying a `jobPosts` JS-object-literal array (unquoted keys, a description
// containing colons and brackets to exercise the string-aware parser) plus the
// canonical /jobs/<slug> anchors used to resolve URLs by id.
const golangCafeFixture = `<!DOCTYPE html><html><head></head><body>
<main>
  <a href="/jobs/Acme-Corp-Senior-Go-Engineer-AAA111">Senior Go Engineer</a>
  <a href="/jobs/Beta-Ltd-Backend-Developer-BBB222">Backend Developer</a>
</main>
<script>
  const data = [null,{"type":"data","data":{jobPosts:[{id:"AAA111",imageId:"AAA111",title:"Senior Go Engineer",company:"Acme Corp",location:"London",country:"GB",description:"Build APIs: scale {fast}, ship [now]. Salary: great.",link:"mailto:jobs@acme.test",currency:"£",remote:"remote",salaryFrom:"90 000",salaryTo:"120 000",tags:[],websites:["Golang"],date:1748563200000},{id:"BBB222",title:"Backend Developer",company:"Beta Ltd",location:"Remote",country:"US",description:"Go backend role",link:"https://beta.test/apply",currency:"$",remote:"on_site",salaryFrom:"",salaryTo:"",tags:[],websites:["Golang"],date:1748476800000}]},"uses":{}}];
  Promise.all([]).then(() => {});
</script>
</body></html>`

func TestGolangCafe_Fetch(t *testing.T) {
	t.Parallel()

	fixedNow := func() time.Time {
		return time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
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
				require.Len(t, items, 2)

				byTitle := map[string]news.Item{}
				for _, it := range items {
					assert.Equal(t, news.SourceGolangCafe, it.Source)
					assert.Equal(t, news.TagJobs, it.Tag)
					byTitle[it.Title] = it
				}

				senior, ok := byTitle["Acme Corp · Senior Go Engineer · London"]
				require.True(t, ok, "salaried posting should be present")
				assert.Equal(t, "https://golang.cafe/jobs/Acme-Corp-Senior-Go-Engineer-AAA111", senior.URL)
				require.NotNil(t, senior.Author)
				assert.Equal(t, "Acme Corp", senior.Author.Name)
				assert.Equal(t, "£90k–£120k", senior.Snippet)
				// Real posting date is used, not collection time.
				assert.Equal(t, time.UnixMilli(1748563200000).UTC(), senior.Published)

				backend, ok := byTitle["Beta Ltd · Backend Developer · Remote"]
				require.True(t, ok, "second posting should be present")
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

func TestGolangCafe_Salary(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		currency, from, to string
		want               string
	}{
		"Dollar range":   {"$", "120 000", "160 000", "$120k–$160k"},
		"Euro min only":  {"€", "80000", "", "€80k+"},
		"Pound max only": {"£", "", "120000", "up to £120k"},
		"Undisclosed":    {"$", "", "", ""},
		"Default symbol": {"", "100 000", "150 000", "$100k–$150k"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, golangCafeSalary(test.currency, test.from, test.to))
		})
	}
}

func TestGolangCafe_JSObjectToJSON(t *testing.T) {
	t.Parallel()

	// Keys quoted; identifiers and brackets inside string values left intact.
	in := `[{id:"x",note:"see {a}: [b], true:false",ok:true,n:42}]`
	want := `[{"id":"x","note":"see {a}: [b], true:false","ok":true,"n":42}]`
	assert.Equal(t, want, jsObjectToJSON(in))
}
