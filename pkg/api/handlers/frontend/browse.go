// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package frontend serves dynamic pieces of the otherwise-static website.
// The /browse page is generated as a static file at build time; listing and
// filtering items happens client-side by fetching the rendered HTML fragments
// from here, so the same templ components render on every surface.
package frontend

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
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

// BrowseResponse is the JSON envelope returned by GET /browse. Each field is
// the rendered HTML for one swappable region of the browse page.
type BrowseResponse struct {
	Tabs string `json:"tabs"`
	Side string `json:"side"`
	Main string `json:"main"`
}

// Browse returns the rendered HTML for the browse page's tabs, filter sidebar,
// and results column for the given filter query. The static /browse page
// fetches these fragments to list and filter items client-side.
func (h *Handler) Browse(c *webkit.Context) error {
	ctx := c.Context()

	props, err := handlers.BuildBrowseProps(ctx, h.issuesRepo, h.itemsRepo, c.Request.URL.Query())
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to build browse view")
	}

	tabs, err := renderComponent(ctx, pages.BrowseTabsRegion(props))
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to render tabs")
	}
	side, err := renderComponent(ctx, pages.BrowseSideRegion(props))
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to render filters")
	}
	main, err := renderComponent(ctx, pages.BrowseMain(props))
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to render results")
	}

	return c.JSON(http.StatusOK, BrowseResponse{Tabs: tabs, Side: side, Main: main})
}

func renderComponent(ctx context.Context, comp templ.Component) (string, error) {
	var b strings.Builder
	if err := comp.Render(ctx, &b); err != nil {
		return "", err
	}
	return b.String(), nil
}
