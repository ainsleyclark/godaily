// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rotation

import (
	"context"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
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
func CTA(ctx context.Context, p ai.Prompter, platform social.Platform, payload CTAPayload) (string, error) {
	return run(ctx, p, platform, ctaGuidance, payload)
}
