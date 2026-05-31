// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package anthropic provides an ai.Prompter implementation backed by the
// Anthropic Messages API via the official anthropic-sdk-go SDK.
package anthropic

import (
	"context"
	"log/slog"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/pkg/errors"
)

const (
	defaultModel = anthropic.ModelClaudeSonnet4_6
	maxTokens    = int64(1024)
	temperature  = 0.4
)

// Client satisfies ai.Prompter using the Anthropic Messages API.
type Client struct {
	client anthropic.Client
}

// New constructs a Client initialising the Anthropic SDK internally.
func New(apiKey string, opts ...option.RequestOption) *Client {
	allOpts := append([]option.RequestOption{option.WithAPIKey(apiKey)}, opts...)
	return &Client{client: anthropic.NewClient(allOpts...)}
}

// Prompt sends system as a single TextBlockParam and user as the user message,
// using the given model (defaulting to Sonnet when empty). Returns the
// concatenated text content bytes of the response.
func (c *Client) Prompt(ctx context.Context, model, system, user string) ([]byte, error) {
	m := anthropic.Model(model)
	if model == "" {
		m = defaultModel
	}

	params := anthropic.MessageNewParams{
		Model:     m,
		MaxTokens: maxTokens,
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	}
	// Opus rejects sampling parameters; steer it via the prompt instead. Only
	// the Sonnet-class default takes an explicit temperature.
	if !strings.HasPrefix(string(m), "claude-opus") {
		params.Temperature = anthropic.Float(temperature)
	}

	slog.InfoContext(
		ctx, "Calling Anthropic",
		"model", m,
		"max_tokens", maxTokens,
	)
	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "anthropic")
	}
	slog.InfoContext(
		ctx, "Anthropic response",
		"model", resp.Model,
		"input_tokens", resp.Usage.InputTokens,
		"output_tokens", resp.Usage.OutputTokens,
		"cache_creation_tokens", resp.Usage.CacheCreationInputTokens,
		"cache_read_tokens", resp.Usage.CacheReadInputTokens,
	)
	var out strings.Builder
	for _, b := range resp.Content {
		if b.Type == "text" {
			out.WriteString(b.Text)
		}
	}
	return []byte(out.String()), nil
}
