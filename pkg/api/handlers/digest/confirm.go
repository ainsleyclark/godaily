// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// Confirm handles GET /confirm.
func (h *Handler) Confirm(c *webkit.Context) error {
	ctx := c.Context()

	token := c.Request.URL.Query().Get("token")
	if token == "" {
		return c.Redirect(http.StatusFound, "/")
	}

	if err := h.subscribers.Confirm(ctx, token); errors.Is(err, store.ErrNotFound) {
		return c.Redirect(http.StatusFound, "/")
	} else if err != nil {
		slog.ErrorContext(ctx, "Failed to confirm subscriber", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to confirm subscriber")
	}

	return c.Redirect(http.StatusFound, "/confirmed/")
}
