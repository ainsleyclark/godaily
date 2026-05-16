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

package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/pkg/errors"

	anthr "github.com/ainsleyclark/godaily/pkg/ai/anthropic"
	"github.com/ainsleyclark/godaily/pkg/news"
)

const maxTitleChars = 80

// DigestMeta is the structured output returned by Synthesise.
type DigestMeta struct {
	Title string `json:"title"` // ≤80 chars — email subject / card title
	Intro string `json:"intro"` // 1–2 sentence digest intro paragraph
}

// Synthesise filters the day's items, calls the model, and returns DigestMeta
// containing the email subject title and intro paragraph. ErrNoItems is returned
// (without making an API call) when there is nothing to synthesise.
func (c *Client) Synthesise(ctx context.Context, day time.Time, sections []news.SourceItems) (DigestMeta, error) {
	items := filterItems(sections, c.filter)
	if len(items) == 0 {
		return DigestMeta{}, ErrNoItems
	}

	user := buildUserPrompt(day, items)
	system := buildSystemText(buildDigestSystemBlocks())

	slog.InfoContext(ctx, "Requesting AI digest meta", "items", len(items))

	primary := anthr.New(c.anthropic, buildDigestSystemBlocks())
	raw, err := prompt(ctx, primary, c.fallback, system, user)
	if err != nil {
		return DigestMeta{}, errors.Wrap(err, "ai synthesise")
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

// parseDigestResponse extracts DigestMeta from the model's text blocks.
// Kept for existing tests.
func parseDigestResponse(m *anthropic.Message) (DigestMeta, error) {
	if m == nil {
		return DigestMeta{}, errors.New("nil message")
	}
	var raw strings.Builder
	for _, b := range m.Content {
		if b.Type == "text" {
			raw.WriteString(b.Text)
		}
	}
	return parseDigestBytes([]byte(raw.String()))
}
