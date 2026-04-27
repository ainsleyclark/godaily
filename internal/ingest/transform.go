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
	"html"
	"regexp"
	"strings"

	"github.com/ainsleyclark/godaily/internal/news"
)

// Transformer is implemented by all per-source response item types.
type Transformer interface {
	// Transform converts the raw API response item into a news.Item. The
	// Snippet field need not be pre-truncated; TransformAll handles that.
	Transform() news.Item

	// ShouldInclude reports whether this item should be included in the
	// feed. Return false to silently drop the item before it reaches the
	// caller.
	ShouldInclude() bool
}

// TransformAll maps a slice of items that implement Transformer to []news.Item,
// skipping any item for which ShouldInclude() returns false. Each produced
// item's snippet is sanitised (HTML stripped, entities unescaped) and
// truncated to maxSnippetLen bytes — sources can put raw API content into
// news.Item.Snippet without per-source cleanup.
func TransformAll[T Transformer](items []T) []news.Item {
	var out []news.Item
	for _, item := range items {
		if !item.ShouldInclude() {
			continue
		}
		i := item.Transform()
		i.Snippet = truncate(sanitise(i.Snippet), maxSnippetLen)
		out = append(out, i)
	}
	return out
}

const maxSnippetLen = 200

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// sanitise produces a clean, single-line snippet: strips HTML tags, unescapes
// entities, and collapses runs of whitespace (including newlines) into a
// single space.
func sanitise(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}
