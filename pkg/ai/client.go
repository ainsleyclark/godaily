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
	"log/slog"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/pkg/errors"

	anthr "github.com/ainsleyclark/godaily/pkg/ai/anthropic"
	"github.com/ainsleyclark/godaily/pkg/news"
)

// Client wraps AI provider(s) with prompt construction and filtering
// logic needed to draft social posts and digest metadata from a day's news.
type Client struct {
	anthropic anthropic.Client
	fallback  Prompter
	filter    filterConfig
}

// New constructs a Client using Anthropic as the sole AI provider.
// Additional request options are forwarded to the SDK — tests pass
// option.WithBaseURL to redirect to an httptest.Server.
func New(apiKey string, opts ...option.RequestOption) *Client {
	return &Client{
		anthropic: anthropic.NewClient(append([]option.RequestOption{option.WithAPIKey(apiKey)}, opts...)...),
		filter:    defaultFilterConfig(),
	}
}

// NewWithFallback constructs a Client that tries Anthropic first and falls
// back to the given Prompter on any error.
func NewWithFallback(apiKey string, fallback Prompter, opts ...option.RequestOption) *Client {
	c := New(apiKey, opts...)
	c.fallback = fallback
	return c
}

// Suggest filters the day's sections to top items, calls the primary AI
// provider (with optional fallback), and returns a parsed Suggestion.
// ErrNoItems is returned (without any API call) when there is nothing to summarise.
func (c *Client) Suggest(ctx context.Context, day time.Time, sections []news.SourceItems) (Suggestion, error) {
	items := filterItems(sections, c.filter)
	if len(items) == 0 {
		return Suggestion{}, ErrNoItems
	}

	user := buildUserPrompt(day, items)
	system := buildSystemText(buildSystemBlocks())

	slog.InfoContext(ctx, "Requesting AI suggestion", "items", len(items))

	primary := anthr.New(c.anthropic, buildSystemBlocks())
	raw, err := prompt(ctx, primary, c.fallback, system, user)
	if err != nil {
		return Suggestion{}, errors.Wrap(err, "ai suggest")
	}

	sug, err := parseSuggestionBytes(raw)
	if err != nil {
		return Suggestion{}, err
	}
	sug.Date = day
	return sug, nil
}
