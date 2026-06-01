// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drafts

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// SocialDraftUpdateRequest is the JSON body accepted by PATCH
// /social/drafts/{id}. Only Text may be edited from the dashboard —
// status transitions go through Publish or Cancel so the lifecycle stays
// linear and explicit.
type SocialDraftUpdateRequest struct {
	Text string `json:"text"`
} //@name SocialDraftUpdateRequest

// Update godoc
//
//	@Summary		Edit a social draft's text.
//	@Description	Replaces the text body of a draft social post. Only rows with status='draft' may be edited; the request is rejected with 409 if the row has already been published, errored, or cancelled.
//	@Tags			social
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int						true	"Social post ID"
//	@Param			body	body		SocialDraftUpdateRequest	true	"Replacement text"
//	@Success		200		{object}	social.Post				"Updated draft"
//	@Failure		400		{object}	api.MessageResponse		"Invalid request"
//	@Failure		404		{object}	api.MessageResponse		"Draft not found"
//	@Failure		409		{object}	api.MessageResponse		"Row is no longer a draft"
//	@Failure		500		{object}	api.MessageResponse		"Failed to update draft"
//	@Router			/social/drafts/{id} [patch]
func (h *Handler) Update(c *webkit.Context) error {
	ctx := c.Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}

	var body SocialDraftUpdateRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid request body")
	}
	text := strings.TrimSpace(body.Text)
	if text == "" {
		return api.Error(c, http.StatusBadRequest, "Text is required")
	}

	if ok, err := h.requireDraft(c, ctx, id); !ok {
		return err
	}

	updated, err := h.socialPosts.Update(ctx, id, social.PostUpdate{Text: &text})
	if errors.Is(err, store.ErrNotFound) {
		return api.Error(c, http.StatusNotFound, "Draft not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update draft", "id", id, "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to update draft")
	}
	return api.OK(c, http.StatusOK, updated, "Successfully updated draft")
}
