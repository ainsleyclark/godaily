// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Unsubscribe handles /unsubscribe.
// GET serves the link click (redirect to /unsubscribed/),
// POST serves the RFC 8058 one-click unsubscribe (return 200 OK).
func (h *Handler) Unsubscribe(c *webkit.Context) error {
	ctx := c.Context()
	r := c.Request

	token := r.URL.Query().Get("token")
	if token == "" {
		return webkit.NewError(http.StatusBadRequest, "missing token")
	}

	if err := h.subscribers.Unsubscribe(ctx, token); err != nil {
		return webkit.NewError(http.StatusInternalServerError, err.Error())
	}

	// RFC 8058: mail clients send a POST for one-click unsubscribe and
	// expect a 2xx response, not a redirect.
	if r.Method == http.MethodPost {
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/unsubscribed/")
}
