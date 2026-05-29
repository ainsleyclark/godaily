// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package frontend serves dynamic pieces of the otherwise-static website.
// The /browse page is generated as a static file at build time; listing and
// filtering items happens client-side by fetching the rendered HTML fragments
// from here, so the same templ components render on every surface.
package frontend

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/web/handlers"
	"github.com/ainsleyclark/godaily/web/views/pages"
)

// Handler holds the narrow dependencies for the frontend fragment endpoints.
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

// Browse renders the browse page's filter-dependent regions for the given
// query. The results column swaps into the htmx target; the tabs and sidebar
// piggyback as out-of-band swaps. The canonical public URL is returned via the
// HX-Push-Url header so the address bar reflects /browse, not /api/browse.
func (h *Handler) Browse(c *webkit.Context) error {
	props, err := handlers.BuildBrowseProps(c.Context(), h.issuesRepo, h.itemsRepo, c.Request.URL.Query())
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to build browse view")
	}

	c.Response.Header().Set("HX-Push-Url", pages.BrowseURL(props.State))
	return c.Render(pages.BrowseFragment(props))
}
