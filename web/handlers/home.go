// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package handlers

import (
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Home handles the GoDaily homepage.
func Home(a *godaily.App) webkit.Handler {
	return func(c *webkit.Context) error {
		ctx := c.Request.Context()

		latest, err := a.Repository.Issues.Latest(ctx, 1)
		if err != nil {
			return c.RenderWithStatus(http.StatusInternalServerError, pages.Error(http.StatusInternalServerError))
		}

		var issue news.Issue
		if len(latest) > 0 {
			issue = latest[0]
		}

		return c.Render(pages.Home(pages.HomeData{
			LatestIssue: issue,
			SampleIssue: issue,
		}))
	}
}
