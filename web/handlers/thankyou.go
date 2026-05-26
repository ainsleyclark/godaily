// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// ThankYou handles the post-subscription confirmation page.
func ThankYou() webkit.Handler {
	return func(c *webkit.Context) error {
		email := c.Request.URL.Query().Get("email")
		return c.Render(pages.ThankYou(email))
	}
}
