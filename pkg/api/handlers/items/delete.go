// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package items

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// Delete godoc
//
//	@Summary		Permanently delete a news item.
//	@Description	Hard-deletes a news item from the database by its numeric ID, regardless of whether it is linked to an issue. The row is removed entirely and will not reappear in a future build. Use the issue unlink endpoint instead to keep the item in the raw pool.
//	@Tags			items
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int				true	"Item ID"
//	@Success		200	{object}	api.MessageResponse	"Successfully deleted item"
//	@Failure		400	{object}	api.MessageResponse	"ID is required or not a positive integer"
//	@Failure		404	{object}	api.MessageResponse	"Item not found"
//	@Failure		500	{object}	api.MessageResponse	"Failed to delete item"
//	@Router			/items/{id} [delete]
func (h *Handler) Delete(c *webkit.Context) error {
	ctx := c.Context()

	raw := c.Param("id")
	if raw == "" {
		return api.Error(c, http.StatusBadRequest, "ID is required")
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}

	if err := h.itemsRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Item not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to delete item")
	}

	return api.OK(c, http.StatusOK, nil, "Successfully deleted item")
}
