// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// UpdateSubscriberRequest is the request body for PATCH /digest/subscribers/:id.
type UpdateSubscriberRequest struct {
	Status string `json:"status"`
}

// UpdateSubscriberResponse is the response envelope.
type UpdateSubscriberResponse = api.Response[audience.Subscriber]

// UpdateSubscriber godoc
//
//	@Summary		Update a subscriber.
//	@Description	Updates a subscriber by ID. Currently supports setting status: active, unsubscribed, suppressed.
//	@Tags			digest
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"Subscriber ID"
//	@Param			body	body		UpdateSubscriberRequest		true	"Update payload"
//	@Success		200		{object}	UpdateSubscriberResponse	"Subscriber updated"
//	@Failure		400		{object}	api.MessageResponse			"Invalid request"
//	@Failure		404		{object}	api.MessageResponse			"Subscriber not found"
//	@Failure		500		{object}	api.MessageResponse			"Failed to update subscriber"
//	@Router			/digest/subscribers/{id} [patch]
func (h *Handler) UpdateSubscriber(c *webkit.Context) error {
	ctx := c.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return api.Error(c, http.StatusBadRequest, "Invalid subscriber ID")
	}

	var req UpdateSubscriberRequest
	if err := c.BindJSON(&req); err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid request body")
	}

	switch req.Status {
	case "active", "unsubscribed", "suppressed":
	default:
		return api.Error(c, http.StatusBadRequest, "Invalid status: must be active, unsubscribed, or suppressed")
	}

	sub, err := h.subscribersRepo.SetStatus(ctx, id, req.Status)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Subscriber not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to update subscriber")
	}

	return api.OK(c, http.StatusOK, sub, "Subscriber updated successfully")
}
