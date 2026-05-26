// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package featured

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
