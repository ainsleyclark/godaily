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
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/news"
)

const maxPostChars = 280

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
	body := stripFences(string(raw))
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

// stripFences defensively removes a wrapping ```json ... ``` (or plain
// ``` ... ```) block if the model emits one despite being told not to.
// Anything outside the fence is discarded.
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[i+1:]
	} else {
		return s
	}
	if j := strings.LastIndex(s, "```"); j >= 0 {
		s = s[:j]
	}
	return strings.TrimSpace(s)
}
