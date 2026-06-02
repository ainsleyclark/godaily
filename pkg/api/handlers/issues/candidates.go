// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"errors"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// CandidateListResponse is the response envelope for GET /issues/{id}/candidates.
type CandidateListResponse = api.Response[[]news.Item] //@name CandidateListResponse

// Candidates godoc
//
//	@Summary		List items not in the issue that could be added to it.
//	@Description	Returns the raw, unlinked items (issue_id IS NULL) published within this issue's build window — the candidate pool that can be promoted into the digest via PUT /issues/{id}/items/{itemID}. Sorted by score descending.
//	@Tags			issues
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int						true	"Issue ID"
//	@Success		200	{object}	CandidateListResponse	"Successfully retrieved candidate items"
//	@Failure		400	{object}	api.MessageResponse		"Invalid id"
//	@Failure		404	{object}	api.MessageResponse		"Issue not found"
//	@Failure		500	{object}	api.MessageResponse		"Failed to list candidate items"
//	@Router			/issues/{id}/candidates [get]
func (h *Handler) Candidates(c *webkit.Context) error {
	ctx := c.Context()

	issueID, ok := parsePositive(c.Param("id"))
	if !ok {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}

	issue, err := h.issuesRepo.Find(ctx, issueID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Issue not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue")
	}

	start, end := digest.BuildWindow(issue.SentAt)
	notInDigest := false
	items, err := h.itemsRepo.List(ctx, news.ItemListOptions{
		From:     &start,
		To:       &end,
		InDigest: &notInDigest,
		Sort:     news.ItemSortTop,
	})
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to list candidate items")
	}

	return api.OK(c, http.StatusOK, items, "Successfully retrieved candidate items")
}
