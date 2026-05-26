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
	"log/slog"
	"net/http"
	"time"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/hook"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Rotation handles GET /social/rotation.
func (h *Handler) Rotation(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	if api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping rotation — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
		return c.NoContent(http.StatusOK)
	}

	if h.social == nil || !h.social.HasPosters() {
		slog.InfoContext(ctx, "Skipping rotation — no posters configured")
		hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
		return c.NoContent(http.StatusOK)
	}

	results, err := h.social.Rotate(ctx, social.RotateOptions{Now: now})
	if err != nil {
		h.slack.MustSend(ctx, "Rotation post failed: "+err.Error())
		return webkit.NewError(http.StatusInternalServerError, "rotation post failed: "+err.Error())
	}

	slog.InfoContext(ctx, "Rotation run complete", "platforms", len(results))
	hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
	return c.NoContent(http.StatusOK)
}
