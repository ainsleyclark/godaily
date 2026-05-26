// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"errors"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Digest handles individual news issues.
func Digest(a *godaily.App) webkit.Handler {
	return func(c *webkit.Context) error {
		ctx := c.Context()

		issue, err := a.Repository.Issues.FindBySlug(ctx, c.Param("slug"))
		if err != nil && errors.Is(err, store.ErrNotFound) {
			return c.RenderWithStatus(http.StatusNotFound, pages.Error(http.StatusNotFound))
		} else if err != nil {
			return c.RenderWithStatus(http.StatusInternalServerError, pages.Error(http.StatusInternalServerError))
		}

		return c.Render(pages.Digest(issue))
	}
}
