// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"github.com/ainsleyclark/godaily/pkg/data"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/meetup"
	digestsvc "github.com/ainsleyclark/godaily/pkg/services/digest"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidates"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform/bluesky"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform/linkedin"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform/mastodon"
)

// buildPosters returns the slice of Poster implementations whose
// credentials are present in the config. Each platform is opt-in:
// missing creds means the platform is skipped entirely.
func buildPosters(c env.Config) []platform.Poster {
	var out []platform.Poster
	if c.BlueskyHandle != "" && c.BlueskyAppPassword != "" {
		out = append(out, bluesky.New(c.BlueskyHandle, c.BlueskyAppPassword))
	}
	if li, ok := NewLinkedIn(c, ""); ok {
		out = append(out, li)
	}
	if c.MastodonServer != "" && c.MastodonAppToken != "" {
		out = append(out, mastodon.New(c.MastodonServer, c.MastodonAppToken))
	}
	return out
}

// NewLinkedIn builds a LinkedIn client from config, enabling automatic
// access-token refresh when refresh credentials are present. memberURN is
// passed through to linkedin.New (use "" to skip self-engagement checks).
// The bool is false when the org URN or usable credentials are missing,
// in which case LinkedIn should be skipped.
func NewLinkedIn(c env.Config, memberURN string) (*linkedin.Client, bool) {
	creds := linkedin.Credentials{
		AccessToken:  c.LinkedInOAuthToken,
		RefreshToken: c.LinkedInRefreshToken,
		ClientID:     c.LinkedInClientID,
		ClientSecret: c.LinkedInClientSecret,
	}
	if c.LinkedInOrgURN == "" || !creds.Usable() {
		return nil, false
	}
	return linkedin.NewFromCredentials(creds, c.LinkedInOrgURN, memberURN), true
}

// buildCandidates wires the rotation candidates the Tue/Wed/Fri rotation
// chooses from. The recap candidate is skipped if metrics aren't wired
// (would never happen in production but keeps tests/no-DB bootstraps
// from blowing up).
func buildCandidates(posts social.PostRepository, metrics engagement.MetricsRepository) []candidate.Candidate {
	out := make([]candidate.Candidate, 0, 5)

	out = append(out, candidates.NewNewSource(social.Profiles, posts))
	out = append(out, candidates.NewSpotlight(social.Profiles, posts))
	out = append(out, candidates.NewCTA(posts))
	out = append(out, candidates.NewCommunity(data.Conferences, data.Meetups, posts, meetup.New()))

	if metrics != nil {
		if recapSvc, err := digestsvc.NewRecapService(metrics); err == nil {
			out = append(out, candidates.NewRecap(recapSvc, posts))
		}
	}
	return out
}

// buildStatFetchers returns a map of platform → StatFetcher for platforms
// whose credentials are present in the config.
func buildStatFetchers(c env.Config) map[social.Platform]platform.StatFetcher {
	out := make(map[social.Platform]platform.StatFetcher)
	if c.BlueskyHandle != "" && c.BlueskyAppPassword != "" {
		out[social.Bluesky] = bluesky.New(c.BlueskyHandle, c.BlueskyAppPassword)
	}
	if li, ok := NewLinkedIn(c, ownerLinkedInMemberID); ok {
		out[social.LinkedIn] = selfEngagementFetcher{
			inner:   li,
			checker: li,
		}
	}
	if c.MastodonServer != "" && c.MastodonAppToken != "" {
		out[social.Mastodon] = mastodon.New(c.MastodonServer, c.MastodonAppToken)
	}
	return out
}
