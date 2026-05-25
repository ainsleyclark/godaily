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

package digest

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/gateway/hook"
)

// HandleBuild handles GET /digest/build.
func HandleBuild(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		now := time.Now().UTC()
		force := r.URL.Query().Get("force") == "true"
		if !force && api.IsWeekend(now) {
			slog.InfoContext(ctx, "Skipping build — weekend")
			hook.Heartbeat(ctx, a.Config.BetterStackBuildHeartbeatURL)
			api.OK(w)
			return
		}

		if err := a.Runner.Build(ctx, now); err != nil {
			api.Error(w, http.StatusInternalServerError, "build failed: "+err.Error())
			return
		}

		hook.Heartbeat(ctx, a.Config.BetterStackBuildHeartbeatURL)

		api.OK(w)
	})(w, r)
}
