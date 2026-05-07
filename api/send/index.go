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

// Package handler is the Vercel serverless function for GET /api/send.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/digest"
)

// Handler is the Vercel serverless function entry point.
func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	app, teardown, err := godaily.Bootstrap(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Bootstrapping app", "error", err)
		http.Error(w, "failed to bootstrap app: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer teardown()

	handle(w, r, app.Runner)
}

func handle(w http.ResponseWriter, r *http.Request, runner digest.Runner) {
	ctx := r.Context()
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)

	if err := runner.SendDigest(ctx, yesterday, false); err != nil {
		slog.ErrorContext(ctx, "Sending digest", "error", err)
		http.Error(w, "send digest failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := runner.SendSuggestion(ctx, yesterday); err != nil {
		slog.ErrorContext(ctx, "Sending suggestion", "error", err)
		http.Error(w, "send suggestion failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	triggerDeploy(ctx, os.Getenv("VERCEL_DEPLOY_HOOK_URL"))
	pingHeartbeat(ctx, os.Getenv("BETTERSTACK_SEND_HEARTBEAT_URL"))
	w.WriteHeader(http.StatusOK)
}

func triggerDeploy(ctx context.Context, url string) {
	if url == "" {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(""))
	if err != nil {
		slog.ErrorContext(ctx, "Creating deploy hook request", "error", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "Triggering deploy hook", "error", err)
		return
	}
	resp.Body.Close()
}

func pingHeartbeat(ctx context.Context, url string) {
	if url == "" {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.ErrorContext(ctx, "Creating heartbeat request", "error", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "Pinging heartbeat", "error", err)
		return
	}
	resp.Body.Close()
}
