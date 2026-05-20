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
	"strings"
	"sync"

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
		fallback = gemini.New(cfg.GeminiAPIKey)
	}
	return &Client{primary: primary, fallback: fallback, notifier: n}
}

// Prompt calls the primary (Anthropic) and fallback (Gemini) prompters in
// parallel, posts a side-by-side comparison of their output to Slack, then
// returns the primary's result. The fallback's result is used only when the
// primary fails.
//
// TODO: the dual-call comparison is temporary, kept while evaluating whether
// Gemini can replace Anthropic. Drop it once a single provider is chosen.
func (c *Client) Prompt(ctx context.Context, system, user string) ([]byte, error) {
	var (
		wg          sync.WaitGroup
		primaryRaw  []byte
		primaryErr  error
		fallbackRaw []byte
		fallbackErr error
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		primaryRaw, primaryErr = c.primary.Prompt(ctx, system, user)
	}()

	if c.fallback != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fallbackRaw, fallbackErr = c.fallback.Prompt(ctx, system, user)
		}()
	}
	wg.Wait()

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
		b.Write(primaryRaw)
	}
	b.WriteString("\n\nGemini (fallback):\n")
	switch {
	case c.fallback == nil:
		b.WriteString("not configured")
	case fallbackErr != nil:
		b.WriteString("error: " + fallbackErr.Error())
	default:
		b.Write(fallbackRaw)
	}
	c.notifier.MustSend(ctx, b.String())
}
