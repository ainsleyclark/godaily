// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
)

// AddItem godoc
//
//	@Summary		Link a raw item into a draft issue.
//	@Description	Adds a currently-unlinked item to a draft issue by setting items.issue_id, appending it after the issue's existing items. The item must be unlinked (in the raw pool) and the issue must be in draft status. Returns the refreshed issue. Returns 409 if the issue is not in draft status.
//	@Tags			issues
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"Issue ID"
//	@Param			itemID	path		int					true	"Item ID"
//	@Success		200		{object}	IssueResponse		"Successfully linked item"
//	@Failure		400		{object}	api.MessageResponse	"Invalid id or itemID"
//	@Failure		404		{object}	api.MessageResponse	"Issue or item not found, or item already linked"
//	@Failure		409		{object}	api.MessageResponse	"Issue is not a draft"
//	@Failure		500		{object}	api.MessageResponse	"Failed to link item"
//	@Router			/issues/{id}/items/{itemID} [put]
func (h *Handler) AddItem(c *webkit.Context) error {
	ctx := c.Context()

	issueID, ok := parsePositive(c.Param("id"))
	if !ok {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}
	itemID, ok := parsePositive(c.Param("itemID"))
	if !ok {
		return api.Error(c, http.StatusBadRequest, "Item ID must be a positive integer")
	}

	if err := h.itemsRepo.LinkToIssue(ctx, issueID, itemID); err != nil {
		return mapItemMutationError(c, err, "Failed to link item")
	}

	return h.respondWithIssue(c, ctx, issueID, "Successfully linked item")
}
