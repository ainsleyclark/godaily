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

// Package synth turns a day's scored news into a short suggested social
// post by calling the Anthropic Messages API. It keeps input cheap by
// filtering to top-N items and caching the static system prompt (the
// embedded style guide) on the request.
package synth

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/news"
)

// Client wraps the Anthropic SDK with the prompt construction and
// filtering needed to draft social posts from a day's news.
type Client struct {
	anthropic anthropic.Client
	filter    filterConfig
}

// New constructs a Client using ANTHROPIC_API_KEY from the environment.
// Request options are forwarded to the SDK — tests pass
// option.WithBaseURL to redirect to an httptest.Server.
func New(opts ...option.RequestOption) *Client {
	return &Client{
		anthropic: anthropic.NewClient(opts...),
		filter:    defaultFilterConfig(),
	}
}

const (
	model       = anthropic.ModelClaudeSonnet4_6
	maxTokens   = int64(1024)
	temperature = 0.4
)

// Suggest filters the day's sections to top items, calls the model with
// a cached system prompt, and returns a parsed Suggestion. ErrNoItems
// is returned (without making an API call) when there is nothing to
// summarise. Token usage and model are logged for cost tracking.
func (c *Client) Suggest(ctx context.Context, day time.Time, sections []news.SourceItems) (Suggestion, error) {
	items := filterItems(sections, c.filter)
	if len(items) == 0 {
		return Suggestion{}, ErrNoItems
	}

	user := buildUserPrompt(day, items)

	slog.InfoContext(ctx, "Calling anthropic",
		"model", model,
		"items", len(items),
		"max_tokens", maxTokens,
	)

	resp, err := c.anthropic.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: anthropic.Float(temperature),
		System:      buildSystemBlocks(),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})
	if err != nil {
		return Suggestion{}, errors.Wrap(err, "anthropic")
	}

	slog.InfoContext(ctx, "Synth response",
		"model", resp.Model,
		"input_tokens", resp.Usage.InputTokens,
		"output_tokens", resp.Usage.OutputTokens,
		"cache_creation_tokens", resp.Usage.CacheCreationInputTokens,
		"cache_read_tokens", resp.Usage.CacheReadInputTokens,
	)

	sug, err := parseResponse(resp)
	if err != nil {
		return Suggestion{}, err
	}

	sug.Date = day
	return sug, nil
}
