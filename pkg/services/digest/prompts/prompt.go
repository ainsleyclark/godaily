// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prompts

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

// styleMD is the embedded voice guide that the model must follow when
// drafting social posts.
//
//go:embed style.md
var styleMD string

// introStyleMD is the embedded editorial voice guide for the digest email
// intro. It is dry and grounded like styleMD, but in email-paragraph form
// rather than the social-post form (no hashtags, line breaks, or char cap).
//
//go:embed intro-style.md
var introStyleMD string

// promptItem is the wire shape sent to the model — a stripped-down
// projection of news.Item that drops fields irrelevant to a post
// (Published, Comments) so input tokens stay low.
type promptItem struct {
	Source  string  `json:"source"`
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Author  string  `json:"author,omitempty"`
	Tag     string  `json:"tag,omitempty"`
	Section string  `json:"section,omitempty"` // canonical section name, for headline priority
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

// defaultFilterConfig is the production default: at most 3 items per source,
// at most 16 items total. The total is set a little above the old 12 because
// the round-robin in filterItems spends slots on breadth first — the extra
// headroom keeps depth in big sections without letting one section crowd the
// others out.
func defaultFilterConfig() filterConfig {
	return filterConfig{topPerSource: 3, totalCap: 16}
}

// filterItems takes the scored, per-source-sorted output from the aggregator
// and produces a flat list of promptItems suitable for feeding to the model.
// Rather than ranking sections against one another — which structurally
// favoured proposals and let them monopolise the payload the model sees — it
// guarantees breadth: items are grouped by canonical section, sorted by score
// within each section, and taken round-robin (one per section per round) until
// the total cap. The per-source score picks the strongest representative
// *within* a section, never ranks one section above another; the resulting
// list order carries no priority. Jobs and social posts are skipped entirely.
func filterItems(sections []news.SourceItems, cfg filterConfig) []promptItem {
	if cfg.topPerSource <= 0 || cfg.totalCap <= 0 {
		return nil
	}

	bySection := make(map[news.Tag][]news.Item)
	for _, section := range sections {
		take := cfg.topPerSource
		if len(section.Items) < take {
			take = len(section.Items)
		}
		for i := 0; i < take; i++ {
			it := section.Items[i]
			// Jobs and social posts add no value to an editorial intro or
			// subject line — skip them entirely.
			if it.Tag == news.TagJobs || it.Tag == news.TagSocial {
				continue
			}
			sec := it.Tag.Section()
			bySection[sec] = append(bySection[sec], it)
		}
	}

	for sec := range bySection {
		its := bySection[sec]
		sort.SliceStable(its, func(i, j int) bool {
			return its[i].Score > its[j].Score
		})
	}

	order := orderedSections(bySection)

	out := make([]promptItem, 0, cfg.totalCap)
	for round := 0; len(out) < cfg.totalCap; round++ {
		took := false
		for _, sec := range order {
			its := bySection[sec]
			if round >= len(its) {
				continue
			}
			out = append(out, toPromptItem(its[round]))
			took = true
			if len(out) >= cfg.totalCap {
				break
			}
		}
		if !took {
			break
		}
	}
	return out
}

// orderedSections returns the populated sections in a deterministic order:
// the canonical news.SectionTags order first, then any sections from unknown
// tags (sorted) so nothing is silently dropped. The order is for reproducible
// output only — the prompt tells the model it carries no priority.
func orderedSections(bySection map[news.Tag][]news.Item) []news.Tag {
	seen := make(map[news.Tag]bool, len(bySection))
	order := make([]news.Tag, 0, len(bySection))
	for _, sec := range news.SectionTags {
		if len(bySection[sec]) > 0 {
			order = append(order, sec)
			seen[sec] = true
		}
	}
	rest := make([]news.Tag, 0)
	for sec := range bySection {
		if !seen[sec] {
			rest = append(rest, sec)
		}
	}
	sort.Slice(rest, func(i, j int) bool { return rest[i] < rest[j] })
	return append(order, rest...)
}

func toPromptItem(it news.Item) promptItem {
	return promptItem{
		Source:  string(it.Source),
		Title:   it.Title,
		URL:     it.URL,
		Author:  it.Author.String(),
		Tag:     string(it.Tag),
		Section: it.Tag.Title(),
		Snippet: it.Snippet,
		Score:   it.Score,
	}
}

// buildUserPrompt formats the day's filtered items as a compact JSON
// payload. The date is rendered in plain text so the model never has to
// guess "today" from the items' Published timestamps.
func buildUserPrompt(day time.Time, items []promptItem) string {
	payload, _ := json.Marshal(items)
	return fmt.Sprintf(
		"Date: %s\nItems (a diverse shortlist across sections; order carries no priority):\n%s\n\nReturn the JSON object only.",
		day.Format("2006-01-02"), string(payload),
	)
}
