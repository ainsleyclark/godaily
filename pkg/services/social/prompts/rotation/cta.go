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

// CTAPayload is the input to the signup-CTA prompt: an angle to anchor
// the call to action so each variant feels different, plus the signup URL.
type CTAPayload struct {
	Angle string `json:"angle"`
	URL   string `json:"url"`
}

const ctaGuidance = `You are writing a sign-up call-to-action for GoDaily — a daily email digest of the best Go-community news.

The input "angle" is the framing the post should hang on (e.g. "save 30 minutes of feed-scrolling", "we read 20+ sources so you don't have to", "free, no spam, one email a day"). Use the angle naturally; don't quote it.

Write ONE post that:
1. Names the value in a single concrete sentence built on the angle.
2. Names what GoDaily is, plainly: a daily email of the best Go news.
3. Ends with the URL on its own line.

Keep it human and low-pressure. No "join the community" boilerplate, no hype, no exclamation marks. Sound like a recommendation from a friend, not an ad.`

// CTA generates a signup CTA post for a single platform.
func CTA(ctx context.Context, p ai.Prompter, platform platform.Name, payload CTAPayload) (string, error) {
	return run(ctx, p, platform, ctaGuidance, payload)
}
