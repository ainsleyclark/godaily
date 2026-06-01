// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// IssueResponse is the response envelope for GET /digest/issues/{id}.
type IssueResponse = api.Response[digest.Issue] //@name IssueResponse

// IssueByID godoc
//
//	@Summary		Fetch a digest issue by ID.
//	@Description	Returns a single digest issue, including its items grouped for rendering.
//	@Tags			digest
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int				true	"Issue ID"
//	@Success		200	{object}	IssueResponse	"Successfully retrieved issue"
//	@Failure		400	{object}	api.MessageResponse	"ID must be a positive integer"
//	@Failure		404	{object}	api.MessageResponse	"Issue not found"
//	@Failure		500	{object}	api.MessageResponse	"Failed to fetch issue"
//	@Router			/digest/issues/{id} [get]
func (h *Handler) IssueByID(c *webkit.Context) error {
	ctx := c.Context()

	raw := c.Param("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}

	issue, err := h.issuesRepo.Find(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Issue not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue")
	}

	return api.OK(c, http.StatusOK, issue, "Successfully retrieved issue")
}
