// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package api contains Vercel serverless function handlers.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/bootstrap"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/welcome"
)

// HandleSubscribe is the Vercel serverless function entry point for POST /api/subscribe.
func HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	bootstrap.Handle(w, r, func(app *godaily.App) {
		handleSubscribe(w, r, app)
	})
}

func handleSubscribe(w http.ResponseWriter, r *http.Request, app *godaily.App) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "email is required"})
		return
	}

	ctx := r.Context()
	repo := app.Repository.Subscribers

	_, err := repo.FindByEmail(ctx, body.Email)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "already subscribed"})
		return
	}
	if !errors.Is(err, store.ErrNotFound) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sub, err := repo.Create(ctx, body.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send welcome email — non-fatal; a failed email must not block the subscription.
	var latestIssueURL, latestIssueTitle string
	if latest, err := app.Repository.Issues.Latest(ctx, 1); err == nil && len(latest) > 0 {
		latestIssueURL = env.AppURL + "/digest/" + latest[0].Slug + "/"
		latestIssueTitle = latest[0].Subject
	}
	unsubURL := env.AppURL + "/api/unsubscribe?token=" + sub.UnsubscribeToken
	if err := welcome.Send(ctx, app.Email, sub.Email, unsubURL, latestIssueURL, latestIssueTitle); err != nil {
		slog.ErrorContext(ctx, "Failed to send welcome email", "email", sub.Email, "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
