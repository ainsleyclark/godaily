// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// IssueUpdateRequest is the JSON body accepted by PATCH /digest/issues/{id}.
type IssueUpdateRequest struct {
	Subject string `json:"subject"`
	Summary string `json:"summary"`
} //@name IssueUpdateRequest

// UpdateIssue godoc
//
//	@Summary		Update a draft digest issue.
//	@Description	Updates the subject and summary of a draft issue. Returns 409 if the issue is not in draft status.
//	@Tags			digest
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"Issue ID"
//	@Param			body	body		IssueUpdateRequest	true	"Fields to update"
//	@Success		200		{object}	IssueResponse	"Successfully updated issue"
//	@Failure		400		{object}	api.MessageResponse	"Invalid request"
//	@Failure		404		{object}	api.MessageResponse	"Issue not found"
//	@Failure		409		{object}	api.MessageResponse	"Issue is not a draft"
//	@Failure		500		{object}	api.MessageResponse	"Failed to update issue"
//	@Router			/digest/issues/{id} [patch]
func (h *Handler) UpdateIssue(c *webkit.Context) error {
	ctx := c.Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}

	var body IssueUpdateRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid request body")
	}
	body.Subject = strings.TrimSpace(body.Subject)
	body.Summary = strings.TrimSpace(body.Summary)
	if body.Subject == "" {
		return api.Error(c, http.StatusBadRequest, "Subject is required")
	}

	updated, err := h.issuesRepo.Update(ctx, digest.Issue{
		ID:      id,
		Subject: body.Subject,
		Summary: body.Summary,
	})
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.Error(c, http.StatusNotFound, "Issue not found")
	case errors.Is(err, digest.ErrIssueNotDraft):
		return api.Error(c, http.StatusConflict, "Only draft issues can be edited")
	case err != nil:
		return api.Error(c, http.StatusInternalServerError, "Failed to update issue")
	}

	return api.OK(c, http.StatusOK, updated, "Successfully updated issue")
}
