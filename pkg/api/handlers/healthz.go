// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

// HealthZ handles GET /healthz.
func HealthZ(c *webkit.Context) error {
	return c.NoContent(http.StatusOK)
}
