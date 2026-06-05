// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ingest

import (
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
)

type fakeTransformer struct {
	title   string
	snippet string
	url     string
	include bool
	enrich  string
}

func (f fakeTransformer) Transform() news.Item {
	return news.Item{Title: f.title, Snippet: f.snippet, URL: f.url}
}
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
		"Zero-width entity":    {input: "Hello &#x200B; world", want: "Hello world"},
		"Zero-width char":      {input: "Hello ​ world", want: "Hello world"},
		"Entity then heading":  {input: "&#x200B;\n## Title", want: "Title"},
		"Numeric entity":       {input: "caf&#233;", want: "café"},
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
		"Empty":                {items: nil, want: nil},
		"Multiple":             {items: []fakeTransformer{{title: "A", include: true}, {title: "B", include: true}}, want: []news.Item{{Title: "A"}, {Title: "B"}}},
		"Filtered by include":  {items: []fakeTransformer{{title: "A", include: true}, {title: "B", include: false}}, want: []news.Item{{Title: "A"}}},
		"Filtered by language": {items: []fakeTransformer{{title: "Сравнимые типы данных в Go", include: true}, {title: "Go Concurrency Patterns", include: true}}, want: []news.Item{{Title: "Go Concurrency Patterns"}}},
		"Filtered by snippet language": {items: []fakeTransformer{
			{title: "Coming from Node.js: What is the Go equivalent to Better Auth?", snippet: "Sou iniciante em Go. Quais são os principais pacotes? Como o Go é usado na web?", include: true},
			{title: "Go Concurrency Patterns", snippet: "A deep dive into goroutines and channels", include: true},
		}, want: []news.Item{{Title: "Go Concurrency Patterns", Snippet: "A deep dive into goroutines and channels"}}},
		"English snippet passes": {items: []fakeTransformer{{title: "Better Auth for Go", snippet: "What is the Go equivalent of the popular Node.js library?", include: true}}, want: []news.Item{{Title: "Better Auth for Go", Snippet: "What is the Go equivalent of the popular Node.js library?"}}},
		"HTML entities in title": {items: []fakeTransformer{{title: "pkg &amp; internal directories are way overused", include: true}}, want: []news.Item{{Title: "pkg & internal directories are way overused"}}},
		"Filtered by self URL":   {items: []fakeTransformer{{title: "A", url: "https://godaily.dev/blog/post", include: true}, {title: "B", include: true}}, want: []news.Item{{Title: "B"}}},
		"Filtered by self title": {items: []fakeTransformer{{title: "Launching GoDaily", include: true}, {title: "B", include: true}}, want: []news.Item{{Title: "B"}}},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, TransformAll(t.Context(), test.items))
		})
	}
}

func TestIsSelfContent(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		title string
		url   string
		want  bool
	}{
		"Unrelated title and URL":       {title: "Go generics deep dive", url: "https://example.com/post", want: false},
		"Title contains GoDaily exact":  {title: "Launching GoDaily", url: "", want: true},
		"Title contains godaily lower":  {title: "subscribe to godaily now", url: "", want: true},
		"Title contains GODAILY upper":  {title: "GODAILY is live", url: "", want: true},
		"URL points to godaily.dev":     {title: "Something else", url: "https://godaily.dev/issues/1", want: true},
		"URL points to www.godaily.dev": {title: "Something else", url: "https://www.godaily.dev/issues/1", want: true},
		"URL points to other domain":    {title: "Something else", url: "https://github.com/ainsleyclark/godaily", want: false},
		"Empty title and URL":           {title: "", url: "", want: false},
		"Invalid URL":                   {title: "Valid title", url: "://bad-url", want: false},
		"Both title and URL match":      {title: "GoDaily v2", url: "https://godaily.dev/v2", want: true},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			item := news.Item{Title: test.title, URL: test.url}
			assert.Equal(t, test.want, isSelfContent(item))
		})
	}
}

func TestIsEnglishText(t *testing.T) {
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

		"Russian title":             {input: "Сравнимые типы данных в Go #shorts", want: false},
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

		// Arabic — must be rejected (was slipping through before Arabic script check was added)
		"Arabic mixed with Latin tech terms": {input: "Learn Go Programming | #09 - Slices | كورس GoLang بالعربي", want: false},
		"Arabic only":                        {input: "شرح كامل لل Slices في ال GoLang بالتفصيل", want: false},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, isEnglishText(test.input))
		})
	}
}
