// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package candidates

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/rotation"
	"github.com/ainsleyclark/godaily/pkg/utm"
)

// ctaCooldown is the lookback window for the CTA eligibility check. Using
// 13 days rather than 14 accounts for the inclusive HasPostedKindSince query:
// the rotation runs at a fixed time each Tuesday, so a post saved seconds
// after the previous run falls inside a strict 14-day window on the next
// Tuesday. 13 days ensures two consecutive Tuesdays (always exactly 14 days
// apart) are always eligible.
const ctaCooldown = 13 * 24 * time.Hour

// ctaAngles is the rotating list of framings for the signup CTA. Each
// angle is paired with the same target URL but produces a different post
// because the AI prompt hangs the copy on the angle. Add to this list to
// expand the rotation; the candidate picks deterministically.
//
// Angles lead with usefulness, not the ask. The subscription link is the
// natural follow-through for someone who finds the content worth reading.
var ctaAngles = []string{
	"We read 20+ Go sources every morning so you don't have to.",
	"One email a day, the best Go news from across the community. No fluff.",
	"The Go release cycle, proposal tracker, and community blogs — one place, once a day.",
	"Free, no spam, one email a day. The Go ecosystem distilled.",
	"If you keep meaning to catch up on Go news, GoDaily does the legwork.",
	"Proposals, releases, articles, conference news — daily, in one read.",
	"The part of your morning routine that pays off in the afternoon.",
}

// CTA posts a "sign up to GoDaily" call to action, rotating through a
// fixed set of angles so the feed reads differently each time.
type CTA struct {
	posts social.PostRepository
}

// NewCTA constructs the candidate.
func NewCTA(posts social.PostRepository) *CTA {
	return &CTA{posts: posts}
}

// Kind reports the candidate's SocialPostKind.
func (c *CTA) Kind() social.PostKind { return social.PostKindCTA }

// Eligible blocks if a CTA was posted to bluesky within the cooldown.
// The angle for this run is derived from the week number so the rotation
// stays stable across retries within the same week.
func (c *CTA) Eligible(ctx context.Context, now time.Time) (candidate.CandidateContext, bool, error) {
	since := now.UTC().Add(-ctaCooldown)
	posted, err := c.posts.HasPostedKindSince(ctx, c.Kind(), platformAnchor, since)
	if err != nil {
		return candidate.CandidateContext{}, false, errors.Wrap(err, "checking CTA cooldown")
	}
	if posted {
		return candidate.CandidateContext{}, false, nil
	}

	angle := pickAngle(now, ctaAngles)
	subject := "cta:" + angleKey(angle)

	return candidate.CandidateContext{
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
func (c *CTA) Generate(ctx context.Context, p ai.Prompter, platform social.Platform, cctx candidate.CandidateContext) (string, error) {
	payload, ok := cctx.Payload.(rotation.CTAPayload)
	if !ok {
		return "", errors.New("cta: payload missing")
	}
	// Tag per platform at generation time (the platform is unknown when
	// Eligible builds the payload) so Plausible can split CTA conversions
	// by the social channel that posted them.
	payload.URL = utm.Tag(payload.URL, "social-"+platform.String(), "social", "cta")
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
