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
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
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
func (c *Spotlight) Eligible(ctx context.Context, _ time.Time) (socialsvc.CandidateContext, bool, error) {
	if len(c.profiles) == 0 {
		return socialsvc.CandidateContext{}, false, nil
	}

	for _, src := range sortedSources(c.profiles) {
		subject := "spotlight:" + string(src)
		posted, err := c.posts.HasPostedBySubject(ctx, subject, platformAnchor)
		if err != nil {
			return socialsvc.CandidateContext{}, false, errors.Wrap(err, "checking spotlight subject")
		}
		if posted {
			continue
		}

		profile := c.profiles[src]
		return socialsvc.CandidateContext{
			Kind:     c.Kind(),
			Subject:  subject,
			URL:      profile.SourceURL,
			Mentions: socialMentionsFor(profile),
			Payload:  profile,
		}, true, nil
	}

	return socialsvc.CandidateContext{}, false, nil
}

// Generate dispatches to the spotlight prompt with the right per-platform
// mention.
func (c *Spotlight) Generate(ctx context.Context, p ai.Prompter, platform platform.Name, cctx socialsvc.CandidateContext) (string, error) {
	profile, ok := cctx.Payload.(social.Profile)
	if !ok {
		return "", errors.New("spotlight: profile payload missing")
	}
	return rotation.Spotlight(ctx, p, platform, rotation.SpotlightPayload{
		DisplayName: profile.DisplayName,
		Mention:     profile.Mention(platform.String()),
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
