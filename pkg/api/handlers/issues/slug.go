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

// IssueResponse is the response envelope for GET /issues/{slug}.
type IssueResponse struct {
	Status  int          `json:"status"`
	Error   bool         `json:"error"`
	Message string       `json:"message" example:"Successfully retrieved issue"`
	Data    digest.Issue `json:"data"`
} //@name IssueResponse

// BySlug godoc
//
//	@Summary		Fetch an issue by slug.
//	@Description	Returns a single digest issue identified by its date slug.
//	@Tags			issues
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Issue date slug"
//	@Success		200		{object}	IssueResponse					"Successfully retrieved issue"
//	@Failure		400		{object}	api.Response					"Slug is required"
//	@Failure		404		{object}	api.Response					"Issue not found"
//	@Failure		500		{object}	api.Response					"Failed to fetch issue"
//	@Router			/issues/{slug} [get]
func (h *Handler) BySlug(c *webkit.Context) error {
	ctx := c.Context()

	slug := c.Param("slug")
	if slug == "" {
		return api.Error(c, http.StatusBadRequest, "Slug is required")
	}

	issue, err := h.issuesRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Issue not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue")
	}

	return api.OK(c, http.StatusOK, issue, "Successfully retrieved issue")
}
