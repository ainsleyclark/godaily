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

package candidates

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/rotation"
)

// ctaCooldown is the minimum gap between two signup CTAs. Posting more
// than once a week feels spammy and trains followers to scroll past.
const ctaCooldown = 7 * 24 * time.Hour

// ctaAngles is the rotating list of framings for the signup CTA. Each
// angle is paired with the same target URL but produces a different post
// because the AI prompt hangs the copy on the angle. Add to this list to
// expand the rotation; the candidate picks deterministically.
var ctaAngles = []string{
	"We read 20+ Go sources every morning so you don't have to.",
	"One email a day, the best Go news from across the community. No fluff.",
	"Save the half-hour of feed-scrolling. Get a curated Go digest in your inbox.",
	"Free, no spam, one email a day. The Go ecosystem distilled.",
	"If you keep meaning to catch up on Go news, GoDaily does the legwork.",
}

// CTA posts a "sign up to GoDaily" call to action, rotating through a
// fixed set of angles so the feed reads differently each time.
type CTA struct {
	posts news.SocialPostRepository
}

// NewCTA constructs the candidate.
func NewCTA(posts news.SocialPostRepository) *CTA {
	return &CTA{posts: posts}
}

// Kind reports the candidate's SocialPostKind.
func (c *CTA) Kind() news.SocialPostKind { return news.SocialPostKindCTA }

// Eligible blocks if a CTA was posted to bluesky within the cooldown.
// The angle for this run is derived from the week number so the rotation
// stays stable across retries within the same week.
func (c *CTA) Eligible(ctx context.Context, now time.Time) (socialsvc.CandidateContext, bool, error) {
	since := now.UTC().Add(-ctaCooldown)
	posted, err := c.posts.HasPostedKindSince(ctx, c.Kind(), platformAnchor, since)
	if err != nil {
		return socialsvc.CandidateContext{}, false, errors.Wrap(err, "checking CTA cooldown")
	}
	if posted {
		return socialsvc.CandidateContext{}, false, nil
	}

	angle := pickAngle(now, ctaAngles)
	subject := "cta:" + angleKey(angle)

	return socialsvc.CandidateContext{
		Kind:    c.Kind(),
		Subject: subject,
		URL:     env.AppURL + "/",
		Payload: rotation.CTAPayload{
			Angle: angle,
			URL:   env.AppURL + "/",
		},
	}, true, nil
}

// Generate dispatches to the cta prompt.
func (c *CTA) Generate(ctx context.Context, p ai.Prompter, platform socialgw.Platform, cctx socialsvc.CandidateContext) (string, error) {
	payload, ok := cctx.Payload.(rotation.CTAPayload)
	if !ok {
		return "", errors.New("cta: payload missing")
	}
	return rotation.CTA(ctx, p, platform, payload)
}

// pickAngle picks a deterministic angle for now's ISO week. Two CTAs in
// the same week (which would only happen via manual triggering, since
// the cooldown blocks the automatic path) get the same angle, so a forced
// re-run never spits out a different framing for the same week.
func pickAngle(now time.Time, angles []string) string {
	year, week := now.UTC().ISOWeek()
	h := fnv.New32a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d-W%02d", year, week)))
	return angles[int(h.Sum32())%len(angles)]
}

func angleKey(angle string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(angle))
	return fmt.Sprintf("%d", h.Sum32()%1000)
}
