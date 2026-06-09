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

const (
	// maxCandidates caps the number of items presented to the model. Set
	// generously: perSectionCap already prevents any one section from
	// dominating, so a higher ceiling simply gives the model every section's
	// top 2-3 items to choose from at a trivial token cost.
	maxCandidates = 18
	// perSectionCap limits how many items any one section may contribute to the
	// shortlist, so a proposal-heavy day cannot fill every slot and crowd out
	// the discussions, articles, and videos that often drive more conversation.
	perSectionCap = 3
)

// excludedSections never anchor a featured social post and are filtered out
// before the shortlist is built. Jobs and social posts are not reshare-worthy
// Go news; events, conferences, and meet-ups are calendar announcements rather
// than content that sparks discussion.
var excludedSections = map[news.Tag]bool{
	news.TagJobs:       true,
	news.TagSocial:     true,
	news.TagEvent:      true,
	news.TagConference: true,
}

// candidate is the wire shape sent to the model. No score is included on
// purpose: the model judges each item on its own merit (title, source, tag,
// snippet) rather than anchoring on a pre-computed, proposal-biased number.
type candidate struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Source  string `json:"source"`
	Tag     string `json:"tag"`
	Snippet string `json:"snippet,omitempty"`
}

// buildCandidates turns the issue's items into a deliberately diverse shortlist
// for the model to choose from. Rather than ranking categories against one
// another by the per-source relevance score — which is tuned for editorial
// authority and structurally favours Go proposals — it guarantees breadth:
// items are grouped by canonical section, the best-scored item of each section
// is taken first (round-robin), and no section may exceed perSectionCap. The
// per-source score is used only to pick the strongest representative *within* a
// section, never to rank one section above another.
//
// Sections in excludedSections (jobs, social, events, conferences) never reach
// the shortlist — they are not reshare-worthy conversation anchors.
func buildCandidates(items []news.Item) []candidate {
	bySection := make(map[news.Tag][]news.Item)
	for _, it := range items {
		sec := it.Tag.Section()
		if excludedSections[sec] {
			continue
		}
		bySection[sec] = append(bySection[sec], it)
	}
	for sec := range bySection {
		its := bySection[sec]
		sort.SliceStable(its, func(i, j int) bool {
			return its[i].Score > its[j].Score
		})
	}

	order := orderedSections(bySection)

	out := make([]candidate, 0, maxCandidates)
	for round := 0; round < perSectionCap && len(out) < maxCandidates; round++ {
		for _, sec := range order {
			its := bySection[sec]
			if round >= len(its) {
				continue
			}
			out = append(out, toCandidate(its[round]))
			if len(out) >= maxCandidates {
				break
			}
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

func toCandidate(it news.Item) candidate {
	return candidate{
		Title:   it.Title,
		URL:     it.URL,
		Source:  string(it.Source),
		Tag:     string(it.Tag),
		Snippet: it.Snippet,
	}
}

const featureSystem = `You select the single most reshare-worthy Go-community item
of the day to anchor a social media post.

You will receive a JSON list — a deliberately diverse shortlist spanning the
day's categories (releases, proposals, articles, tutorials, discussions, videos,
trending projects). The list order carries NO priority and no scores are given:
judge each item on its own merit.

Pick the one item a senior Go developer would be most likely to share AND that
would get the community talking. Weigh two things equally:

1. Substance: a concrete, real change or insight — a release, an accepted/shipped
   proposal, an in-depth article, a measurable result, a removed footgun.

2. Conversation value: how much the item would get Go developers discussing,
   debating, or passing it along. A well-argued opinion piece, a lively community
   discussion, or a recognised voice's take on the language's direction is a
   legitimate winner and can rightly beat a routine proposal or minor release.

Substance and conversation value are co-equal — neither outranks the other by
category. Do NOT default to a proposal or release just because it is "official";
choose what people would actually share and argue about today.

If multiple items cover the same topic (same release, same proposal, same
project), treat them as one and pick the canonical link.

Output strict JSON, schema:
{
  "title":  string,        // copied verbatim from the chosen item
  "url":    string,        // copied verbatim from the chosen item
  "source": string,        // copied verbatim from the chosen item
  "tag":    string,        // copied verbatim from the chosen item
  "hook":   string         // 1 short sentence, max 25 words, factual:
                           //   what specifically changed and why a Go dev
                           //   would care enough to share it. Name the
                           //   concrete thing: version, API, metric, date.
                           //   NO marketing language. NO emojis.
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
		"Date: %s\nCandidates (a diverse shortlist across categories; order is not a ranking):\n%s\n\nReturn the JSON object only.",
		day.Format("2006-01-02"), string(payload),
	)

	raw, err := p.Prompt(ctx, ai.ModelSonnet, featureSystem, user)
	if err != nil {
		return Featured{}, errors.Wrap(err, "ai")
	}

	return parseFeatured(raw)
}

func parseFeatured(raw []byte) (Featured, error) {
	body := aiutil.ExtractJSON(string(raw))
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
