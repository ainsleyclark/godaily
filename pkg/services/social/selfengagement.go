// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"log/slog"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
)

// ownerLinkedInMemberURN is Ainsley Clark's personal LinkedIn member URN.
//
// To find yours:
//  1. Visit https://www.linkedin.com/developers/tools/oauth/token-inspector
//     and paste your personal access token.
//  2. The "id" value in the decoded token data is your numeric member ID.
//  3. Alternatively, call GET https://api.linkedin.com/rest/me with your
//     token — the "id" field gives the same numeric ID.
//  4. Confirm the URN format by making a GET /rest/reactions?q=entity&entity=<share-urn>
//     on a post you have liked and noting the "actor" field — it will be
//     either "urn:li:person:<encodedId>" or "urn:li:member:<numericId>".
//     Use whatever format the API returns for your account.
const ownerLinkedInMemberURN = "urn:li:person:REPLACE_ME"

// selfEngagementFetcher wraps a StatFetcher and subtracts the account
// owner's own engagement from the returned counts. LinkedIn's stats
// endpoint returns aggregate totals, so the owner's own like and repost
// inflate the numbers. Unlike a fixed offset, this implementation
// confirms engagement via the reactions API before deducting — if the
// owner didn't engage (or the check fails), nothing is subtracted.
type selfEngagementFetcher struct {
	inner   platform.StatFetcher
	checker platform.ReactionChecker
}

func (f selfEngagementFetcher) Platform() social.Platform {
	return f.inner.Platform()
}

func (f selfEngagementFetcher) Stats(ctx context.Context, postURL string) (platform.Stats, error) {
	stats, err := f.inner.Stats(ctx, postURL)
	if err != nil {
		return stats, err
	}
	if liked, err := f.checker.HasLiked(ctx, postURL); err != nil {
		slog.WarnContext(ctx, "self-engagement like check failed; not deducting", "err", err, "post_url", postURL)
	} else if liked {
		stats.Likes = max(0, stats.Likes-1)
	}
	if reposted, err := f.checker.HasReposted(ctx, postURL); err != nil {
		slog.WarnContext(ctx, "self-engagement repost check failed; not deducting", "err", err, "post_url", postURL)
	} else if reposted {
		stats.Reposts = max(0, stats.Reposts-1)
	}
	return stats, nil
}
