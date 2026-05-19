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
		"Empty":                  {items: nil, want: nil},
		"Multiple":               {items: []fakeTransformer{{title: "A", include: true}, {title: "B", include: true}}, want: []news.Item{{Title: "A"}, {Title: "B"}}},
		"Filtered by include":    {items: []fakeTransformer{{title: "A", include: true}, {title: "B", include: false}}, want: []news.Item{{Title: "A"}}},
		"Filtered by language":   {items: []fakeTransformer{{title: "Сравнимые типы данных в Go", include: true}, {title: "Go Concurrency Patterns", include: true}}, want: []news.Item{{Title: "Go Concurrency Patterns"}}},
		"HTML entities in title": {items: []fakeTransformer{{title: "pkg &amp; internal directories are way overused", include: true}}, want: []news.Item{{Title: "pkg & internal directories are way overused"}}},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, TransformAll(t.Context(), test.items))
		})
	}
}

func TestIsEnglishTitle(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input string
		want  bool
	}{
		// Clearly English — must always pass
		"Plain English sentence":      {input: "Writing a concurrent web server in Go", want: true},
		"English with Go jargon":      {input: "Understanding goroutines and channels", want: true},
		"English with version number": {input: "Go 1.23 release notes and new features", want: true},
		"English with hashtags":       {input: "Go generics deep dive #golang #shorts", want: true},
		"English with URL-like text":  {input: "pkg.go.dev tips and tricks", want: true},
		"English with code snippet":   {input: "Why use sync.Mutex instead of channels", want: true},
		"English with numbers":        {input: "10 mistakes every Go developer makes", want: true},
		"English repo name style":     {input: "[Livecoding] Building a REST API in Go", want: true},
		"English with trailing URL":   {input: "Understanding Go modules https://go.dev/ref/mod", want: true},

		// Ambiguous / short — pass through; not enough natural-language words
		// for lingua to make a reliable decision after URL/hashtag stripping.
		"Empty string":               {input: "", want: true},
		"Only numbers":               {input: "123 456", want: true},
		"Only hashtags":              {input: "#golang #go #shorts!", want: true},
		"Only a URL":                 {input: "https://pkg.go.dev/net/http", want: true},
		"Single common English word": {input: "Golang", want: true},
		"Two words only":             {input: "Go programming", want: true},

		// Russian (Cyrillic) — must be rejected
		"Russian mixed with Latin tech terms": {input: "Go: Потоки, Sysmon и GC #shorts", want: false},
		"Russian title":                        {input: "Сравнимые типы данных в Go #shorts", want: false},
		"Russian salary post":       {input: "Тирлист зарплат в IT #golang #it #собеседование", want: false},
		"Russian mutex explanation": {input: "RV-мьютексы против обычных: когда читать быстрее", want: false},
		"Russian subscribe CTA":     {input: "ПОДПИШИСЬ НА ТГ: cdmtn #it #code #golang", want: false},
		"Russian goroutines post":   {input: "Горутины в Go: полное руководство", want: false},

		// German (Latin script) — must be rejected
		"German gamedev post":       {input: "So langsam kommt Leben in das kleine Spiel", want: false},
		"German technical sentence": {input: "Nebenläufigkeit in Go mit Goroutinen und Kanälen", want: false},

		// Other non-English languages
		"Japanese title":              {input: "Goの並行処理パターンを理解する", want: false},
		"Chinese title":               {input: "Go语言并发编程实战", want: false},
		"Korean title":                {input: "Go 언어로 웹 서버 만들기", want: false},
		"Indonesian livecoding title": {input: "Bikin Web Forum Pake Golang dan React", want: false},
		"Portuguese title":            {input: "Concorrência em Go com goroutines e canais", want: false},
		"Spanish title":               {input: "Construyendo microservicios con Go y gRPC", want: false},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, isEnglishTitle(test.input))
		})
	}
}
