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
	pkgapi "github.com/ainsleyclark/godaily/pkg/api"
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
		stripped := http.StripPrefix("/api", mux.Handler(app))
		handler = pkgapi.Limiter.Limit(stripped.ServeHTTP)
	})
	handler.ServeHTTP(w, r)
}
