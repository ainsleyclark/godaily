// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
