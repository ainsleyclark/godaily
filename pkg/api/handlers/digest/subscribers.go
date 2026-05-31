// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"net/http"
	"strings"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// SubscriberListResponse is the response envelope for GET /digest/subscribers.
type SubscriberListResponse = api.Response[api.PaginatedResponse[audience.Subscriber]] //@name SubscriberListResponse

// Subscribers godoc
//
//	@Summary		List subscribers.
//	@Description	Returns a paginated list of all subscribers.
//	@Tags			digest
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number (default 1)"
//	@Param			per_page	query		int	false	"Items per page (default 20, max 100)"
//	@Success		200			{object}	SubscriberListResponse											"Successfully retrieved subscribers"
//	@Failure		500			{object}	api.MessageResponse												"Failed to list subscribers"
//	@Router			/digest/subscribers [get]
func (h *Handler) Subscribers(c *webkit.Context) error {
	ctx := c.Context()
	r := c.Request

	search := strings.TrimSpace(r.URL.Query().Get("search"))
	page := api.QueryInt(r, "page", api.DefaultPage)
	perPage := api.QueryInt(r, "per_page", api.DefaultPerPage)

	if page < 1 {
		page = api.DefaultPage
	}
	if perPage < 1 || perPage > api.MaxPerPage {
		perPage = api.DefaultPerPage
	}

	total, err := h.subscribersRepo.CountFiltered(ctx, search)
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to count subscribers")
	}

	subs, err := h.subscribersRepo.List(ctx, store.ListOptions{Page: page, PerPage: perPage, Search: search})
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
