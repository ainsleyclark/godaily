// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
