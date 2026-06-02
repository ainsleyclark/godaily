// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"context"
	"errors"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
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
	kit.Get("/{id}/candidates", h.Candidates, auth)
	kit.Patch("/{id}", h.Update, auth)
	kit.Put("/{id}/items/{itemID}", h.AddItem, auth)
	kit.Delete("/{id}/items/{itemID}", h.DeleteItem, auth)
	kit.Patch("/{id}/items/reorder", h.ReorderItems, auth)
}

// mapItemMutationError translates repository errors from the item mutation
// endpoints into HTTP responses. Shared because DeleteItem and ReorderItems
// use the same error vocabulary.
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
func (h *Handler) respondWithIssue(c *webkit.Context, ctx context.Context, issueID int64, msg string) error {
	issue, err := h.issuesRepo.Find(ctx, issueID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Issue not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue")
	}
	return api.OK(c, http.StatusOK, issue, msg)
}
