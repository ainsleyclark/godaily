// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"net/http"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Subscribers handles GET /digest/subscribers.
func (h *Handler) Subscribers(c *webkit.Context) error {
	ctx := c.Context()
	r := c.Request

	page := api.QueryInt(r, "page", api.DefaultPage)
	perPage := api.QueryInt(r, "per_page", api.DefaultPerPage)

	if page < 1 {
		page = api.DefaultPage
	}
	if perPage < 1 || perPage > api.MaxPerPage {
		perPage = api.DefaultPerPage
	}

	total, err := h.subscribersRepo.CountAll(ctx)
	if err != nil {
		return webkit.NewError(http.StatusInternalServerError, "failed to count subscribers")
	}

	subs, err := h.subscribersRepo.List(ctx, store.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return webkit.NewError(http.StatusInternalServerError, "failed to list subscribers")
	}

	return c.JSON(http.StatusOK, api.PaginatedResponse[audience.Subscriber]{
		Data:    subs,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}
