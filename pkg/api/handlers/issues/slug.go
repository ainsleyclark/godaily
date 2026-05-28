// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"errors"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// Handler holds the narrow dependencies for issues HTTP handlers.
type Handler struct {
	issuesRepo digest.IssueRepository
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		issuesRepo: a.Repository.Issues,
	}
}

// Routes registers all issues routes on kit.
func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
	kit.Get("/{slug}", h.BySlug, auth)
}

// BySlug handles GET /issues/{slug}.
func (h *Handler) BySlug(c *webkit.Context) error {
	slug := c.Param("slug")
	if slug == "" {
		return api.Error(c, http.StatusBadRequest, "Slug is required")
	}

	issue, err := h.issuesRepo.FindBySlug(c.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Issue not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue")
	}

	return api.OK(c, http.StatusOK, issue, "Successfully retrieved issue")
}
