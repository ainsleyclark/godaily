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

// ownerLinkedInMemberID is the base64-encoded ID portion of Ainsley Clark's
// personal LinkedIn profile URN. Storing just the ID (not the full URN) makes
// matching format-agnostic — the reactions API may return the actor as
// urn:li:person:<id>, urn:li:fsd_profile:<id>, or urn:li:member:<id> depending
// on the API version; all contain this same encoded string.
//
// To find yours, view source on your own LinkedIn profile page and search for
// "fsd_profile:" — the value after the colon is your encoded ID.
const ownerLinkedInMemberID = "ACoAABmJE4IBOoNt3FvzgLxwuVL6aWFKmeSLk0M"

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
