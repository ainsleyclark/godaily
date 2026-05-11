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

package api

import (
	"context"
	"log"
	"net/http"
	"sync"

	godaily "github.com/ainsleyclark/godaily/pkg"
)

type appContextKey struct{}

var (
	app   *godaily.App
	appMu sync.RWMutex
)

// WithApp stores a into ctx so that GetApp returns it without touching the global.
// Use this in tests to inject a mock app per request.
func WithApp(ctx context.Context, a *godaily.App) context.Context {
	return context.WithValue(ctx, appContextKey{}, a)
}

// SetApp sets the singleton App used in production.
func SetApp(a *godaily.App) {
	appMu.Lock()
	defer appMu.Unlock()
	app = a
}

// GetApp returns the App stored in ctx (injected via WithApp), falling back to
// the global singleton and bootstrapping it on first call if neither is set.
func GetApp(ctx context.Context) *godaily.App {
	if a, ok := ctx.Value(appContextKey{}).(*godaily.App); ok && a != nil {
		return a
	}
	appMu.RLock()
	a := app
	appMu.RUnlock()
	if a != nil {
		return a
	}
	appMu.Lock()
	defer appMu.Unlock()
	if app != nil {
		return app
	}
	var err error
	app, _, err = godaily.Bootstrap(ctx)
	if err != nil {
		log.Fatalf("bootstrapping app: %v", err)
	}
	return app
}

// AppHandler is an HTTP handler that receives the request context and the
// bootstrapped App alongside the standard response/request pair, so handlers
// do not need to call r.Context() or GetApp themselves.
type AppHandler func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App)

// HandleAuth is like Handle but rejects requests that fail authentication
// against the App's configured API secret.
func HandleAuth(next AppHandler) http.HandlerFunc {
	return Handle(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		if !Authenticated(r, a.Config.APISecret) {
			Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(ctx, w, r, a)
	})
}

// Handle applies the standard API middleware chain to next, injecting the
// request context and bootstrapped App. Rate limiting is skipped when the App
// has been injected via WithApp (i.e. in tests).
func Handle(next AppHandler) http.HandlerFunc {
	inner := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		next(ctx, w, r, GetApp(ctx))
	}
	limited := Limiter.Limit(inner)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(appContextKey{}) != nil {
			inner(w, r)
		} else {
			limited(w, r)
		}
	}
}
