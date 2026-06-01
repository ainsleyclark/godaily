// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
)

// selfEngagementFetcher wraps a StatFetcher and subtracts the account
// owner's own engagement from the returned counts. LinkedIn's stats
// endpoint returns aggregate totals, so the owner's own like and repost
// of each post inflate the numbers — subtract 1 for each action the
// owner consistently takes on every post.
type selfEngagementFetcher struct {
	inner   platform.StatFetcher
	likes   int64
	reposts int64
}

func (f selfEngagementFetcher) Platform() social.Platform {
	return f.inner.Platform()
}

func (f selfEngagementFetcher) Stats(ctx context.Context, postURL string) (platform.Stats, error) {
	stats, err := f.inner.Stats(ctx, postURL)
	if err != nil {
		return stats, err
	}
	stats.Likes = max(0, stats.Likes-f.likes)
	stats.Reposts = max(0, stats.Reposts-f.reposts)
	return stats, nil
}
