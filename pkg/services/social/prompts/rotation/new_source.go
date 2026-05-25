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
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
)

// NewSourcePayload is the input to the new-source announcement prompt:
// the source we just started pulling from, with its mention/url/blurb.
type NewSourcePayload struct {
	DisplayName string `json:"display_name"`
	Mention     string `json:"mention"`
	Blurb       string `json:"blurb"`
	URL         string `json:"url"`
}

const newSourceGuidance = `You are announcing that GoDaily has added a new source to its daily digest. The point of the post is to tell subscribers (and the source's creator) that their stuff now flows into GoDaily, and to give a one-line reason why a Go dev should care about this source.

Inputs include a display name, a platform-specific handle ("mention") — use it verbatim where present, or fall back to the display name — a one-sentence blurb about the source, and the source's URL.

Write ONE post that:
1. Says GoDaily now pulls from this source.
2. Tags the source by their mention (handle) so they see it.
3. Uses one line adapted from the blurb to explain why it's worth following.
4. Includes the source URL on its own line.

Tone is collegial — you're crediting a source and pointing readers at them, not selling. Do not say "we're excited" or similar.`

// NewSource generates a "GoDaily now pulls from X" post for one platform.
func NewSource(ctx context.Context, p ai.Prompter, platform platform.Name, payload NewSourcePayload) (string, error) {
	return run(ctx, p, platform, newSourceGuidance, payload)
}
