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

const linkedInGuidance = `- The audience is engineering leaders and senior Go developers. Slightly more elaborate than Bluesky/Mastodon, still no fluff.

## Structure (required, in this exact order)
1. Hook line: one factual sentence. Lead with the concrete thing - a name, a version, a date, a venue. Pack detail into the hook itself rather than saving it for the body.
2. Blank line.
3. Body: 2-3 sentences of plain prose. Must contain something a reader can't get from the link preview: a concrete change (what now exists that didn't), a "who this is for" line, a version/date/venue anchor, or a comparison to what came before.
4. Blank line.
5. URL on its own line, verbatim.
6. Blank line.
7. Hashtags from the list above on the final line.

## Voice
- Plain prose paragraphs. No bullet lists. No markdown. No first-person editorial.
- Do NOT start with "Exciting news", "Today in Go", "Check out", or any throat-clearing. The hook is the lede.
- If the body feels like padding, the hook is too thin: rewrite the hook with the missing detail rather than removing the body.

## Length
- Sweet spot 300-600 characters total. Under 200 characters means the body is missing - go back and add the concrete detail.

## Worked examples

Event:
GoLab 2026 lands in Bologna on November 18-20, co-located with RustLab.

Three days of Go talks alongside the Rust track in one venue, useful if your team is already running Go services next to Rust components and wants the same people in the same room. CFP and early-bird tickets are open now.

https://golab.io/

#golang #softwareengineering #programming

Release:
Go 1.24 ships generic type aliases and a faster map implementation built on Swiss tables.

Generic aliases close one of the last sharp edges in the generics design: library authors can now re-export parameterised types without leaking implementation. The new map is roughly 30% faster on lookup-heavy workloads, with no API change required.

https://go.dev/blog/go1.24

#golang #softwareengineering #programming`

// LinkedIn reframes the featured item as a LinkedIn organisation-page post.
func LinkedIn(ctx context.Context, p ai.Prompter, f Featured) (string, error) {
	return reframe(ctx, p, platformConfig{
		name:      "LinkedIn",
		charLimit: linkedInCharLimit,
		hashtags:  LinkedInHashtags,
		guidance:  linkedInGuidance,
	}, f)
}
