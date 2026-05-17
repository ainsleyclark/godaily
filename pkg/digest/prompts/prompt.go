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

package prompts

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ainsleyclark/godaily/pkg/news"
)

// styleMD is the embedded voice guide that the model must follow when
// drafting posts.
//
//go:embed style.md
var styleMD string

// systemIntro is the task framing prepended to the style guide for social posts.
const systemIntro = `You write a single short social media post about the
Go programming language community in the voice of Ainsley Clark.

You will receive a JSON list of items aggregated from Go news sources for
a single day, already ranked by relevance. Pick the SINGLE most notable
item (the one with the most technical substance) and write one short,
punchy post about that one topic. Go deep on one thing, do not summarise
the day, do not list multiple items, do not produce a checklist or
roundup.

If a small cluster of items is clearly the same topic (same release,
same proposal, same project), treat them as one and reference both. The
"references" array should contain only the item(s) the post is actually
about (usually one, occasionally two).

Output strict JSON, schema:
{
  "post":       string  // <= 280 chars, one topic
  "references": [{"title": string, "url": string, "source": string}, ...]
}

Output the JSON object alone. No prose, no markdown fences, no commentary.`

// digestSystemIntro is the task framing for digest metadata synthesis.
const digestSystemIntro = `You are an editor writing metadata for a daily Go programming language digest email.

You will receive a JSON list of items aggregated from Go news sources for a single day, already ranked by relevance.

Output strict JSON, schema:
{
  "title": string  // <=80 chars — punchy email subject line teaser drawn from the top item (e.g. "Go 1.24 lands, goroutines got faster")
  "intro": string  // 1-2 plain sentences summarising what mattered most today, for the top of the email body
}

Do not begin the intro with "Today" or the date. Write in present tense, active voice, no filler.
Output the JSON object alone. No prose, no markdown fences, no commentary.`

// buildSuggestSystem returns the complete system prompt string for social-post suggestion.
func buildSuggestSystem() string {
	return systemIntro + "\n\n## Style guide\n\n" + styleMD
}

// buildDigestSystem returns the complete system prompt string for digest metadata synthesis.
func buildDigestSystem() string {
	return digestSystemIntro + "\n\n## Style guide\n\n" + styleMD
}

// promptItem is the wire shape sent to the model — a stripped-down
// projection of news.Item that drops fields irrelevant to a post
// (Published, Comments) so input tokens stay low.
type promptItem struct {
	Source  string  `json:"source"`
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Author  string  `json:"author,omitempty"`
	Tag     string  `json:"tag,omitempty"`
	Snippet string  `json:"snippet,omitempty"`
	Score   float64 `json:"score"`
}

// filterConfig caps how many items reach the model. The per-source cap
// guarantees signal diversity (a noisy Reddit day cannot drown a Go Blog
// post); the total cap bounds input tokens.
type filterConfig struct {
	topPerSource int
	totalCap     int
}

// defaultFilterConfig is the production default: at most 3 items per
// source, at most 12 items total.
func defaultFilterConfig() filterConfig {
	return filterConfig{topPerSource: 3, totalCap: 12}
}

// filterItems takes the scored, per-source-sorted output from the
// aggregator and produces a flat, score-desc list of promptItems
// suitable for feeding to the model. Empty sections are skipped.
func filterItems(sections []news.SourceItems, cfg filterConfig) []promptItem {
	if cfg.topPerSource <= 0 || cfg.totalCap <= 0 {
		return nil
	}

	out := make([]promptItem, 0, cfg.totalCap)
	for _, section := range sections {
		take := cfg.topPerSource
		if len(section.Items) < take {
			take = len(section.Items)
		}
		for i := 0; i < take; i++ {
			it := section.Items[i]
			out = append(out, promptItem{
				Source:  string(it.Source),
				Title:   it.Title,
				URL:     it.URL,
				Author:  it.Author.String(),
				Tag:     string(it.Tag),
				Snippet: it.Snippet,
				Score:   it.Score,
			})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})

	if len(out) > cfg.totalCap {
		out = out[:cfg.totalCap]
	}

	return out
}

// buildUserPrompt formats the day's filtered items as a compact JSON
// payload. The date is rendered in plain text so the model never has to
// guess "today" from the items' Published timestamps.
func buildUserPrompt(day time.Time, items []promptItem) string {
	payload, _ := json.Marshal(items)
	return fmt.Sprintf(
		"Date: %s\nItems (highest score first):\n%s\n\nReturn the JSON object only.",
		day.Format("2006-01-02"), string(payload),
	)
}
