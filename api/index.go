// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package api is the single Vercel serverless function entrypoint. All routes
// are dispatched through pkg/api/mux so Vercel only builds one binary.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api/mux"
)

var (
	handler     http.Handler
	handlerOnce sync.Once
)

// Handler is the Vercel serverless function entrypoint. sync.Once bootstraps
// the app once per cold start; Fluid Compute reuses the instance across
// invocations.
func Handler(w http.ResponseWriter, r *http.Request) {
	handlerOnce.Do(func() {
		ctx := context.Background()
		app, _, err := godaily.Bootstrap(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "bootstrapping app", "error", err)
			os.Exit(1)
		}
		handler = http.StripPrefix("/api", mux.Handler(app))
	})
	handler.ServeHTTP(w, r)
}
