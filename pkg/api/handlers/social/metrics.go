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

package social

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

// metricsSince is the look-back window for fetching social post stats.
const metricsSince = 30 * 24 * time.Hour

// HandleMetrics handles GET /social/metrics.
//
// vercel.json schedules this once daily at 03:00 UTC. For every social post
// published in the last 30 days it fetches the current engagement counts from
// the originating platform and upserts them into social_metrics.
func HandleMetrics(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		since := time.Now().UTC().Add(-metricsSince)

		posts, err := a.Repository.SocialPosts.List(ctx, social.PostListOptions{Since: &since})
		if err != nil {
			slog.ErrorContext(ctx, "Failed to list recent social posts", "err", err)
			api.Error(w, http.StatusInternalServerError, "failed to list posts")
			return
		}

		var updated int
		for _, post := range posts {
			if post.PostURL == "" {
				continue
			}

			fetcher, ok := a.StatFetchers[social.Platform(post.Platform)]
			if !ok {
				continue
			}

			stats, err := fetcher.Stats(ctx, post.PostURL)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to fetch social stats",
					"platform", post.Platform, "post_id", post.ID, "err", err)
				continue
			}

			if err = a.Repository.SocialMetrics.Upsert(ctx, engagement.SocialMetric{
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
		api.OK(w)
	})(w, r)
}
