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

const digestSystemIntro = `You are an editor writing metadata for a daily Go programming language digest email.

You will receive a JSON list of items aggregated from Go news sources for a single day, already ranked by relevance.

Output strict JSON, schema:
{
  "title": string  // <=80 chars — punchy email subject line teaser drawn from the headline item (e.g. "Go 1.24 lands, goroutines got faster")
  "intro": string  // 1-2 plain sentences stating the key technical fact(s) from the headline item(s), for the top of the email body
}

Picking the headline item:
- Prefer high-signal Go sources for the headline: releases, security advisories, accepted/shipped/open proposals on golang/go, and the official Go blog. Pick the highest-ranked item from these when one is in the top few.
- Only headline a discussion thread (Reddit, Hacker News, Lobsters, golang-nuts) when no release, proposal, or official post is present in the top items.

Rules for the intro:
- FACTUAL ONLY. State the technical substance directly. Never describe a discussion, thread, or conversation — write what the content covers or what shipped.
- Do not use framing like "A post explores...", "A thread discusses...", "The conversation unpacks..." — report the fact itself.
- Use neutral verbs about the technical content: "covers", "explains", "ships", "proposes", "lands", "walks through".
- Do not begin with "Today" or the date. Write in present tense, active voice, no filler.
Output the JSON object alone. No prose, no markdown fences, no commentary.`

func buildDigestSystem() string {
	return digestSystemIntro + "\n\n## Style guide\n\n" + styleMD
}

// Synthesise builds the digest-meta prompt, calls p, and parses the response.
// ErrNoItems is returned (without calling p) when sections is empty.
func Synthesise(ctx context.Context, p ai.Prompter, day time.Time, sections []news.SourceItems) (DigestMeta, error) {
	items := filterItems(sections, defaultFilterConfig())
	if len(items) == 0 {
		return DigestMeta{}, ErrNoItems
	}
	user := buildUserPrompt(day, items)
	raw, err := p.Prompt(ctx, buildDigestSystem(), user)
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
