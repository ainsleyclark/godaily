// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ai

import (
	"context"

	"github.com/ainsleyclark/godaily/pkg/ai/anthropic"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/util/aiutil"
)

// Client wraps the Anthropic Prompter.
type Client struct {
	primary Prompter
}

// New constructs a Client backed by Anthropic.
func New(cfg env.Config) *Client {
	return &Client{primary: anthropic.New(cfg.AnthropicAPIKey)}
}

// Prompt calls the Anthropic prompter with the given model, system, and user
// prompts, then strips off-brand em dashes from the response so no
// AI-generated content — social posts or digest copy — ever ships with one.
//
// Sanitising the raw bytes here, at the single point every model response
// flows through, is the brand guarantee for all generated text; doing it
// per-caller proved too easy to forget. Replacing in the raw bytes is safe:
// an em dash is never JSON structure, and downstream parsers run afterwards.
func (c *Client) Prompt(ctx context.Context, model, system, user string) ([]byte, error) {
	raw, err := c.primary.Prompt(ctx, model, system, user)
	if err != nil {
		return nil, err
	}
	return []byte(aiutil.SanitisePost(string(raw))), nil
}
