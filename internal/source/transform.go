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

package source

import "github.com/ainsleyclark/godaily/internal/news"

// transformer is implemented by all per-source response item types.
type transformer interface {
	// transform converts the raw API response item into a news.Item. The
	// Snippet field need not be pre-truncated; transformAll handles that.
	transform() news.Item
	// shouldInclude reports whether this item should be included in the feed.
	// Return false to silently drop the item before it reaches the caller.
	shouldInclude() bool
}

// transformAll maps a slice of items that implement transformer to []news.Item,
// skipping any item for which shouldInclude() returns false. The Snippet field
// of each produced item is capped at snippetMaxLen bytes.
func transformAll[T transformer](items []T) []news.Item {
	var out []news.Item
	for _, item := range items {
		if !item.shouldInclude() {
			continue
		}
		i := item.transform()
		i.Snippet = truncateSnippet(i.Snippet, snippetMaxLen)
		out = append(out, i)
	}
	return out
}

const snippetMaxLen = 200

func truncateSnippet(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
