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

package featured

import (
	"context"

	"github.com/ainsleyclark/godaily/pkg/ai"
)

// BlueskyHashtags is the canonical hashtag list appended to every Bluesky
// post. Exposed as a variable so tests and the social service can read it.
var BlueskyHashtags = []string{"#golang"}

const blueskyCharLimit = 300

const blueskyGuidance = `- Bluesky users are heavily developer-focused. Speak like you're posting in a Go channel.
- Lead with the specific factual hook (Line 1). One supporting detail is welcome but not required.
- Bluesky linkifies bare URLs automatically: drop the URL on its own line, no markdown.
- Strict structure (line breaks matter):
    Line 1: factual hook (what shipped, who shipped it, what's notable)
    Line 2 (optional): one extra detail
    Line 3: blank
    Line 4: URL
    Line 5: hashtags from the list above
- Keep it tight. 280 chars is plenty; 200 is often better.`

// Bluesky reframes the featured item as a Bluesky post.
func Bluesky(ctx context.Context, p ai.Prompter, f Featured) (string, error) {
	return reframe(ctx, p, platformConfig{
		name:      "Bluesky",
		charLimit: blueskyCharLimit,
		hashtags:  BlueskyHashtags,
		guidance:  blueskyGuidance,
	}, f)
}
