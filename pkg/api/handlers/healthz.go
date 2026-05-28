// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
)

// HealthZ godoc
//
//	@Summary		Health check.
//	@Description	Returns 200 OK when the API is up.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	api.MessageResponse	"ok"
//	@Router			/healthz [get]
func HealthZ(c *webkit.Context) error {
	return api.OK(c, http.StatusOK, nil, "ok")
}
