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

// defaultFilterConfig is the production default: at most 3 items per
// source, at most 12 items total.
func defaultFilterConfig() filterConfig {
	return filterConfig{topPerSource: 3, totalCap: 12}
}

// sectionRank maps a tag's canonical section to its position in
// news.SectionTags (0 = highest priority). Tags outside the list sort last.
// It is the single source of truth the headline rule also references, so the
// payload the model sees is ordered the same way the digest itself is.
func sectionRank(tag news.Tag) int {
	section := tag.Section()
	for i, s := range news.SectionTags {
		if s == section {
			return i
		}
	}
	return len(news.SectionTags)
}

// filterItems takes the scored, per-source-sorted output from the aggregator
// and produces a flat list of promptItems suitable for feeding to the model.
// Items are ordered by section priority first (news.SectionTags), then by
// score within a section, and truncated to the total cap. Ordering by section
// rather than raw score ensures high-priority sections (releases, proposals)
// survive truncation instead of being crowded out by a high-scoring but
// lower-priority source. Empty sections are skipped.
func filterItems(sections []news.SourceItems, cfg filterConfig) []promptItem {
	if cfg.topPerSource <= 0 || cfg.totalCap <= 0 {
		return nil
	}

	type ranked struct {
		item promptItem
		rank int
	}

	out := make([]ranked, 0, cfg.totalCap)
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
			out = append(out, ranked{
				item: promptItem{
					Source:  string(it.Source),
					Title:   it.Title,
					URL:     it.URL,
					Author:  it.Author.String(),
					Tag:     string(it.Tag),
					Section: it.Tag.Title(),
					Snippet: it.Snippet,
					Score:   it.Score,
				},
				rank: sectionRank(it.Tag),
			})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].rank != out[j].rank {
			return out[i].rank < out[j].rank
		}
		return out[i].item.Score > out[j].item.Score
	})

	if len(out) > cfg.totalCap {
		out = out[:cfg.totalCap]
	}

	items := make([]promptItem, len(out))
	for i, r := range out {
		items[i] = r.item
	}
	return items
}

// buildUserPrompt formats the day's filtered items as a compact JSON
// payload. The date is rendered in plain text so the model never has to
// guess "today" from the items' Published timestamps.
func buildUserPrompt(day time.Time, items []promptItem) string {
	payload, _ := json.Marshal(items)
	return fmt.Sprintf(
		"Date: %s\nItems (ordered by section priority, then score):\n%s\n\nReturn the JSON object only.",
		day.Format("2006-01-02"), string(payload),
	)
}
