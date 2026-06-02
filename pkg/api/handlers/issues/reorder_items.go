// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"encoding/json"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
)

// ItemReorderRequest is the JSON body accepted by PATCH /issues/{id}/items/reorder.
// item_ids must be the full ordered list of items currently linked to the
// issue; the new position of each item is its index in this slice.
type ItemReorderRequest struct {
	ItemIDs []int64 `json:"item_ids"`
} //@name ItemReorderRequest

// ReorderItems godoc
//
//	@Summary		Reorder digest items within a draft issue.
//	@Description	Rewrites the position of each item in the issue using the supplied order — item_ids[i] becomes position i. The submitted ids must exactly match the set of items currently linked to the issue; partial reorders are rejected. Returns the refreshed issue. Returns 409 if the issue is not in draft status.
//	@Tags			issues
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"Issue ID"
//	@Param			body	body		ItemReorderRequest	true	"Ordered list of item IDs"
//	@Success		200		{object}	IssueResponse		"Successfully reordered items"
//	@Failure		400		{object}	api.MessageResponse	"Invalid request"
//	@Failure		404		{object}	api.MessageResponse	"Issue or item not found"
//	@Failure		409		{object}	api.MessageResponse	"Issue is not a draft"
//	@Failure		500		{object}	api.MessageResponse	"Failed to reorder items"
//	@Router			/issues/{id}/items/reorder [patch]
func (h *Handler) ReorderItems(c *webkit.Context) error {
	ctx := c.Context()

	issueID, ok := api.ParseID(c.Param("id"))
	if !ok {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}

	var body ItemReorderRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid request body")
	}
	if len(body.ItemIDs) == 0 {
		return api.Error(c, http.StatusBadRequest, "item_ids must not be empty")
	}

	if err := h.itemsRepo.ReorderInIssue(ctx, issueID, body.ItemIDs); err != nil {
		return mapItemMutationError(c, err, "Failed to reorder items")
	}

	return h.respondWithIssue(c, ctx, issueID, "Successfully reordered items")
}
