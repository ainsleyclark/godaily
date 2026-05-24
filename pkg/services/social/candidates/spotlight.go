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
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/rotation"
)

// spotlightCooldown is the minimum gap between two spotlight posts for
// the same source. With ~8 eligible sources, this gives each one a turn
// roughly every 8 weeks — frequent enough to keep the rotation alive,
// rare enough not to feel like spam.
const spotlightCooldown = 30 * 24 * time.Hour

// Spotlight thanks a curated source and points followers their way. Rotates
// through SourceProfiles in stable name order, skipping any source whose
// last spotlight was inside the cooldown window.
type Spotlight struct {
	profiles map[news.Source]socialsvc.SourceProfile
	posts    news.SocialPostRepository
}

// NewSpotlight constructs the candidate.
func NewSpotlight(profiles map[news.Source]socialsvc.SourceProfile, posts news.SocialPostRepository) *Spotlight {
	return &Spotlight{profiles: profiles, posts: posts}
}

// Kind reports the candidate's SocialPostKind.
func (c *Spotlight) Kind() news.SocialPostKind { return news.SocialPostKindSpotlight }

// Eligible walks SourceProfiles in stable name order, returning the first
// source whose last spotlight is older than the cooldown.
func (c *Spotlight) Eligible(ctx context.Context, now time.Time) (socialsvc.CandidateContext, bool, error) {
	if len(c.profiles) == 0 {
		return socialsvc.CandidateContext{}, false, nil
	}

	sources := sortedSources(c.profiles)
	since := now.UTC().Add(-spotlightCooldown)

	for _, src := range sources {
		subject := "spotlight:" + string(src)
		posted, err := c.posts.HasPostedKindSince(ctx, c.Kind(), platformAnchor, since)
		if err != nil {
			return socialsvc.CandidateContext{}, false, errors.Wrap(err, "checking spotlight cooldown")
		}
		_ = posted // The kind-level check throttles "is rotation talking too often" overall.

		recent, err := c.posts.HasPostedBySubject(ctx, subject, platformAnchor)
		if err != nil {
			return socialsvc.CandidateContext{}, false, errors.Wrap(err, "checking spotlight subject")
		}
		if recent {
			// Already posted about this source on the anchor platform — the
			// subject-level check is permanent rather than time-windowed,
			// so the rotation moves on until we've covered everyone, then
			// we'll need a manual reset (rare and intentional).
			continue
		}

		profile := c.profiles[src]
		return socialsvc.CandidateContext{
			Kind:     c.Kind(),
			Subject:  subject,
			URL:      profile.SourceURL,
			Mentions: cloneMentions(profile.Mentions),
			Payload:  profile,
		}, true, nil
	}

	return socialsvc.CandidateContext{}, false, nil
}

// Generate dispatches to the spotlight prompt with the correct
// per-platform mention.
func (c *Spotlight) Generate(ctx context.Context, p ai.Prompter, platform socialgw.Platform, cctx socialsvc.CandidateContext) (string, error) {
	profile, ok := cctx.Payload.(socialsvc.SourceProfile)
	if !ok {
		return "", errors.New("spotlight: profile payload missing")
	}
	return rotation.Spotlight(ctx, p, platform, rotation.SpotlightPayload{
		DisplayName: profile.DisplayName,
		Mention:     profile.Mention(platform),
		Blurb:       profile.SpotlightBlurb,
		URL:         profile.SourceURL,
	})
}

// platformAnchor is the platform used as the "have I covered this
// already?" probe. Any consistently-configured platform works; bluesky
// is always wired up in practice. The actual post still goes to every
// configured platform via the publish loop.
const platformAnchor = "bluesky"

func sortedSources(profiles map[news.Source]socialsvc.SourceProfile) []news.Source {
	out := make([]news.Source, 0, len(profiles))
	for s := range profiles {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })
	return out
}

func cloneMentions(in map[socialgw.Platform]string) map[socialgw.Platform]string {
	if in == nil {
		return nil
	}
	out := make(map[socialgw.Platform]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
