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

	"github.com/ainsleyclark/godaily/pkg/news"
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

func TestSanitise(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input string
		want  string
	}{
		"Plain text":           {input: "hello world", want: "hello world"},
		"HTML tags":            {input: "<b>bold</b> text", want: "bold text"},
		"HTML entities":        {input: "foo &amp; bar", want: "foo & bar"},
		"Inline code":          {input: "use `wg.Add(1)` to increment", want: "use to increment"},
		"Fenced code block":    {input: "```go\nfmt.Println()\n```", want: ""},
		"Emphasis markers":     {input: "**bold** and _italic_", want: "bold and italic"},
		"Heading marker":       {input: "## Section title", want: "Section title"},
		"Markdown link":        {input: "[Testo](https://github.com/ozontech/testo)", want: "Testo"},
		"Link mid-sentence":    {input: "built on [testing.T](https://pkg.go.dev/testing) package", want: "built on testing.T package"},
		"Multiple links":       {input: "[A](https://a.com) and [B](https://b.com)", want: "A and B"},
		"Collapsed whitespace": {input: "foo   \n  bar", want: "foo bar"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, sanitise(test.input))
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
