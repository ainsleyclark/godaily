// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
)

// DeleteItem godoc
//
//	@Summary		Unlink a digest item from a draft issue.
//	@Description	Removes an item from a draft issue by clearing items.issue_id. The item row is preserved in the raw pool and can be re-included by a future build. Returns the refreshed issue. Returns 409 if the issue is not in draft status.
//	@Tags			issues
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int				true	"Issue ID"
//	@Param			itemID	path		int				true	"Item ID"
//	@Success		200		{object}	IssueResponse	"Successfully unlinked item"
//	@Failure		400		{object}	api.MessageResponse	"Invalid id or itemID"
//	@Failure		404		{object}	api.MessageResponse	"Issue or item not found"
//	@Failure		409		{object}	api.MessageResponse	"Issue is not a draft"
//	@Failure		500		{object}	api.MessageResponse	"Failed to unlink item"
//	@Router			/issues/{id}/items/{itemID} [delete]
func (h *Handler) DeleteItem(c *webkit.Context) error {
	ctx := c.Context()

	issueID, ok := parsePositive(c.Param("id"))
	if !ok {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}
	itemID, ok := parsePositive(c.Param("itemID"))
	if !ok {
		return api.Error(c, http.StatusBadRequest, "Item ID must be a positive integer")
	}

	if err := h.itemsRepo.UnlinkFromIssue(ctx, issueID, itemID); err != nil {
		return mapItemMutationError(c, err, "Failed to unlink item")
	}

	return h.respondWithIssue(c, ctx, issueID, "Successfully unlinked item")
}
