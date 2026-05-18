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
	"context"
	"html"
	"regexp"
	"strings"
	"unicode"

	"github.com/ainsleyclark/godaily/pkg/news"
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

	// EnrichmentURL returns the URL to crawl for snippet/image enrichment,
	// or "" to skip. Implementations live next to URL construction so
	// per-source URL knowledge stays in the source.
	EnrichmentURL() string
}

// TransformAll maps a slice of items that implement Transformer to []news.Item,
// skipping any item for which ShouldInclude() returns false. Each produced
// item's snippet is sanitised (HTML stripped, entities unescaped) and
// truncated to maxSnippetLen bytes — sources can put raw API content into
// news.Item.Snippet without per-source cleanup.
//
// Items whose transformer returns a non-empty EnrichmentURL are then passed
// to Enrich, which fills empty Snippet/ImageURL fields by fetching that URL
// once and extracting og:/twitter: meta tags.
func TransformAll[T Transformer](ctx context.Context, items []T) []news.Item {
	var (
		out        []news.Item
		enrichURLs []string
	)
	for _, item := range items {
		if !item.ShouldInclude() {
			continue
		}
		i := item.Transform()
		if !isEnglishTitle(i.Title) {
			continue
		}
		i.Snippet = truncate(sanitise(i.Snippet), maxSnippetLen)
		out = append(out, i)
		enrichURLs = append(enrichURLs, item.EnrichmentURL())
	}

	var targets []enrichTarget
	for i := range out {
		if enrichURLs[i] != "" {
			targets = append(targets, enrichTarget{URL: enrichURLs[i], Item: &out[i]})
		}
	}
	if len(targets) > 0 {
		enrich(ctx, targets)
	}

	return out
}

const maxSnippetLen = 200

// isEnglishTitle returns false when ≥25% of the letters in s belong to a
// non-Latin Unicode script (Cyrillic, CJK, Arabic, …). Titles with no letters
// (pure numbers, symbols, code snippets) are accepted.
func isEnglishTitle(s string) bool {
	var letters, nonLatin int
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if !unicode.Is(unicode.Latin, r) {
				nonLatin++
			}
		}
	}
	return letters == 0 || float64(nonLatin)/float64(letters) < 0.25
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

var (
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
	mdLinkRe  = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	mdNoiseRe = regexp.MustCompile("(?m)```[^`]*```|`[^`]*`|[#*_~]+")
)

// sanitise produces a clean, single-line snippet: strips HTML tags, collapses
// markdown links to their visible text, strips remaining markdown syntax
// (code fences, inline code, emphasis markers), unescapes HTML entities, and
// collapses runs of whitespace into a single space.
func sanitise(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = mdLinkRe.ReplaceAllString(s, "$1")
	s = mdNoiseRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}
