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

// MastodonHashtags is the canonical hashtag list appended to every Mastodon
// status.
var MastodonHashtags = []string{"#golang", "#go", "#programming"}

const mastodonCharLimit = 500

const mastodonGuidance = `- Mastodon users skew technical. The fediverse uses hashtags actively for discovery — keep them.
- Lead with the factual hook (Line 1). One or two short supporting lines for context.
- Drop the URL on its own line. Mastodon renders it as a clickable link.
- Strict structure (line breaks matter):
    Line 1: factual hook
    Line 2 (optional): one extra detail
    Line 3: blank
    Line 4: URL
    Line 5: hashtags from the list above
- 280-400 chars is the sweet spot.`

// Mastodon reframes the featured item as a Mastodon status.
func Mastodon(ctx context.Context, p ai.Prompter, f Featured) (string, error) {
	return reframe(ctx, p, platformConfig{
		name:      "Mastodon",
		charLimit: mastodonCharLimit,
		hashtags:  MastodonHashtags,
		guidance:  mastodonGuidance,
	}, f)
}
