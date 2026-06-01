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
	"errors"
	"log/slog"
	"net/http"

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
