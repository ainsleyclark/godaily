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
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/rotation"
)

// NewSource announces a source freshly added to GoDaily. It iterates
// news.SocialProfiles in stable source-name order, and picks the first
// Announceable source whose announcement (subject "new_source:<source>")
// has not yet been posted to the anchor platform.
//
// On first deploy the production database should be seeded with
// new_source rows for every source already shipped — otherwise the
// rotation will fire N times in a row. Sample SQL:
//
//	INSERT INTO social_posts (kind, subject, platform, text, posted_at)
//	VALUES ('new_source', 'new_source:hacker_news', 'bluesky', '(backfill)', '2020-01-01');
//
// New sources added after that point trigger one announcement each,
// gated by the same subject check that powers all rotation candidates.
type NewSource struct {
	profiles map[news.Source]social.Profile
	posts    social.PostRepository
}

// NewNewSource constructs the candidate. The name reads awkwardly; it
// matches the Kind convention (NewX returns *X) the other candidates use.
func NewNewSource(profiles map[news.Source]social.Profile, posts social.PostRepository) *NewSource {
	return &NewSource{profiles: profiles, posts: posts}
}

// Kind reports the candidate's SocialPostKind.
func (c *NewSource) Kind() social.PostKind { return social.PostKindNewSource }

// Eligible walks Announceable profiles in stable source-name order and
// returns the first source we haven't announced on the anchor platform yet.
func (c *NewSource) Eligible(ctx context.Context, _ time.Time) (socialsvc.CandidateContext, bool, error) {
	if len(c.profiles) == 0 {
		return socialsvc.CandidateContext{}, false, nil
	}

	sources := sortedAnnounceable(c.profiles)
	for _, src := range sources {
		subject := "new_source:" + string(src)
		posted, err := c.posts.HasPostedBySubject(ctx, subject, platformAnchor)
		if err != nil {
			return socialsvc.CandidateContext{}, false, errors.Wrap(err, "checking new_source subject")
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

// Generate dispatches to the new_source prompt with the right per-platform
// mention.
func (c *NewSource) Generate(ctx context.Context, p ai.Prompter, plat social.Platform, cctx socialsvc.CandidateContext) (string, error) {
	profile, ok := cctx.Payload.(social.Profile)
	if !ok {
		return "", errors.New("new_source: profile payload missing")
	}
	return rotation.NewSource(ctx, p, plat, rotation.NewSourcePayload{
		DisplayName: profile.DisplayName,
		Mention:     profile.Mention(plat.String()),
		Blurb:       profile.SpotlightBlurb,
		URL:         profile.SourceURL,
	})
}

func sortedAnnounceable(profiles map[news.Source]social.Profile) []news.Source {
	out := make([]news.Source, 0, len(profiles))
	for s, p := range profiles {
		if p.Announceable {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })
	return out
}

// socialMentionsFor translates SocialProfile.Mentions (string-keyed, so
// the news package stays free of socialgw imports) into the typed map the
// rotation orchestrator carries around.
func socialMentionsFor(p social.Profile) map[social.Platform]string {
	if len(p.Mentions) == 0 {
		return nil
	}
	out := make(map[social.Platform]string, len(p.Mentions))
	for k, v := range p.Mentions {
		out[social.Platform(k)] = v
	}
	return out
}
