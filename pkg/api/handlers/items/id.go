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

// ByID handles GET /items/{id}.
func (h *Handler) ByID(c *webkit.Context) error {
	raw := c.Param("id")
	if raw == "" {
		return api.Error(c, http.StatusBadRequest, "ID is required")
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}

	item, err := h.itemsRepo.Find(c.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Item not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch item")
	}

	return api.OK(c, http.StatusOK, item, "Successfully retrieved item")
}
