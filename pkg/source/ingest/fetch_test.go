// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ingest

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

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
				t.Helper()
				assert.ErrorContains(t, err, "request creation failed")
			},
		},
		"Do Error": {
			url:       closedServer.URL,
			unmarshal: json.Unmarshal,
			want: func(t *testing.T, _ fakeItem, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "fetch")
			},
		},
		"Bad Status": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			unmarshal: json.Unmarshal,
			want: func(t *testing.T, _ fakeItem, err error) {
				t.Helper()
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
				t.Helper()
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
				t.Helper()
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
				t.Helper()
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
				t.Helper()
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

			got, err := Fetch[fakeItem](t.Context(), url, "test", test.unmarshal)
			test.want(t, got, err)
		})
	}
}
