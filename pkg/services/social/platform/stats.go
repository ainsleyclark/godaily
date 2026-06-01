// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

import (
	"context"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

//go:generate go run go.uber.org/mock/mockgen -package=mocksocial -destination=../../../mocks/social/StatFetcher.go github.com/ainsleyclark/godaily/pkg/services/social/platform StatFetcher
//go:generate go run go.uber.org/mock/mockgen -package=mocksocial -destination=../../../mocks/social/ReactionChecker.go github.com/ainsleyclark/godaily/pkg/services/social/platform ReactionChecker

// StatFetcher fetches engagement stats for a published post by its URL.
type StatFetcher interface {
	// Platform identifies the platform this fetcher targets.
	Platform() social.Platform

	// Stats returns the current engagement counts for the post at postURL.
	// postURL is the canonical web URL stored in social_posts.post_url.
	Stats(ctx context.Context, postURL string) (Stats, error)
}

// ReactionChecker verifies whether the account owner personally engaged with
// a post. Implementations query the platform's per-member reaction APIs so
// engagement is only deducted when it actually happened.
type ReactionChecker interface {
	HasLiked(ctx context.Context, postURL string) (bool, error)
	HasReposted(ctx context.Context, postURL string) (bool, error)
}

// Stats holds the engagement counts returned by a platform for a single post.
type Stats struct {
	Likes       int64
	Reposts     int64
	Comments    int64
	Impressions int64
}
