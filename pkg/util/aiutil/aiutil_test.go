// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aiutil_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/util/aiutil"
)

func TestSanitisePost(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"No Em Dash":           {in: "Go is fast", want: "Go is fast"},
		"Em Dash Mid Sentence": {in: "fast — really fast", want: "fast - really fast"},
		"Em Dash No Spaces":    {in: "fast—really fast", want: "fast-really fast"},
		"Multiple Em Dashes":   {in: "a — b — c", want: "a - b - c"},
		"Em Dash At Start":     {in: "— leading", want: "- leading"},
		"Em Dash At End":       {in: "trailing —", want: "trailing -"},
		"Hyphen Preserved":     {in: "swiss-table", want: "swiss-table"},
		"Empty String":         {in: "", want: ""},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, aiutil.SanitisePost(tc.in))
		})
	}
}

func TestTruncatePost(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in    string
		limit int
		want  string
	}{
		"Empty String":         {in: "", limit: 10, want: ""},
		"Under Limit":          {in: "short", limit: 10, want: "short"},
		"Exactly At Limit":     {in: "exactly10!", limit: 10, want: "exactly10!"},
		"Word Boundary":        {in: "one two three four", limit: 10, want: "one two…"},
		"Single Long Token":    {in: "abcdefghijklmnop", limit: 10, want: "abcdefghi…"},
		"Zero Limit":           {in: "anything", limit: 0, want: ""},
		"Negative Limit":       {in: "anything", limit: -5, want: ""},
		"Trailing Space Trim":  {in: "aaaa bbbbbbbbbb", limit: 8, want: "aaaa…"},
		"Multibyte Under":      {in: "café", limit: 4, want: "café"},
		"Multibyte Truncation": {in: "héllo wörld foo", limit: 10, want: "héllo…"},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := aiutil.TruncatePost(tc.in, tc.limit)
			assert.Equal(t, tc.want, got)
			if tc.limit > 0 {
				assert.LessOrEqual(t, utf8.RuneCountInString(got), tc.limit,
					"result must never exceed the rune limit")
			}
		})
	}
}

// TestTruncatePost_NeverExceedsLimit guards the core invariant the Bluesky
// 300-grapheme cap relies on: the result is always within the rune limit
// regardless of input, so it can never exceed the platform grapheme limit.
func TestTruncatePost_NeverExceedsLimit(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("word ", 200) // 1000 chars
	for _, limit := range []int{1, 10, 50, 280, 300, 500} {
		got := aiutil.TruncatePost(long, limit)
		assert.LessOrEqualf(t, utf8.RuneCountInString(got), limit,
			"limit=%d produced %d runes", limit, utf8.RuneCountInString(got))
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
