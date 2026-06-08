// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rotation

import (
	"context"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

// RecapItem is one entry in the weekly recap input.
type RecapItem struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Source string `json:"source"`
	Clicks int64  `json:"clicks"`
}

// RecapPayload is the input to the recap prompt.
type RecapPayload struct {
	WeekLabel string      `json:"week_label"`
	Items     []RecapItem `json:"items"`
}

const recapGuidance = `You are writing the Monday weekly recap for GoDaily: the most-clicked Go-community stories from last week's daily digests.

The input is a list of items, already ranked by GoDaily subscriber clicks (highest first). The "week_label" is an ISO week identifier you can reference loosely (e.g. "last week").

Write ONE post that:
1. Opens with a single short line framing the recap ("Most-clicked Go stories last week:").
2. Lists each item as a short entry. Format: one tight descriptive phrase (not just the raw title), then the URL on the next line. No bullet characters, no numbers — line breaks alone. The phrase should tell the reader what it's actually about in plain language, so the post is worth reading even without clicking any link.
3. Does NOT name click counts. The fact these were popular is enough; numbers come across as boastful and date the post.
4. If only 1 or 2 items are supplied, write a shorter post — never pad.

The recap should read like a useful week-in-Go summary, not a link dump.

End the post body, then a blank line, then the platform's hashtag line.`

// Recap generates a weekly recap post for a single platform.
func Recap(ctx context.Context, p ai.Prompter, platform social.Platform, payload RecapPayload) (string, error) {
	return run(ctx, p, platform, recapGuidance, payload)
}
