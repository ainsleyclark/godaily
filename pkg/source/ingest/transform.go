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

	lingua "github.com/pemistahl/lingua-go"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

var (
	urlRe       = regexp.MustCompile(`https?://\S+`)
	hashtagRe   = regexp.MustCompile(`#\S+`)
	techTokenRe = regexp.MustCompile(`\S*/\S*|\w+(?:\.\w+){2,}`) // strips r/golang, net/http, pkg.go.dev
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
		i.Title = html.UnescapeString(i.Title)
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

// langDetector is built once at startup with the languages most commonly seen
// in non-English Go content. Limiting to a targeted set keeps the in-memory
// model footprint small compared to loading all 75 supported languages.
var langDetector = lingua.NewLanguageDetectorBuilder().
	FromLanguages(
		lingua.English,
		lingua.Russian,
		lingua.German,
		lingua.French,
		lingua.Chinese,
		lingua.Japanese,
		lingua.Korean,
		lingua.Indonesian,
		lingua.Portuguese,
		lingua.Spanish,
	).
	Build()

// isEnglishTitle returns false when lingua confidently detects a non-English
// language. Before detection:
//   - URLs, hashtags, and path-like tech tokens (r/golang, pkg.go.dev, net/http)
//     are stripped so they don't mislead the detector on short English phrases.
//   - If fewer than 8 runes remain the text is too short for reliable detection
//     and the item passes through, avoiding false drops.
func isEnglishTitle(s string) bool {
	clean := urlRe.ReplaceAllString(s, " ")
	clean = hashtagRe.ReplaceAllString(clean, " ")
	clean = techTokenRe.ReplaceAllString(clean, " ")
	clean = strings.Join(strings.Fields(clean), " ")
	// A single character from a non-Latin script is conclusive — reject before
	// the probabilistic detector runs. This catches mixed titles like
	// "Go: Потоки, Sysmon и GC #shorts" where Latin tech terms outvote Cyrillic.
	if hasNonLatinScript(clean) {
		return false
	}
	if len([]rune(clean)) < 8 {
		return true
	}
	lang, exists := langDetector.DetectLanguageOf(clean)
	return !exists || lang == lingua.English
}

func hasNonLatinScript(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) ||
			unicode.Is(unicode.Han, r) ||
			unicode.Is(unicode.Hiragana, r) ||
			unicode.Is(unicode.Katakana, r) ||
			unicode.Is(unicode.Hangul, r) ||
			unicode.Is(unicode.Arabic, r) ||
			unicode.Is(unicode.Hebrew, r) ||
			unicode.Is(unicode.Devanagari, r) ||
			unicode.Is(unicode.Thai, r) {
			return true
		}
	}
	return false
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
