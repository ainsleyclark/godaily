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
	"github.com/ainsleyclark/godaily/pkg/gateway/social"
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

const recapGuidance = `You are writing the Friday weekly recap for GoDaily: the most-clicked Go-community stories from this week's daily digests.

The input is a list of items, already ranked by GoDaily subscriber clicks (highest first). The "week_label" is an ISO week identifier you can reference loosely (e.g. "this week").

Write ONE post that:
1. Opens with a single short line framing the recap ("Most-clicked Go stories from GoDaily subscribers this week:").
2. Lists each item as a separate line: short title, then the URL. No bullet characters, no numbered list — line breaks alone. Keep titles tight; trim if they're long.
3. Does NOT name click counts. The fact these were popular is enough; numbers come across as boastful and date the post.
4. If only 1 or 2 items are supplied, write a shorter post — never pad.

End the post body, then a blank line, then the platform's hashtag line.`

// Recap generates a weekly recap post for a single platform.
func Recap(ctx context.Context, p ai.Prompter, platform social.Platform, payload RecapPayload) (string, error) {
	return run(ctx, p, platform, recapGuidance, payload)
}
