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

// SelfReleasePayload is the input to the self-release prompt: the new
// GoDaily GitHub release we want to talk about.
type SelfReleasePayload struct {
	Tag         string `json:"tag"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
}

const selfReleaseGuidance = `You are announcing a new release of GoDaily, the daily Go-community email digest at https://godaily.dev.

The release tag and notes are in the input. Write ONE post that:
1. Names the version (e.g. "GoDaily v1.4 is out").
2. Surfaces the most user-facing thing from the release notes in one sentence. If the notes are vague (chore/refactor/CI), say something honest like "internals tidy-up — no behaviour change for subscribers" and stop there.
3. Always includes the release URL on its own line.

If the release body is empty or trivial, do NOT invent features. Keep it short.
Do not pretend the release is more dramatic than it is.`

// SelfRelease generates a self-release post for a single platform.
func SelfRelease(ctx context.Context, p ai.Prompter, platform social.Platform, payload SelfReleasePayload) (string, error) {
	return run(ctx, p, platform, selfReleaseGuidance, payload)
}
