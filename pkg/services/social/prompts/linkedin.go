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

package prompts

import (
	"context"

	"github.com/ainsleyclark/godaily/pkg/ai"
)

// LinkedInHashtags is the canonical hashtag list appended to every LinkedIn
// post.
var LinkedInHashtags = []string{"#golang", "#softwareengineering", "#programming"}

const linkedInCharLimit = 1300

const linkedInGuidance = `- The audience is engineering leaders and senior developers. Slightly more elaborate than Bluesky/Mastodon, still no fluff.
- Open with the same factual hook. Then 1 short paragraph explaining WHY a Go-shop tech lead should care: what changes for their team, what risk it removes, what new capability it unlocks.
- Use plain prose paragraphs separated by a blank line. No bullet lists. No markdown.
- End with the URL on its own line, then a blank line, then the hashtags from the list above on the final line.
- Length sweet spot: 300-600 characters. Hard limit is much higher; do NOT pad.
- Do not start with "Exciting news" or any first-person editorial. The hook is the lede.`

// LinkedIn reframes the featured item as a LinkedIn organisation-page post.
func LinkedIn(ctx context.Context, p ai.Prompter, f Featured) (string, error) {
	return reframe(ctx, p, platformConfig{
		name:      "LinkedIn",
		charLimit: linkedInCharLimit,
		hashtags:  LinkedInHashtags,
		guidance:  linkedInGuidance,
	}, f)
}
