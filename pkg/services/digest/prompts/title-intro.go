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
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/news"
)

const maxTitleChars = 80

const digestSystemIntro = `You are an editor writing metadata for a daily Go programming language digest email.

You will receive a JSON list of items aggregated from Go news sources for a single day, already ranked by relevance.

Output strict JSON, schema:
{
  "title": string  // <=80 chars — punchy email subject line teaser drawn from the top item (e.g. "Go 1.24 lands, goroutines got faster")
  "intro": string  // 1-2 plain sentences summarising what mattered most today, for the top of the email body
}

Do not begin the intro with "Today" or the date. Write in present tense, active voice, no filler.
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
	body := stripFences(string(raw))
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
