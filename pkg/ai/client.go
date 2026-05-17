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
)

// Client chains a primary Prompter with an optional fallback.
// It satisfies Prompter itself so it can be composed freely.
type Client struct {
	primary  Prompter
	fallback Prompter
}

// New constructs a Client. Pass nil fallback to disable fallback.
func New(primary Prompter, fallback Prompter) *Client {
	return &Client{primary: primary, fallback: fallback}
}

// Prompt calls primary; on error, logs a warning and tries fallback (if set).
func (c *Client) Prompt(ctx context.Context, system, user string) ([]byte, error) {
	raw, err := c.primary.Prompt(ctx, system, user)
	if err == nil {
		return raw, nil
	}
	if c.fallback == nil {
		return nil, err
	}
	slog.WarnContext(ctx, "Primary AI call failed, trying fallback", "err", err)
	return c.fallback.Prompt(ctx, system, user)
}
