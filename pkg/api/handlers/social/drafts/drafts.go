// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package drafts owns the HTTP surface for editing pending social
// posts. Mounted under /social/drafts. The endpoints sit in their own
// package because they have a richer lifecycle than the publish-cron
// handler in the parent package: list, edit text, cancel — each backed
// by a status check so a published or errored row cannot be retroactively
// mutated.
package drafts

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ainsleydev/webkit/pkg/webkit"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// Handler holds the dependencies for /social/drafts routes.
type Handler struct {
	socialPosts social.PostRepository
}

// New constructs a drafts Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{socialPosts: a.Repository.SocialPosts}
}

// Routes registers all /social/drafts routes on kit. The parent social
// handler group-mounts this under /drafts, so the relative paths here
// resolve to /social/drafts/*.
func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
	kit.Get("/", h.List, auth)
	kit.Patch("/{id}", h.Update, auth)
	kit.Post("/{id}/cancel", h.Cancel, auth)
}

// SocialDraftUpdateRequest is the JSON body accepted by PATCH
// /social/drafts/{id}. Only Text may be edited from the dashboard —
// status transitions go through Publish or Cancel so the lifecycle stays
// linear and explicit.
type SocialDraftUpdateRequest struct {
	Text string `json:"text"`
} //@name SocialDraftUpdateRequest

// SocialDraftListResponse is the response envelope for GET /social/drafts.
type SocialDraftListResponse struct {
	Items []social.Post `json:"items"`
} //@name SocialDraftListResponse

// List godoc
//
//	@Summary		List pending social drafts.
//	@Description	Returns every row in social_posts with status='draft' (any kind, any platform). The dashboard renders these in the Drafts tab so an operator can review and edit before the 11:00 publish cron.
//	@Tags			social
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	SocialDraftListResponse	"Successfully listed drafts"
//	@Failure		500	{object}	api.MessageResponse		"Failed to list drafts"
//	@Router			/social/drafts [get]
func (h *Handler) List(c *webkit.Context) error {
	ctx := c.Context()
	draft := social.PostStatusDraft

	rows, err := h.socialPosts.List(ctx, social.PostListOptions{Status: &draft})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list drafts", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to list drafts")
	}
	return api.OK(c, http.StatusOK, SocialDraftListResponse{Items: rows}, "Successfully listed drafts")
}

// Update godoc
//
//	@Summary		Edit a social draft's text.
//	@Description	Replaces the text body of a draft social post. Only rows with status='draft' may be edited; the request is rejected with 409 if the row has already been published, errored, or cancelled.
//	@Tags			social
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int						true	"Social post ID"
//	@Param			body	body		SocialDraftUpdateRequest	true	"Replacement text"
//	@Success		200		{object}	social.Post				"Updated draft"
//	@Failure		400		{object}	api.MessageResponse		"Invalid request"
//	@Failure		404		{object}	api.MessageResponse		"Draft not found"
//	@Failure		409		{object}	api.MessageResponse		"Row is no longer a draft"
//	@Failure		500		{object}	api.MessageResponse		"Failed to update draft"
//	@Router			/social/drafts/{id} [patch]
func (h *Handler) Update(c *webkit.Context) error {
	ctx := c.Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}

	var body SocialDraftUpdateRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid request body")
	}
	text := strings.TrimSpace(body.Text)
	if text == "" {
		return api.Error(c, http.StatusBadRequest, "Text is required")
	}

	if ok, err := h.requireDraft(c, ctx, id); !ok {
		return err
	}

	updated, err := h.socialPosts.Update(ctx, id, social.PostUpdate{Text: &text})
	if errors.Is(err, store.ErrNotFound) {
		return api.Error(c, http.StatusNotFound, "Draft not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update draft", "id", id, "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to update draft")
	}
	return api.OK(c, http.StatusOK, updated, "Successfully updated draft")
}

// Cancel godoc
//
//	@Summary		Cancel a social draft so the publish cron skips it.
//	@Description	Transitions a draft row to status='cancelled'. The publish cron filters cancelled rows out and rotation idempotency counts them as "already handled", preventing tomorrow's build from regenerating the same subject.
//	@Tags			social
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int						true	"Social post ID"
//	@Success		200	{object}	social.Post				"Cancelled draft"
//	@Failure		400	{object}	api.MessageResponse		"Invalid request"
//	@Failure		404	{object}	api.MessageResponse		"Draft not found"
//	@Failure		409	{object}	api.MessageResponse		"Row is no longer a draft"
//	@Failure		500	{object}	api.MessageResponse		"Failed to cancel draft"
//	@Router			/social/drafts/{id}/cancel [post]
func (h *Handler) Cancel(c *webkit.Context) error {
	ctx := c.Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		return api.Error(c, http.StatusBadRequest, "ID must be a positive integer")
	}

	if ok, err := h.requireDraft(c, ctx, id); !ok {
		return err
	}

	cancelled := social.PostStatusCancelled
	updated, err := h.socialPosts.Update(ctx, id, social.PostUpdate{Status: &cancelled})
	if errors.Is(err, store.ErrNotFound) {
		return api.Error(c, http.StatusNotFound, "Draft not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "Failed to cancel draft", "id", id, "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to cancel draft")
	}
	return api.OK(c, http.StatusOK, updated, "Successfully cancelled draft")
}

// requireDraft fetches the row at id and writes a 404 / 409 / 500
// response if it is missing, no longer a draft, or unreachable. Returns
// (true, nil) when the caller should proceed; (false, err) when the
// caller must return err without further work (the response has already
// been written).
func (h *Handler) requireDraft(c *webkit.Context, ctx context.Context, id int64) (bool, error) {
	row, err := h.socialPosts.Find(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return false, api.Error(c, http.StatusNotFound, "Draft not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "Failed to load draft", "id", id, "err", err)
		return false, api.Error(c, http.StatusInternalServerError, "Failed to load draft")
	}
	if row.Status != social.PostStatusDraft {
		return false, api.Error(c, http.StatusConflict, "Row is no longer a draft")
	}
	return true, nil
}
