// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drafts

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// Cancel godoc
//
//	@Summary		Cancel a social draft so the publish cron skips it.
//	@Description	Transitions a draft row to status='cancelled'. The publish cron filters cancelled rows out and rotation idempotency counts them as "already handled", preventing tomorrow's build from regenerating the same subject.
//	@Tags			social
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int						true	"Social post ID"
//	@Success		200	{object}	social.Post				"Cancelled draft"
//	@Failure		400	{object}	api.MessageResponse		"Invalid request"
//	@Failure		404	{object}	api.MessageResponse		"Draft not found"
//	@Failure		409	{object}	api.MessageResponse		"Row is no longer a draft"
//	@Failure		500	{object}	api.MessageResponse		"Failed to cancel draft"
//	@Router			/social/drafts/{id}/cancel [post]
func (h *Handler) Cancel(c *webkit.Context) error {
	ctx := c.Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}

	if ok, err := h.requireDraft(c, ctx, id); !ok {
		return err
	}

	cancelled := social.PostStatusCancelled
	updated, err := h.socialPosts.Update(ctx, id, social.PostUpdate{Status: &cancelled})
	if errors.Is(err, store.ErrNotFound) {
		return api.Error(c, http.StatusNotFound, "Draft not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "Failed to cancel draft", "id", id, "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to cancel draft")
	}
	return api.OK(c, http.StatusOK, updated, "Successfully cancelled draft")
}
