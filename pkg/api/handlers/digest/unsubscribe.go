// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"log/slog"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
)

// Unsubscribe godoc
//
//	@Summary		Unsubscribe an email address.
//	@Description	Removes a subscriber using the token from the digest email.
//	@Description	GET redirects to /unsubscribed/; POST implements the RFC 8058 one-click unsubscribe and returns 200 OK.
//	@Tags			subscription
//	@Produce		json
//	@Param			token	query		string			true	"Unsubscribe token"
//	@Success		200		{object}	api.Response	"Successfully unsubscribed (POST one-click)"
//	@Success		302		"Redirect to /unsubscribed/ (GET)"
//	@Failure		400		{object}	api.Response	"Missing token"
//	@Failure		500		{object}	api.Response	"Failed to unsubscribe"
//	@Router			/unsubscribe [get]
//	@Router			/unsubscribe [post]
func (h *Handler) Unsubscribe(c *webkit.Context) error {
	ctx := c.Context()
	r := c.Request

	token := r.URL.Query().Get("token")
	if token == "" {
		return api.Error(c, http.StatusBadRequest, "Missing token")
	}

	if err := h.subscribers.Unsubscribe(ctx, token); err != nil {
		slog.ErrorContext(ctx, "Failed to unsubscribe", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to unsubscribe")
	}

	// RFC 8058: mail clients send a POST for one-click unsubscribe and
	// expect a 2xx response, not a redirect.
	if r.Method == http.MethodPost {
		return api.OK(c, http.StatusOK, nil, "Successfully unsubscribed")
	}

	return c.Redirect(http.StatusFound, "/unsubscribed/")
}
