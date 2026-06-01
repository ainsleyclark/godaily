// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ai

import (
	"context"

	"github.com/ainsleyclark/godaily/pkg/ai/anthropic"
	"github.com/ainsleyclark/godaily/pkg/env"
)

// Client wraps the Anthropic Prompter.
type Client struct {
	primary Prompter
}

// New constructs a Client backed by Anthropic.
func New(cfg env.Config) *Client {
	return &Client{primary: anthropic.New(cfg.AnthropicAPIKey)}
}

// Prompt calls the Anthropic prompter with the given model, system, and user prompts.
func (c *Client) Prompt(ctx context.Context, model, system, user string) ([]byte, error) {
	return c.primary.Prompt(ctx, model, system, user)
}
