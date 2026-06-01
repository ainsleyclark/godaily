// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drafts

import (
	"log/slog"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

// SocialDraftListResponse is the response envelope for GET /social/drafts.
type SocialDraftListResponse struct {
	Items []social.Post `json:"items"`
} //@name SocialDraftListResponse

// List godoc
//
//	@Summary		List pending social drafts.
//	@Description	Returns every row in social_posts with status='draft' (any kind, any platform). The dashboard renders these in the Drafts tab so an operator can review and edit before the 11:00 publish cron.
//	@Tags			social
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	SocialDraftListResponse	"Successfully listed drafts"
//	@Failure		500	{object}	api.MessageResponse		"Failed to list drafts"
//	@Router			/social/drafts [get]
func (h *Handler) List(c *webkit.Context) error {
	ctx := c.Context()
	draft := social.PostStatusDraft

	rows, err := h.socialPosts.List(ctx, social.PostListOptions{Status: &draft})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list drafts", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to list drafts")
	}
	return api.OK(c, http.StatusOK, SocialDraftListResponse{Items: rows}, "Successfully listed drafts")
}
