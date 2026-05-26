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

package rotation

import (
	"context"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

// SpotlightPayload is the input to the source-spotlight prompt: the
// source we're shouting out, plus its platform-formatted mention so the
// model includes it verbatim.
type SpotlightPayload struct {
	DisplayName string `json:"display_name"`
	Mention     string `json:"mention"`
	Blurb       string `json:"blurb"`
	URL         string `json:"url"`
}

const spotlightGuidance = `You are giving a shout-out to one of GoDaily's curated sources to thank them for great Go content and to drive followers their way.

Inputs include a hand-written 1-sentence blurb (use it verbatim or lightly adapted), a display name, a platform-specific handle ("mention"), and the source's URL.

Write ONE post that:
1. Mentions the source by their platform handle (the "mention" field) verbatim — it's already in the right syntax. On platforms where mention falls back to a plain name, just use the name.
2. Says one specific thing about why they're worth following — adapt the blurb, do not just quote it.
3. Includes the URL on its own line near the end.

The point is goodwill and discovery, not a hard sell. Sound like one engineer recommending another, not a marketing campaign.`

// Spotlight generates a source-spotlight post for a single platform.
func Spotlight(ctx context.Context, p ai.Prompter, platform social.Platform, payload SpotlightPayload) (string, error) {
	return run(ctx, p, platform, spotlightGuidance, payload)
}
