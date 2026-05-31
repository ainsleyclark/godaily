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
