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
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	"github.com/ainsleyclark/godaily/pkg/services/digest"
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/rotation"
)

// recapMinItems is the floor for posting a recap. Three items reads as a
// recap; one or two reads as "we picked our favourite", which the daily
// featured slot already does.
const recapMinItems = 3

// recapCooldown prevents two recaps in the same calendar week even if the
// cron fires twice (manual trigger + scheduled).
const recapCooldown = 6 * 24 * time.Hour

// Recap posts the top-clicked items of the current ISO week. Delegates
// all dataset computation to pkg/services/digest so the same machinery
// can be reused by email outros, web pages, RSS, etc.
type Recap struct {
	recap *digest.RecapService
	posts social.PostRepository
}

// NewRecap constructs the candidate.
func NewRecap(svc *digest.RecapService, posts social.PostRepository) *Recap {
	return &Recap{recap: svc, posts: posts}
}

// Kind reports the candidate's SocialPostKind.
func (c *Recap) Kind() social.PostKind { return social.PostKindRecap }

// Eligible blocks if a recap was already posted within the cooldown, and
// requires the recap dataset to contain at least recapMinItems entries.
func (c *Recap) Eligible(ctx context.Context, now time.Time) (socialsvc.CandidateContext, bool, error) {
	if c.recap == nil {
		return socialsvc.CandidateContext{}, false, nil
	}

	since := now.UTC().Add(-recapCooldown)
	posted, err := c.posts.HasPostedKindSince(ctx, social.PostKindRecap, platformAnchor, since)
	if err != nil {
		return socialsvc.CandidateContext{}, false, errors.Wrap(err, "checking recap cooldown")
	}
	if posted {
		return socialsvc.CandidateContext{}, false, nil
	}

	top, err := c.recap.Top(ctx, now, digest.TopOptions{MinItems: recapMinItems})
	if err != nil {
		return socialsvc.CandidateContext{}, false, errors.Wrap(err, "computing recap")
	}
	if !top.HasItems() {
		return socialsvc.CandidateContext{}, false, nil
	}

	items := make([]rotation.RecapItem, 0, len(top.Items))
	for _, it := range top.Items {
		items = append(items, rotation.RecapItem{
			Title:  it.Title,
			URL:    it.URL,
			Source: it.Source,
			Clicks: it.Clicks,
		})
	}

	return socialsvc.CandidateContext{
		Kind:    c.Kind(),
		Subject: "recap:" + top.Period.Label,
		Payload: rotation.RecapPayload{
			WeekLabel: top.Period.Label,
			Items:     items,
		},
	}, true, nil
}

// Generate dispatches to the recap prompt.
func (c *Recap) Generate(ctx context.Context, p ai.Prompter, platform socialgw.Platform, cctx socialsvc.CandidateContext) (string, error) {
	payload, ok := cctx.Payload.(rotation.RecapPayload)
	if !ok {
		return "", errors.New("recap: payload missing")
	}
	return rotation.Recap(ctx, p, platform, payload)
}
