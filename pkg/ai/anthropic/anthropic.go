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

// Package anthropic provides an ai.Prompter implementation backed by the
// Anthropic Messages API via the official anthropic-sdk-go SDK.
package anthropic

import (
	"context"
	"log/slog"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/pkg/errors"
)

const (
	model       = anthropic.ModelClaudeSonnet4_6
	maxTokens   = int64(1024)
	temperature = 0.4
)

// Client satisfies ai.Prompter using the Anthropic Messages API.
// The system string argument to Prompt is unused — the pre-built cached
// TextBlockParams stored at construction time are used instead.
type Client struct {
	client anthropic.Client
	system []anthropic.TextBlockParam
}

// New constructs a Client with a pre-initialised SDK client and system blocks.
// The SDK client is initialised once by the caller (ai.Client constructor) and
// shared across calls for connection reuse.
func New(sdkClient anthropic.Client, system []anthropic.TextBlockParam) *Client {
	return &Client{client: sdkClient, system: system}
}

// Prompt sends user to the Anthropic Messages API with the stored system
// blocks. The system string argument is ignored (see type doc). Returns the
// concatenated text content bytes of the response.
func (c *Client) Prompt(ctx context.Context, _, user string) ([]byte, error) {
	slog.InfoContext(ctx, "Calling Anthropic",
		"model", model,
		"max_tokens", maxTokens,
	)
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: anthropic.Float(temperature),
		System:      c.system,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "anthropic")
	}
	slog.InfoContext(ctx, "Anthropic response",
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
