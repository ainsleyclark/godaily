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

package api

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryInt(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		query    string
		key      string
		fallback int64
		want     int64
	}{
		"Present and valid":  {query: "?n=42", key: "n", fallback: 1, want: 42},
		"Missing key":        {query: "", key: "n", fallback: 7, want: 7},
		"Non-numeric value":  {query: "?n=abc", key: "n", fallback: 5, want: 5},
		"Negative value":     {query: "?n=-3", key: "n", fallback: 1, want: -3},
		"Zero value":         {query: "?n=0", key: "n", fallback: 1, want: 0},
		"Empty string value": {query: "?n=", key: "n", fallback: 9, want: 9},
		"Different key":      {query: "?n=10", key: "m", fallback: 3, want: 3},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("GET", "/"+test.query, nil)
			got := QueryInt(r, test.key, test.fallback)
			assert.Equal(t, test.want, got)
		})
	}
}
