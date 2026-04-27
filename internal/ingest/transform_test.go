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

package ingest

import (
	"testing"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
)

type fakeTransformer struct {
	title   string
	include bool
	enrich  string
}

func (f fakeTransformer) Transform() news.Item  { return news.Item{Title: f.title} }
func (f fakeTransformer) ShouldInclude() bool   { return f.include }
func (f fakeTransformer) EnrichmentURL() string { return f.enrich }

func TestTruncate(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input string
		max   int
		want  string
	}{
		"Below limit": {input: "short", max: 200, want: "short"},
		"At limit":    {input: "exactly10!", max: 10, want: "exactly10!"},
		"Above limit": {input: "hello world", max: 5, want: "hello"},
		"Empty":       {input: "", max: 200, want: ""},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, truncate(test.input, test.max))
		})
	}
}

func TestTransformAll(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		items []fakeTransformer
		want  []news.Item
	}{
		"Empty":    {items: nil, want: nil},
		"Multiple": {items: []fakeTransformer{{title: "A", include: true}, {title: "B", include: true}}, want: []news.Item{{Title: "A"}, {Title: "B"}}},
		"Filtered": {items: []fakeTransformer{{title: "A", include: true}, {title: "B", include: false}}, want: []news.Item{{Title: "A"}}},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, TransformAll(t.Context(), test.items))
		})
	}
}
