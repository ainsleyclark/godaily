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
	"github.com/ainsleyclark/godaily/pkg/gateway/hook"
	"github.com/ainsleyclark/godaily/pkg/services/social"
)

// HandleFeatured handles GET /social/featured.
//
// vercel.json schedules six cron firings per day (every 10 minutes between
// 11:00 and 11:50 UTC, weekdays). For each firing this handler:
//   - skips weekends entirely
//   - asks social.PickSlot which 10-minute slot today's posts belong in
//   - returns OK with no-op if the current minute is not the chosen slot
//   - otherwise delegates to a.Social.Post() which is itself idempotent
//     via the social_posts table.
func HandleFeatured(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		now := time.Now().UTC()
		if api.IsWeekend(now) {
			slog.InfoContext(ctx, "Skipping featured — weekend")
			hook.Heartbeat(ctx, a.Config.BetterStackSocialFeaturedHeartbeatURL)
			api.OK(w)
			return
		}

		today := now.Truncate(24 * time.Hour)

		if !social.ShouldRun(now, today) {
			slog.InfoContext(
				ctx, "Skipping featured — wrong slot",
				"minute", now.Minute(), "picked", social.PickSlot(today),
			)
			api.OK(w)
			return
		}

		if a.Social == nil || !a.Social.HasPosters() {
			slog.InfoContext(ctx, "Skipping featured — no posters configured")
			hook.Heartbeat(ctx, a.Config.BetterStackSocialFeaturedHeartbeatURL)
			api.OK(w)
			return
		}

		results, err := a.Social.Post(ctx, social.PostOptions{Date: today})
		if err != nil {
			a.Slack.MustSend(ctx, "Featured post failed: "+err.Error())
			api.Error(w, http.StatusInternalServerError, "featured post failed: "+err.Error())
			return
		}

		slog.InfoContext(ctx, "Featured run complete", "platforms", len(results))
		hook.Heartbeat(ctx, a.Config.BetterStackSocialFeaturedHeartbeatURL)
		api.OK(w)
	})(w, r)
}
