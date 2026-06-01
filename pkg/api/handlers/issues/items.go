// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// ItemReorderRequest is the JSON body accepted by PATCH /issues/{id}/items/reorder.
// item_ids must be the full ordered list of items currently linked to the
// issue; the new position of each item is its index in this slice.
type ItemReorderRequest struct {
	ItemIDs []int64 `json:"item_ids"`
} //@name ItemReorderRequest

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

	return respondWithIssue(c, h, ctx, issueID, "Successfully unlinked item")
}

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

	issueID, ok := parsePositive(c.Param("id"))
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

	return respondWithIssue(c, h, ctx, issueID, "Successfully reordered items")
}

// parsePositive returns (n, true) if raw parses as a strictly positive int64.
func parsePositive(raw string) (int64, bool) {
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// mapItemMutationError translates repository errors from the item mutation
// endpoints into HTTP responses. Shared because both DeleteItem and
// ReorderItems use the same error vocabulary.
func mapItemMutationError(c *webkit.Context, err error, internalMsg string) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.Error(c, http.StatusNotFound, "Issue or item not found")
	case errors.Is(err, digest.ErrIssueNotDraft):
		return api.Error(c, http.StatusConflict, "Only draft issues can be edited")
	default:
		return api.Error(c, http.StatusInternalServerError, internalMsg)
	}
}

// respondWithIssue refetches the issue (including its items) and writes a 200
// envelope. Used by item mutations so the dashboard can hydrate from a single
// response without a follow-up GET.
func respondWithIssue(c *webkit.Context, h *Handler, ctx context.Context, issueID int64, msg string) error {
	issue, err := h.issuesRepo.Find(ctx, issueID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Issue not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue")
	}
	return api.OK(c, http.StatusOK, issue, msg)
}
