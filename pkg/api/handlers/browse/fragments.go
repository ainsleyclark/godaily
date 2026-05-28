// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package browse

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/web/handlers"
	"github.com/ainsleyclark/godaily/web/views/pages"
)

// FragmentResponse is the JSON envelope returned by GET /browse. Each field
// is the rendered HTML for one swappable region of the browse page.
type FragmentResponse struct {
	Tabs string `json:"tabs"`
	Side string `json:"side"`
	Main string `json:"main"`
}

// Fragments godoc
//
//	@Summary		Render the browse page's filterable regions.
//	@Description	Returns the rendered HTML for the tabs, filter sidebar, and results column for the given filter query. Used by the static /browse page to list and filter items client-side.
//	@Tags			browse
//	@Produce		json
//	@Success		200	{object}	FragmentResponse	"Rendered fragments"
//	@Failure		500	{object}	api.MessageResponse	"Failed to build browse view"
//	@Router			/browse [get]
func (h *Handler) Fragments(c *webkit.Context) error {
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

	return c.JSON(http.StatusOK, FragmentResponse{Tabs: tabs, Side: side, Main: main})
}

func renderComponent(ctx context.Context, comp templ.Component) (string, error) {
	var b strings.Builder
	if err := comp.Render(ctx, &b); err != nil {
		return "", err
	}
	return b.String(), nil
}
