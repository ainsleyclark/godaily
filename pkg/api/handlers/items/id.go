// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package items

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ainsleydev/webkit/pkg/webkit"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// Handler holds the narrow dependencies for items HTTP handlers.
type Handler struct {
	itemsRepo news.ItemRepository
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		itemsRepo: a.Repository.Items,
	}
}

// Routes registers all items routes on kit.
func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
	kit.Get("/{id}", h.ByID, auth)
}

// ItemResponse is the response envelope for GET /items/{id}.
type ItemResponse struct {
	Status  int       `json:"status"`
	Error   bool      `json:"error"`
	Message string    `json:"message" example:"Successfully retrieved item"`
	Data    news.Item `json:"data"`
} //@name ItemResponse

// ByID godoc
//
//	@Summary		Fetch a news item by ID.
//	@Description	Returns a single news item identified by its numeric ID.
//	@Tags			items
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int				true	"Item ID"
//	@Success		200	{object}	ItemResponse	"Successfully retrieved item"
//	@Failure		400	{object}	api.Response	"ID is required or not a positive integer"
//	@Failure		404	{object}	api.Response	"Item not found"
//	@Failure		500	{object}	api.Response	"Failed to fetch item"
//	@Router			/items/{id} [get]
func (h *Handler) ByID(c *webkit.Context) error {
	ctx := c.Context()

	raw := c.Param("id")
	if raw == "" {
		return api.Error(c, http.StatusBadRequest, "ID is required")
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}

	item, err := h.itemsRepo.Find(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Item not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch item")
	}

	return api.OK(c, http.StatusOK, item, "Successfully retrieved item")
}
