// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"

	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleydev/webkit/pkg/webkit"
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
		return webkit.NewError(http.StatusInternalServerError, err.Error())
	}

	return c.Redirect(http.StatusFound, "/confirmed/")
}
