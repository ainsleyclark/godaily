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
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/util/aiutil"
)

const maxPostChars = 280

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

func buildSuggestSystem() string {
	return systemIntro + "\n\n## Style guide\n\n" + styleMD
}

// Suggest builds the social-post prompt, calls p, and parses the response.
// ErrNoItems is returned (without calling p) when sections is empty.
func Suggest(ctx context.Context, p ai.Prompter, day time.Time, sections []news.SourceItems) (Suggestion, error) {
	items := filterItems(sections, defaultFilterConfig())
	if len(items) == 0 {
		return Suggestion{}, ErrNoItems
	}
	user := buildUserPrompt(day, items)
	raw, err := p.Prompt(ctx, buildSuggestSystem(), user)
	if err != nil {
		return Suggestion{}, errors.Wrap(err, "ai")
	}
	sug, err := parseSuggestionBytes(raw)
	if err != nil {
		return Suggestion{}, err
	}
	sug.Date = day
	return sug, nil
}

// parseSuggestionBytes parses raw model output bytes into a Suggestion.
func parseSuggestionBytes(raw []byte) (Suggestion, error) {
	body := aiutil.StripFences(string(raw))
	if body == "" {
		return Suggestion{}, errors.New("empty response body")
	}
	var out struct {
		Post       string `json:"post"`
		References []Ref  `json:"references"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return Suggestion{}, fmt.Errorf("parse (raw=%q): %w", body, err)
	}
	if out.Post == "" {
		return Suggestion{}, errors.New("missing post field")
	}
	if n := utf8.RuneCountInString(out.Post); n > maxPostChars {
		slog.Warn("Post exceeded char limit", "chars", n, "max", maxPostChars)
	}
	return Suggestion{Post: out.Post, References: out.References}, nil
}
