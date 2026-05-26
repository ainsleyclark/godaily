// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package items

import (
	"errors"
	"net/http"
	"strconv"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleydev/webkit/pkg/webkit"
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
		return webkit.NewError(http.StatusBadRequest, "id is required")
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return webkit.NewError(http.StatusBadRequest, "id must be a positive integer")
	}

	item, err := h.itemsRepo.Find(c.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return webkit.NewError(http.StatusNotFound, "item not found")
		}
		return webkit.NewError(http.StatusInternalServerError, "failed to fetch item")
	}

	return c.JSON(http.StatusOK, item)
}
