// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"github.com/ainsleydev/webkit/pkg/webkit"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

// Handler holds the narrow dependencies for issues HTTP handlers.
type Handler struct {
	issuesRepo digest.IssueRepository
	itemsRepo  news.ItemRepository
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		issuesRepo: a.Repository.Issues,
		itemsRepo:  a.Repository.Items,
	}
}

// Routes registers all /issues routes on kit. Authenticated.
func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
	kit.Get("/", h.List, auth)
	kit.Get("/{key}", h.Find, auth)
	kit.Patch("/{id}", h.Update, auth)
	kit.Delete("/{id}/items/{itemID}", h.DeleteItem, auth)
	kit.Patch("/{id}/items/reorder", h.ReorderItems, auth)
}
