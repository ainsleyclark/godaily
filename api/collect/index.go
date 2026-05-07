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

// Package handler is the Vercel serverless function for GET /api/collect.
package handler

import (
	"log/slog"
	"net/http"
	"os"

	bootstrap "github.com/ainsleyclark/godaily/api/internal"
	"github.com/ainsleyclark/godaily/pkg/digest"
	"github.com/ainsleyclark/godaily/pkg/hook"
)

// Handler is the Vercel serverless function entry point.
func Handler(w http.ResponseWriter, r *http.Request) {
	bootstrap.Handle(w, r, func(runner digest.Runner) {
		handle(w, r, runner)
	})
}

func handle(w http.ResponseWriter, r *http.Request, runner digest.Runner) {
	ctx := r.Context()

	if _, err := runner.Collect(ctx, digest.CollectOptions{}); err != nil {
		slog.ErrorContext(ctx, "Collecting digest", "error", err)
		http.Error(w, "collect failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	hook.Heartbeat(ctx, os.Getenv("BETTERSTACK_COLLECT_HEARTBEAT_URL"))
	w.WriteHeader(http.StatusOK)
}
