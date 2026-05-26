// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Issues handles the GoDaily issues archive page.
func Issues(a *godaily.App) webkit.Handler {
	return func(c *webkit.Context) error {
		ctx := c.Context()

		issues, err := a.Repository.Issues.List(ctx, store.ListOptions{})
		if err != nil {
			return c.RenderWithStatus(http.StatusInternalServerError, pages.Error(http.StatusInternalServerError))
		}

		for i, issue := range issues {
			full, err := a.Repository.Issues.Find(ctx, issue.ID)
			if err != nil {
				return c.RenderWithStatus(http.StatusInternalServerError, pages.Error(http.StatusInternalServerError))
			}
			issues[i] = full
		}

		return c.Render(pages.IssuesArchive(issues))
	}
}
