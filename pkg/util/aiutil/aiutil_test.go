// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aiutil_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/util/aiutil"
)

func TestSanitisePost(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"No Em Dash":              {in: "Go is fast", want: "Go is fast"},
		"Em Dash Mid Sentence":    {in: "fast — really fast", want: "fast - really fast"},
		"Em Dash No Spaces":       {in: "fast—really fast", want: "fast-really fast"},
		"Multiple Em Dashes":      {in: "a — b — c", want: "a - b - c"},
		"Em Dash At Start":        {in: "— leading", want: "- leading"},
		"Em Dash At End":          {in: "trailing —", want: "trailing -"},
		"Hyphen Preserved":        {in: "swiss-table", want: "swiss-table"},
		"Empty String":            {in: "", want: ""},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, aiutil.SanitisePost(tc.in))
		})
	}
}

func TestStripFences(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"No Fence":                        {in: `{"a":1}`, want: `{"a":1}`},
		"Whitespace Only":                 {in: "  \n  ", want: ""},
		"JSON Fence":                      {in: "```json\n{\"a\":1}\n```", want: `{"a":1}`},
		"Plain Fence":                     {in: "```\n{\"a\":1}\n```", want: `{"a":1}`},
		"Surrounding Spaces":              {in: "  ```json\n{\"a\":1}\n```  ", want: `{"a":1}`},
		"Fence Without Close":             {in: "```json\n{\"a\":1}", want: `{"a":1}`},
		"Fence Without Newline (no body)": {in: "```json", want: "```json"},
		"Single Line Fence":               {in: "```{a}```", want: "```{a}```"},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, aiutil.StripFences(tc.in))
		})
	}
}
