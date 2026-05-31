// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
- The post must be worth reading even if the reader never clicks the link. The hook (Line 1) should give them the concrete fact — version number, API name, the specific change — not just "something shipped".
- Lead with the specific factual hook (Line 1). One supporting detail is welcome but not required.
- Bluesky linkifies bare URLs automatically: drop the URL on its own line, no markdown.
- Strict structure (line breaks matter):
    Line 1: factual hook (what shipped, who shipped it, what's notable — be specific)
    Line 2 (optional): one extra detail that adds context not visible from the link preview
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
