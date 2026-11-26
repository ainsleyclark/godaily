// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rotation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/compose"
	socialprompts "github.com/ainsleyclark/godaily/pkg/services/social/prompts"
	"github.com/ainsleyclark/godaily/pkg/util/aiutil"
)

// RecapItem is one entry in the weekly recap input.
type RecapItem struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Source string `json:"source"`
	Clicks int64  `json:"clicks"`
}

// RecapPayload is the input to the recap prompt.
type RecapPayload struct {
	WeekLabel string      `json:"week_label"`
	Items     []RecapItem `json:"items"`
}

// defaultRecapIntro is used when the model returns no intro line.
const defaultRecapIntro = "Most-clicked Go stories last week:"

// recapPhraseLimits caps each item's descriptive phrase per platform so a
// full recap (intro, items, hashtags and Subscribe line) fits the
// platform's limit without cutting anything mid-sentence.
var recapPhraseLimits = map[social.Platform]int{
	social.Bluesky:  45,
	social.LinkedIn: 140,
	social.Mastodon: 70,
}

const recapGuidance = `You are writing the Monday weekly recap for GoDaily: the most-clicked Go-community stories from last week's daily digests.

The input is a list of items, already ranked by GoDaily subscriber clicks (highest first). The "week_label" is an ISO week identifier you can reference loosely (e.g. "last week").

You write the words; the post itself (links, hashtags, Subscribe line) is assembled for you. Provide:
1. "intro": a single short line framing the recap, e.g. "Most-clicked Go stories last week:". At most 40 characters.
2. "phrases": exactly one entry per input item, in the same order. Each is one tight descriptive phrase (not just the raw title) that tells the reader what the item is actually about in plain language, so the post is worth reading even without clicking any link. At most %d characters each. A complete phrase, never cut off. No URLs, no bullet characters, no numbers, no hashtags.

Do NOT name click counts. The fact these were popular is enough; numbers come across as boastful and date the post.

The recap should read like a useful week-in-Go summary, not a link dump.`

// Recap generates a weekly recap post for a single platform.
//
// The model only writes the intro and one phrase per item; the post is
// assembled here so the URLs are exactly the ones supplied, the
// Subscribe line is always present, and the post fits the platform's
// limit by dropping the lowest-ranked items rather than truncating
// mid-sentence.
func Recap(ctx context.Context, p ai.Prompter, platform social.Platform, payload RecapPayload) (string, error) {
	if p == nil {
		return "", errors.New("rotation: ai.Prompter is required")
	}
	cfg, ok := platformProfiles[platform]
	if !ok {
		return "", errors.Errorf("rotation: unknown platform %q", platform)
	}
	if len(payload.Items) == 0 {
		return "", errors.New("rotation: recap has no items")
	}

	user, err := buildUser(payload)
	if err != nil {
		return "", err
	}

	raw, err := p.Prompt(ctx, ai.ModelSonnet, recapSystem(cfg, recapPhraseLimits[platform]), user)
	if err != nil {
		return "", errors.Wrap(err, "ai")
	}

	intro, phrases, err := parseRecapResponse(raw)
	if err != nil {
		return "", err
	}

	return assembleRecap(platform, cfg.hashtags, intro, phrases, payload.Items), nil
}

// recapSystem builds the recap system prompt. It does not reuse
// assembleSystem because the model returns structured parts, not a
// finished post.
func recapSystem(cfg platformProfile, phraseLimit int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You write the copy for one social media post for %s in the voice of Ainsley Clark, a Go engineer.\n\n", cfg.name)
	b.WriteString(socialprompts.BrandRules)
	b.WriteString("\n")
	b.WriteString("## Platform guidance\n")
	b.WriteString(cfg.guidance)
	b.WriteString("\n\n")
	b.WriteString("## This post\n")
	fmt.Fprintf(&b, recapGuidance, phraseLimit)
	b.WriteString("\n\n")
	b.WriteString("## Output format\n")
	b.WriteString("Output strict JSON, schema:\n{\n  \"intro\": string,\n  \"phrases\": [string]   // one per input item, same order\n}\n\n")
	b.WriteString("Output the JSON object alone. No prose, no markdown fences.")
	return b.String()
}

func parseRecapResponse(raw []byte) (string, []string, error) {
	body := aiutil.ExtractJSON(string(raw))
	if body == "" {
		return "", nil, errors.New("empty recap response")
	}
	var out struct {
		Intro   string   `json:"intro"`
		Phrases []string `json:"phrases"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return "", nil, fmt.Errorf("parse recap post (raw=%q): %w", body, err)
	}
	return out.Intro, out.Phrases, nil
}

// assembleRecap lays the recap out as:
//
//	intro
//
//	phrase
//	url
//
//	...
//
//	#hashtags
//
//	Subscribe: https://godaily.dev/?utm_...
//
// Items are dropped from the bottom of the ranking until the post fits
// the platform's limit. Items missing a phrase fall back to their title.
func assembleRecap(platform social.Platform, hashtags []string, intro string, phrases []string, items []RecapItem) string {
	intro = singleLine(intro)
	if intro == "" {
		intro = defaultRecapIntro
	}

	entries := make([]string, 0, len(items))
	for i, it := range items {
		phrase := ""
		if i < len(phrases) {
			phrase = singleLine(phrases[i])
		}
		if phrase == "" {
			phrase = singleLine(it.Title)
		}
		entries = append(entries, phrase+"\n"+it.URL)
	}

	tags := strings.Join(hashtags, " ")
	footer := compose.SubscribeLine(platform, string(social.PostKindRecap))
	build := func(n int) string {
		parts := append([]string{intro}, entries[:n]...)
		if tags != "" {
			parts = append(parts, tags)
		}
		parts = append(parts, footer)
		return strings.Join(parts, "\n\n")
	}

	limit := compose.Limit(platform)
	for n := len(entries); n >= 1; n-- {
		post := build(n)
		if limit == 0 || compose.Length(platform, post) <= limit {
			if n < len(entries) {
				slog.Info("Recap trimmed to fit platform limit",
					"platform", platform, "items", n, "of", len(entries))
			}
			return post
		}
	}

	// Even a single item does not fit: keep the footer and trim the rest.
	body := strings.TrimSuffix(build(1), "\n\n"+footer)
	return compose.WithFooter(platform, body, footer)
}

// singleLine collapses whitespace (including newlines) so a phrase can't
// break the post's line structure.
func singleLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
