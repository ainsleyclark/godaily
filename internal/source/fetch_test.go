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
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
)

type fakeItem struct {
	Name string `json:"name" xml:"name"`
}

func TestFetch(t *testing.T) {
	// closedServer gives us a valid host:port that refuses connections.
	closedServer := httptest.NewServer(nil)
	closedServer.Close()

	t.Parallel()

	tt := map[string]struct {
		stub      http.HandlerFunc
		url       string // non-empty overrides the stub server URL
		unmarshal func([]byte, any) error
		want      func(t *testing.T, got fakeItem, err error)
	}{
		"Error Creating Request": {
			url:       ":@!£$",
			unmarshal: json.Unmarshal,
			want: func(t *testing.T, _ fakeItem, err error) {
				assert.ErrorContains(t, err, "request creation failed")
			},
		},
		"Do Error": {
			url:       closedServer.URL,
			unmarshal: json.Unmarshal,
			want: func(t *testing.T, _ fakeItem, err error) {
				assert.ErrorContains(t, err, "fetch")
			},
		},
		"Bad Status": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			unmarshal: json.Unmarshal,
			want: func(t *testing.T, _ fakeItem, err error) {
				assert.ErrorContains(t, err, "unexpected status code")
			},
		},
		"JSON Decode Error": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`bad json`))
			},
			unmarshal: json.Unmarshal,
			want: func(t *testing.T, _ fakeItem, err error) {
				assert.ErrorContains(t, err, "parsing response")
			},
		},
		"OK JSON": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"name":"gopher"}`))
			},
			unmarshal: json.Unmarshal,
			want: func(t *testing.T, got fakeItem, err error) {
				assert.NoError(t, err)
				assert.Equal(t, fakeItem{Name: "gopher"}, got)
			},
		},
		"XML Decode Error": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`not xml <<>>`))
			},
			unmarshal: xml.Unmarshal,
			want: func(t *testing.T, _ fakeItem, err error) {
				assert.ErrorContains(t, err, "parsing response")
			},
		},
		"OK XML": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`<root><name>gopher</name></root>`))
			},
			unmarshal: xml.Unmarshal,
			want: func(t *testing.T, got fakeItem, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "gopher", got.Name)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			url := test.url
			if url == "" {
				s := httptest.NewServer(test.stub)
				defer s.Close()
				url = s.URL
			}

			got, err := fetch[fakeItem](t.Context(), url, "test", test.unmarshal)
			test.want(t, got, err)
		})
	}
}

type fakeTransformer struct{ title string }

func (f fakeTransformer) transform() news.Item { return news.Item{Title: f.title} }

func TestTransformAll(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		items []fakeTransformer
		want  []news.Item
	}{
		"Empty":    {items: nil, want: []news.Item{}},
		"Multiple": {items: []fakeTransformer{{title: "A"}, {title: "B"}}, want: []news.Item{{Title: "A"}, {Title: "B"}}},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, transformAll(test.items))
		})
	}
}
