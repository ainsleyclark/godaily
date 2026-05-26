// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package featured

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/util/aiutil"
)

// tagWeights bias the candidate shortlist toward items most likely to drive
// engagement: shipped/accepted proposals, releases, and rich articles win
// over trending discussions and short videos.
var tagWeights = map[news.Tag]float64{
	news.TagProposalShipped:  1.0,
	news.TagProposalAccepted: 0.95,
	news.TagRelease:          0.9,
	news.TagProposal:         0.85,
	news.TagArticle:          0.8,
	news.TagDiscussion:       0.5,
	news.TagPodcast:          0.45,
	news.TagVideo:            0.4,
	news.TagTrending:         0.2,
}

// maxCandidates caps the number of items presented to the model. Twelve is
// enough variety without blowing token budget.
const maxCandidates = 12

// candidate is the wire shape sent to the model.
type candidate struct {
	Title    string  `json:"title"`
	URL      string  `json:"url"`
	Source   string  `json:"source"`
	Tag      string  `json:"tag"`
	Snippet  string  `json:"snippet,omitempty"`
	Score    float64 `json:"score"`
	Weighted float64 `json:"weighted_score"`
}

// buildCandidates filters and scores items, returning the highest-weighted
// shortlist. The weighted score multiplies the item's per-source relevance
// score by the tag weight, so a borderline article still loses to a fresh
// language release.
func buildCandidates(items []news.Item) []candidate {
	out := make([]candidate, 0, len(items))
	for _, it := range items {
		w, ok := tagWeights[it.Tag]
		if !ok {
			// Unknown tags get a modest baseline so they remain eligible
			// but don't beat tagged items.
			w = 0.3
		}
		weighted := it.Score * w
		if weighted == 0 {
			weighted = w
		}
		out = append(out, candidate{
			Title:    it.Title,
			URL:      it.URL,
			Source:   string(it.Source),
			Tag:      string(it.Tag),
			Snippet:  it.Snippet,
			Score:    it.Score,
			Weighted: weighted,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Weighted > out[j].Weighted
	})

	if len(out) > maxCandidates {
		out = out[:maxCandidates]
	}

	return out
}

const featureSystem = `You select the single most engaging Go-community item
of the day to anchor a social media post.

You will receive a JSON list of items already scored and filtered to today's
shortlist. Pick the one item with the most engagement potential, biased
heavily toward (in order): accepted/shipped language proposals, major
releases, formal proposals, in-depth technical articles. A "discussion" or
"trending" item is only acceptable if nothing more substantial is available.

If multiple items cover the same topic (same release, same proposal, same
project), treat them as one and pick the canonical link.

Output strict JSON, schema:
{
  "title":  string,        // copied verbatim from the chosen item
  "url":    string,        // copied verbatim from the chosen item
  "source": string,        // copied verbatim from the chosen item
  "tag":    string,        // copied verbatim from the chosen item
  "hook":   string         // 1 short sentence, max 25 words, factual:
                           //   what shipped/landed/dropped and why a Go dev
                           //   would care. NO marketing language. NO emojis.
                           //   NO "exciting"/"huge"/"game-changer".
}

Output the JSON object alone. No prose, no markdown fences.`

// Feature asks the model to pick the day's featured item and supply a short
// factual hook. The hook is reused as seed material by the per-platform
// reframing prompts.
func Feature(ctx context.Context, p ai.Prompter, day time.Time, items []news.Item) (Featured, error) {
	if p == nil {
		return Featured{}, errors.New("prompts: ai.Prompter is required")
	}

	cands := buildCandidates(items)
	if len(cands) == 0 {
		return Featured{}, ErrNoCandidates
	}

	payload, err := json.Marshal(cands)
	if err != nil {
		return Featured{}, errors.Wrap(err, "marshalling candidates")
	}

	user := fmt.Sprintf(
		"Date: %s\nCandidates (highest weighted score first):\n%s\n\nReturn the JSON object only.",
		day.Format("2006-01-02"), string(payload),
	)

	raw, err := p.Prompt(ctx, featureSystem, user)
	if err != nil {
		return Featured{}, errors.Wrap(err, "ai")
	}

	return parseFeatured(raw)
}

func parseFeatured(raw []byte) (Featured, error) {
	body := aiutil.StripFences(string(raw))
	if body == "" {
		return Featured{}, errors.New("empty featured response")
	}

	var out struct {
		Title  string `json:"title"`
		URL    string `json:"url"`
		Source string `json:"source"`
		Tag    string `json:"tag"`
		Hook   string `json:"hook"`
	}

	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return Featured{}, fmt.Errorf("parse featured (raw=%q): %w", body, err)
	}

	if out.URL == "" || out.Title == "" {
		return Featured{}, errors.New("featured missing title or url")
	}

	if out.Hook == "" {
		return Featured{}, errors.New("featured missing hook")
	}

	return Featured{
		Title:  out.Title,
		URL:    out.URL,
		Source: news.Source(out.Source),
		Tag:    news.Tag(out.Tag),
		Hook:   out.Hook,
	}, nil
}
