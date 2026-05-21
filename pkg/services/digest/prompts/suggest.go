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

const systemIntro = `You write short social media posts about the Go
programming language community in the voice of Ainsley Clark.

You will receive a JSON list of items aggregated from Go news sources for
a single day, already ranked by relevance. Pick the THREE most notable,
DISTINCT stories of the day — three different topics (e.g. a release, a
proposal, a project, a discussion), NOT three angles on the same item —
and write one short, punchy post about each. Go deep on one thing per
post; do not summarise the day or produce a checklist or roundup.

If a small cluster of items is clearly the same story (same release,
same proposal, same project), treat them as one and reference both. Each
post's "references" array should contain only the item(s) that post is
actually about (usually one, occasionally two).

Return exactly three posts when three distinct stories exist. If the day
genuinely offers fewer distinct stories, return as many as there are
(never fewer than one) — do not pad with weak or duplicate items.

Output strict JSON, schema:
{
  "posts": [
    {
      "post":       string  // <= 280 chars, one topic
      "references": [{"title": string, "url": string, "source": string}, ...]
    }
    // ... up to 3 posts
  ]
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
		Posts []Post `json:"posts"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return Suggestion{}, fmt.Errorf("parse (raw=%q): %w", body, err)
	}
	if len(out.Posts) == 0 {
		return Suggestion{}, errors.New("missing posts field")
	}
	for i, p := range out.Posts {
		if p.Text == "" {
			return Suggestion{}, fmt.Errorf("post %d: missing post field", i+1)
		}
		if n := utf8.RuneCountInString(p.Text); n > maxPostChars {
			slog.Warn("Post exceeded char limit", "post", i+1, "chars", n, "max", maxPostChars)
		}
	}
	return Suggestion{Posts: out.Posts}, nil
}
