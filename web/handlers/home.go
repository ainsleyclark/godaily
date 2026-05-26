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

// Home handles the GoDaily homepage.
func Home(a *godaily.App) webkit.Handler {
	return func(c *webkit.Context) error {
		ctx := c.Request.Context()

		recent, err := a.Repository.Issues.Latest(ctx, 4)
		if err != nil {
			return c.RenderWithStatus(http.StatusInternalServerError, pages.Error(http.StatusInternalServerError))
		}

		var issue digest.Issue
		if len(recent) > 0 {
			issue = recent[0]
		}

		var flash string
		if c.Request.URL.Query().Get("confirmed") != "" {
			flash = "You're confirmed! Digest arrives weekday mornings."
		}

		return c.Render(pages.Home(pages.HomeData{
			LatestIssue:  issue,
			SampleIssue:  issue,
			RecentIssues: recent,
			Flash:        flash,
		}))
	}
}
