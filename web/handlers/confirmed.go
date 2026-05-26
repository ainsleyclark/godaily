// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Confirmed handles the post-confirmation page shown after a subscriber
// clicks the confirmation link in their email.
func Confirmed(a *godaily.App) webkit.Handler {
	return func(c *webkit.Context) error {
		ctx := c.Request.Context()

		latest, err := a.Repository.Issues.Latest(ctx, 1)
		if err != nil {
			return c.RenderWithStatus(http.StatusInternalServerError, pages.Error(http.StatusInternalServerError))
		}

		var issue digest.Issue
		if len(latest) > 0 {
			issue = latest[0]
		}

		return c.Render(pages.Confirmed(issue))
	}
}
