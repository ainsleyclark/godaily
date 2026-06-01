// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// IssueResponse is the response envelope for GET /issues/{key}.
type IssueResponse = api.Response[digest.Issue] //@name IssueResponse

// Find godoc
//
//	@Summary		Fetch a digest issue by ID or slug.
//	@Description	Returns a single digest issue, including its items. The {key} path parameter is interpreted as a numeric issue ID if it parses as a positive integer; otherwise it is treated as a slug.
//	@Tags			issues
//	@Produce		json
//	@Security		BearerAuth
//	@Param			key	path		string			true	"Issue ID (numeric) or slug"
//	@Success		200	{object}	IssueResponse	"Successfully retrieved issue"
//	@Failure		400	{object}	api.MessageResponse	"Key is required"
//	@Failure		404	{object}	api.MessageResponse	"Issue not found"
//	@Failure		500	{object}	api.MessageResponse	"Failed to fetch issue"
//	@Router			/issues/{key} [get]
func (h *Handler) Find(c *webkit.Context) error {
	ctx := c.Context()

	key := c.Param("key")
	if key == "" {
		return api.Error(c, http.StatusBadRequest, "Key is required")
	}

	var (
		issue digest.Issue
		err   error
	)
	if id, parseErr := strconv.ParseInt(key, 10, 64); parseErr == nil && id > 0 {
		issue, err = h.issuesRepo.Find(ctx, id)
	} else {
		issue, err = h.issuesRepo.FindBySlug(ctx, key)
	}
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Issue not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue")
	}

	return api.OK(c, http.StatusOK, issue, "Successfully retrieved issue")
}
