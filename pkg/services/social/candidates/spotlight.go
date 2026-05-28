// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package candidates

import (
	"context"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/rotation"
)

// Spotlight thanks a curated source and points followers their way. It
// iterates news.SocialProfiles in stable source-name order, skipping any
// source already covered on the anchor platform.
type Spotlight struct {
	profiles map[news.Source]social.Profile
	posts    social.PostRepository
}

// NewSpotlight constructs the candidate.
func NewSpotlight(profiles map[news.Source]social.Profile, posts social.PostRepository) *Spotlight {
	return &Spotlight{profiles: profiles, posts: posts}
}

// Kind reports the candidate's SocialPostKind.
func (c *Spotlight) Kind() social.PostKind { return social.PostKindSpotlight }

// Eligible walks SocialProfiles in stable name order, returning the first
// source not yet covered on the anchor platform. Once every source has
// been covered the candidate goes silent until rows are manually pruned.
func (c *Spotlight) Eligible(ctx context.Context, _ time.Time) (candidate.CandidateContext, bool, error) {
	if len(c.profiles) == 0 {
		return candidate.CandidateContext{}, false, nil
	}

	for _, src := range sortedSources(c.profiles) {
		subject := "spotlight:" + string(src)
		posted, err := c.posts.HasPostedBySubject(ctx, subject, platformAnchor)
		if err != nil {
			return candidate.CandidateContext{}, false, errors.Wrap(err, "checking spotlight subject")
		}
		if posted {
			continue
		}

		profile := c.profiles[src]
		return candidate.CandidateContext{
			Kind:     c.Kind(),
			Subject:  subject,
			URL:      profile.SourceURL,
			Mentions: profile.Mentions,
			Payload:  profile,
		}, true, nil
	}

	return candidate.CandidateContext{}, false, nil
}

// Generate dispatches to the spotlight prompt with the right per-platform
// mention.
func (c *Spotlight) Generate(ctx context.Context, p ai.Prompter, platform social.Platform, cctx candidate.CandidateContext) (string, error) {
	profile, ok := cctx.Payload.(social.Profile)
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

func sortedSources(profiles map[news.Source]social.Profile) []news.Source {
	out := make([]news.Source, 0, len(profiles))
	for s := range profiles {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })
	return out
}
