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

package og

import (
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) *Generator {
	t.Helper()
	g, err := New()
	require.NoError(t, err)
	return g
}

func TestGenerator_Home(t *testing.T) {
	t.Parallel()

	g := setup(t)
	got, err := g.Home()

	require.NoError(t, err)
	assert.Greater(t, len(got), 0)
}

func TestGenerator_Issue(t *testing.T) {
	t.Parallel()

	g := setup(t)

	tt := map[string]struct {
		input news.Issue
	}{
		"With items and date": {
			input: news.Issue{
				ID:      42,
				Slug:    "2026-05-12",
				Subject: "A Go roundup before standup",
				SentAt:  time.Date(2026, 5, 12, 8, 0, 0, 0, time.UTC),
				Items: []news.Item{
					{Title: "Go vs Java: The Minimalist vs The Enterprise Veteran"},
					{Title: "GoLand 2026.2 Early Access Program has started"},
					{Title: "60 days running — agent reliability in production Go"},
					{Title: "The Go memory model explained"},
					{Title: "Building a CLI in Go with cobra"},
				},
			},
		},
		"Fewer than 3 items": {
			input: news.Issue{
				ID:      1,
				Slug:    "2026-01-01",
				Subject: "First issue",
				Items: []news.Item{
					{Title: "Go 1.26 released"},
				},
			},
		},
		"No items, zero date": {
			input: news.Issue{
				ID:      99,
				Slug:    "draft",
				Subject: "Long headline that should be truncated if it exceeds the maximum rune limit set in the generator",
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := g.Issue(test.input)
			require.NoError(t, err)
			assert.Greater(t, len(got), 0)
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input string
		max   int
		want  string
	}{
		"No truncation needed": {input: "hello", max: 10, want: "hello"},
		"Exact max":            {input: "hello", max: 5, want: "hello"},
		"Truncated":            {input: "hello world", max: 8, want: "hello w…"},
		"Unicode truncation":   {input: "こんにちは世界", max: 5, want: "こんにち…"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := truncate(test.input, test.max)
			assert.Equal(t, test.want, got)
		})
	}
}
