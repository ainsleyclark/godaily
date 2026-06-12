// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rotation

import (
	"context"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

// NewSourcePayload is the input to the new-source announcement prompt:
// the source we just started pulling from, with its mention/url/blurb.
type NewSourcePayload struct {
	DisplayName string `json:"display_name"`
	Mention     string `json:"mention"`
	Blurb       string `json:"blurb"`
	URL         string `json:"url"`
}

const newSourceGuidance = `You are announcing that GoDaily has added a new source to its daily digest. The point of the post is to tell subscribers (and the source's creator) that their stuff now flows into GoDaily, and to give a one-line reason why a Go dev should care about this source.

Inputs include a display name, a platform-specific handle ("mention"), a one-sentence blurb about the source, and the source's URL.

Mention handling:
- If the "mention" field starts with "@", use it verbatim — it is the source's platform handle.
- If the "mention" field does NOT start with "@", it is a plain display name. Use it naturally in the sentence. Do not add an "@" prefix or attempt to create a social handle.

Write ONE post that:
1. Says GoDaily pulls from this source.
2. References the source using the mention field (per the rules above).
3. Uses one line adapted from the blurb to explain why it's worth following.
4. Includes the source URL on its own line.

Tone is collegial — you're crediting a source and pointing readers at them, not selling. Do not say "we're excited" or similar.`

// NewSource generates a "GoDaily pulls from X" post for one platform.
func NewSource(ctx context.Context, p ai.Prompter, platform social.Platform, payload NewSourcePayload) (string, error) {
	return run(ctx, p, platform, newSourceGuidance, payload)
}
