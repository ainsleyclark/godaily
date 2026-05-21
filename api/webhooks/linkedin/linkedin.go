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

package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/social/linkedin"
)

// Handler is the Vercel serverless function entry point for
// POST /api/webhooks/linkedin.
//
// LinkedIn pushes engagement events (likes, comments, shares) to this endpoint
// when your app is approved for the Community Management API product. Each
// event triggers a refresh of that post's engagement stats via the LinkedIn
// API, which are then upserted into social_metrics.
//
// Requests without a valid X-LI-Signature are rejected with 401.
func Handler(w http.ResponseWriter, r *http.Request) {
	api.Handle(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		if r.Method != http.MethodPost {
			api.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		secret := a.Config.LinkedInWebhookSecret
		if secret == "" {
			api.Error(w, http.StatusInternalServerError, "linkedin webhook secret is not configured")
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			api.Error(w, http.StatusBadRequest, "cannot read request body")
			return
		}

		sig := r.Header.Get("X-LI-Signature")
		if err = linkedin.VerifyWebhook(body, sig, secret); err != nil {
			slog.WarnContext(ctx, "Rejected LinkedIn webhook with invalid signature", "err", err)
			api.Error(w, http.StatusUnauthorized, "invalid signature")
			return
		}

		evt, err := linkedin.ParseWebhook(body)
		if err != nil {
			api.Error(w, http.StatusBadRequest, "invalid payload")
			return
		}

		fetcher, ok := a.StatFetchers[socialgw.PlatformLinkedIn]
		if !ok {
			slog.WarnContext(ctx, "LinkedIn stat fetcher not configured — skipping metrics refresh")
			api.OK(w)
			return
		}

		// Refresh stats for every LinkedIn post referenced in the event batch.
		// We look up the social_posts row by platform and search for posts
		// from the last 90 days so we cover any delayed webhook delivery.
		since := time.Now().UTC().AddDate(0, 0, -90)
		posts, err := a.Repository.SocialPosts.ListSince(ctx, since)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to list recent social posts", "err", err)
			api.Error(w, http.StatusInternalServerError, "failed to list posts")
			return
		}

		// Collect the entity URNs from the event so we only refresh affected posts.
		urnSet := make(map[string]struct{}, len(evt.Events))
		for _, e := range evt.Events {
			if e.EntityUrn != "" {
				urnSet[e.EntityUrn] = struct{}{}
			}
		}

		for _, post := range posts {
			if post.Platform != socialgw.PlatformLinkedIn.String() || post.PostURL == "" {
				continue
			}

			// If LinkedIn sent specific entity URNs, skip posts not in the set.
			if len(urnSet) > 0 {
				matched := false
				for urn := range urnSet {
					if containsURN(post.PostURL, urn) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}

			stats, err := fetcher.GetStats(ctx, post.PostURL)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to fetch LinkedIn stats", "post_id", post.ID, "err", err)
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
				slog.ErrorContext(ctx, "Failed to upsert LinkedIn social metrics", "post_id", post.ID, "err", err)
				continue
			}

			slog.InfoContext(ctx, "Upserted LinkedIn social metrics", "post_id", post.ID,
				"likes", stats.Likes, "comments", stats.Comments)
		}

		api.OK(w)
	})(w, r)
}

// containsURN reports whether postURL contains the given LinkedIn entity URN,
// accounting for both plain and URL-encoded (colon → %3A) forms.
func containsURN(postURL, urn string) bool {
	if urn == "" || postURL == "" {
		return false
	}
	encoded := strings.ReplaceAll(urn, ":", "%3A")
	return strings.Contains(postURL, urn) || strings.Contains(postURL, encoded)
}
