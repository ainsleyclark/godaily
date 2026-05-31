// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ainsleyclark/godaily/pkg/ai/anthropic"
	"github.com/ainsleyclark/godaily/pkg/ai/gemini"
	"github.com/ainsleyclark/godaily/pkg/env"
)

// notifier posts an AI-provider comparison to a chat channel.
// It is satisfied by *slack.Client.
type notifier interface {
	MustSend(ctx context.Context, message string)
}

// Client chains a primary Prompter with an optional fallback.
// It satisfies Prompter itself so it can be composed freely.
type Client struct {
	primary  Prompter
	fallback Prompter
	notifier notifier
}

// New constructs a Client from config, using Anthropic as the primary
// Prompter and Gemini as an optional fallback when GeminiAPIKey is set.
// n receives a side-by-side comparison of both providers' output.
func New(cfg env.Config, n notifier) *Client {
	primary := anthropic.New(cfg.AnthropicAPIKey)
	var fallback Prompter
	if cfg.GeminiAPIKey != "" {
		g, err := gemini.New(cfg.GeminiAPIKey)
		if err != nil {
			slog.Warn("Gemini client initialisation failed, fallback disabled", "err", err)
		} else {
			fallback = g
		}
	}
	return &Client{primary: primary, fallback: fallback, notifier: n}
}

// Prompt calls the primary (Anthropic) and fallback (Gemini) prompters with
// the given model, posts a side-by-side comparison of their output to Slack,
// then returns the primary's result. The fallback's result is used only when
// the primary fails.
//
// TODO: the dual-call comparison is temporary, kept while evaluating whether
// Gemini can replace Anthropic. Drop it once a single provider is chosen.
func (c *Client) Prompt(ctx context.Context, model, system, user string) ([]byte, error) {
	primaryRaw, primaryErr := c.primary.Prompt(ctx, model, system, user)

	var (
		fallbackRaw []byte
		fallbackErr error
	)
	if c.fallback != nil {
		fallbackRaw, fallbackErr = c.fallback.Prompt(ctx, geminiModelFor(model), system, user)
	}

	c.notifyComparison(ctx, primaryRaw, primaryErr, fallbackRaw, fallbackErr)

	if primaryErr == nil {
		return primaryRaw, nil
	}
	if c.fallback == nil {
		return nil, primaryErr
	}
	slog.WarnContext(ctx, "Primary AI call failed, using fallback result", "err", primaryErr)
	if fallbackErr != nil {
		return nil, fallbackErr
	}
	return fallbackRaw, nil
}

// notifyComparison posts both providers' output to Slack so they can be
// reviewed side by side. It is a no-op when no notifier is configured.
func (c *Client) notifyComparison(ctx context.Context, primaryRaw []byte, primaryErr error, fallbackRaw []byte, fallbackErr error) {
	if c.notifier == nil {
		return
	}
	var b strings.Builder
	b.WriteString("AI provider comparison\n\nAnthropic (primary):\n")
	if primaryErr != nil {
		b.WriteString("error: " + primaryErr.Error())
	} else {
		b.WriteString(renderForSlack(primaryRaw))
	}
	b.WriteString("\n\nGemini (fallback):\n")
	switch {
	case c.fallback == nil:
		b.WriteString("not configured")
	case fallbackErr != nil:
		b.WriteString("error: " + fallbackErr.Error())
	default:
		b.WriteString(renderForSlack(fallbackRaw))
	}
	c.notifier.MustSend(ctx, b.String())
}

// renderForSlack extracts human-readable string fields from a JSON AI response.
// It surfaces the known output fields (post, title, intro) in a readable format
// and falls back to the raw string if the bytes are not valid JSON.
func renderForSlack(raw []byte) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return string(raw)
	}
	var b strings.Builder
	for _, key := range []string{"title", "intro", "post"} {
		v, ok := m[key]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			fmt.Fprintf(&b, "*%s:* %s\n", key, s)
		}
	}
	if b.Len() == 0 {
		return string(raw)
	}
	return strings.TrimRight(b.String(), "\n")
}
