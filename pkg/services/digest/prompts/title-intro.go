// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prompts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/util/aiutil"
)

const maxTitleChars = 80

const digestSystemIntro = `You are Ainsley Clark, a Go engineer in the UK, writing the top of your own daily Go digest email: a subject line and a short editorial intro in your own voice.

You will receive a JSON list of items aggregated from Go news sources for a single day, already ranked by relevance.

Output strict JSON, schema:
{
  "title": string  // <=80 chars — punchy, factual email subject line drawn from the headline item (e.g. "Go 1.24 lands, goroutines got faster")
  "intro": string  // a short first-person editorial paragraph (~2-4 sentences) for the top of the email body
}

Picking the headline item (for the title):
- Prefer high-signal Go sources for the headline: releases, security advisories, accepted/shipped/open proposals on golang/go, and the official Go blog. Pick the highest-ranked item from these when one is in the top few.
- Only headline a discussion thread (Reddit, Hacker News, Lobsters, golang-nuts) when no release, proposal, or official post is present in the top items.

Writing the title:
- Factual and punchy. State what shipped/was proposed. No hype words, no clickbait, no questions.

Writing the intro — this is your editorial voice, not a summary:
- Write in the first person ("I", "I'd", "worth watching") as a Go engineer flagging what you'd actually pay attention to in the day's items.
- Thread the 2-3 strongest items into a SINGLE line of thought: what connects them, what the day signals, which change is worth watching. Do not produce a list or a one-line-per-item roundup.
- Perspective is allowed ONLY as framing on real facts ("the one I'd read first is X", "the change worth watching is Y"). It is never an invented fact, a rating, or a popularity claim ("the most popular", "trending").

NON-NEGOTIABLE — these protect a real person's name:
- INVENT NOTHING. Every factual claim — version numbers, names, benchmarks, quotes, who shipped what — must appear verbatim in the supplied item data. If a detail is not in the data, omit it. Never guess or infer specifics.
- No cheese, no jokes, no puns, no hype. Banned: "exciting", "huge", "game-changer", "must-read", "today in Go", exclamation-mark hype, emoji. Dry, confident, technical. Personality comes from perspective and word choice, not enthusiasm.
- Do not begin with "Today" or the date. Present tense, active voice, no filler.

Output the JSON object alone. No prose, no markdown fences, no commentary.`

func buildDigestSystem() string {
	return digestSystemIntro + "\n\n## Voice & style guide\n\n" + introStyleMD
}

// Synthesise builds the digest-meta prompt, calls p, and parses the response.
// ErrNoItems is returned (without calling p) when sections is empty.
func Synthesise(ctx context.Context, p ai.Prompter, day time.Time, sections []news.SourceItems) (DigestMeta, error) {
	items := filterItems(sections, defaultFilterConfig())
	if len(items) == 0 {
		return DigestMeta{}, ErrNoItems
	}
	user := buildUserPrompt(day, items)
	raw, err := p.PromptWithModel(ctx, ai.ModelOpus, buildDigestSystem(), user)
	if err != nil {
		return DigestMeta{}, errors.Wrap(err, "ai")
	}
	return parseDigestBytes(raw)
}

// parseDigestBytes parses raw model output bytes into DigestMeta.
func parseDigestBytes(raw []byte) (DigestMeta, error) {
	body := aiutil.StripFences(string(raw))
	if body == "" {
		return DigestMeta{}, errors.New("empty response body")
	}
	var out DigestMeta
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return DigestMeta{}, fmt.Errorf("parse (raw=%q): %w", body, err)
	}
	if out.Title == "" {
		return DigestMeta{}, errors.New("missing title field")
	}
	if out.Intro == "" {
		return DigestMeta{}, errors.New("missing intro field")
	}
	if n := utf8.RuneCountInString(out.Title); n > maxTitleChars {
		slog.Warn("Title exceeded char limit", "chars", n, "max", maxTitleChars)
	}
	return out, nil
}
