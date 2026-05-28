// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

// metricsSince is the look-back window for fetching social post stats.
const metricsSince = 30 * 24 * time.Hour

// Metrics handles GET /social/metrics.
//
// vercel.json schedules this once daily at 03:00 UTC. For every social post
// published in the last 30 days it fetches the current engagement counts from
// the originating platform and upserts them into social_metrics.
func (h *Handler) Metrics(c *webkit.Context) error {
	ctx := c.Context()
	since := time.Now().UTC().Add(-metricsSince)

	posts, err := h.socialPosts.List(ctx, social.PostListOptions{Since: &since})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list recent social posts", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to list posts")
	}

	var updated int
	for _, post := range posts {
		if post.PostURL == "" {
			continue
		}

		fetcher, ok := h.statFetchers[social.Platform(post.Platform)]
		if !ok {
			continue
		}

		stats, err := fetcher.Stats(ctx, post.PostURL)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to fetch social stats",
				"platform", post.Platform, "post_id", post.ID, "err", err)
			continue
		}

		if err = h.socialMetrics.Upsert(ctx, engagement.SocialMetric{
			SocialPostID: post.ID,
			Platform:     post.Platform,
			Likes:        stats.Likes,
			Reposts:      stats.Reposts,
			Comments:     stats.Comments,
			Impressions:  stats.Impressions,
			FetchedAt:    time.Now().UTC(),
		}); err != nil {
			slog.ErrorContext(ctx, "Failed to upsert social metrics",
				"platform", post.Platform, "post_id", post.ID, "err", err)
			continue
		}

		updated++
		slog.InfoContext(ctx, "Upserted social metrics",
			"platform", post.Platform, "post_id", post.ID,
			"likes", stats.Likes, "reposts", stats.Reposts, "comments", stats.Comments)
	}

	slog.InfoContext(ctx, "Social metrics refresh complete",
		"posts_checked", len(posts), "posts_updated", updated)
	return api.OK(c, http.StatusOK, nil, "Successfully refreshed social metrics")
}
