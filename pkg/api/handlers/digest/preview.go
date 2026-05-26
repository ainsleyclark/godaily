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
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/gateway/hook"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Preview handles GET /digest/preview.
// It runs at 6 AM UTC on weekdays, sending the draft digest and AI synth suggestion
// to the owner before the full subscriber send at 8 AM.
func (h *Handler) Preview(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	if api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping preview — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSendHeartbeatURL)
		return c.NoContent(http.StatusOK)
	}

	today := now.Truncate(24 * time.Hour)

	if err := h.runner.SendPreview(ctx, today); err != nil {
		h.slack.MustSend(ctx, "Send preview failed: "+err.Error())
		return fmt.Errorf("send preview failed: %w", err)
	}

	hook.Heartbeat(ctx, h.config.BetterStackSendHeartbeatURL)

	return c.NoContent(http.StatusOK)
}
