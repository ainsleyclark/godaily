// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/store"
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
		return api.Error(c, http.StatusInternalServerError, "Failed to count subscribers")
	}

	subs, err := h.subscribersRepo.List(ctx, store.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to list subscribers")
	}

	return api.OK(c, http.StatusOK, api.PaginatedResponse[audience.Subscriber]{
		Data:    subs,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}, "Successfully retrieved subscribers")
}
