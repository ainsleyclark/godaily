// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Privacy handles the privacy policy page.
func Privacy() webkit.Handler {
	return func(c *webkit.Context) error {
		return c.Render(pages.Privacy())
	}
}
