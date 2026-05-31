// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
- The post must be worth reading even if the reader never clicks the link. The hook should give the concrete fact — version number, API name, the specific change — not just a vague prompt to click through.
- Lead with the factual hook (Line 1). One or two short supporting lines for context.
- Drop the URL on its own line. Mastodon renders it as a clickable link.
- Strict structure (line breaks matter):
    Line 1: factual hook (be specific — name the version, the change, the date)
    Line 2 (optional): one extra detail that adds context not visible from the link preview
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
