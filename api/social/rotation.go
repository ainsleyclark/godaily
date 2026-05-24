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

// HandleRotation is the Vercel serverless function entry point for
// GET /api/social/rotation.
//
// vercel.json schedules this once on Tue and Fri at 15:00 UTC. The handler
// delegates the day-of-week routing to the social service:
//   - Tuesday: walks self_release → spotlight → cta and picks the first
//     eligible candidate.
//   - Friday:  runs the weekly recap (or no-ops if there's no click data).
//   - Other days: no-op.
//
// Idempotency lives in each candidate; the handler just authorises the
// call, runs it, and emits a heartbeat.
func HandleRotation(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		now := time.Now().UTC()
		if api.IsWeekend(now) {
			slog.InfoContext(ctx, "Skipping rotation — weekend")
			hook.Heartbeat(ctx, a.Config.BetterStackSocialRotationHeartbeatURL)
			api.OK(w)
			return
		}

		if a.Social == nil || !a.Social.HasPosters() {
			slog.InfoContext(ctx, "Skipping rotation — no posters configured")
			hook.Heartbeat(ctx, a.Config.BetterStackSocialRotationHeartbeatURL)
			api.OK(w)
			return
		}

		results, err := a.Social.Rotate(ctx, social.RotateOptions{Now: now})
		if err != nil {
			a.Slack.MustSend(ctx, "Rotation post failed: "+err.Error())
			api.Error(w, http.StatusInternalServerError, "rotation post failed: "+err.Error())
			return
		}

		slog.InfoContext(ctx, "Rotation run complete", "platforms", len(results))
		hook.Heartbeat(ctx, a.Config.BetterStackSocialRotationHeartbeatURL)
		api.OK(w)
	})(w, r)
}
