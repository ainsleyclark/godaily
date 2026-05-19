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

package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/digest"
	"github.com/ainsleyclark/godaily/pkg/hook"
)

// HandleCollect is the Vercel serverless function entry point for GET /api/collect.
func HandleCollect(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		if wd := time.Now().UTC().Weekday(); wd == time.Saturday || wd == time.Sunday {
			slog.InfoContext(ctx, "Skipping collect — weekend")
			hook.Heartbeat(ctx, a.Config.BetterStackCollectHeartbeatURL)
			api.OK(w)
			return
		}

		if _, err := a.Runner.Collect(ctx, digest.CollectOptions{}); err != nil {
			a.Slack.MustSend(ctx, "Collect failed: "+err.Error())
			api.Error(w, http.StatusInternalServerError, "collect failed: "+err.Error())
			return
		}

		hook.Heartbeat(ctx, a.Config.BetterStackCollectHeartbeatURL)

		api.OK(w)
	})(w, r)
}
