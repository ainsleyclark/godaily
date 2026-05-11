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
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package api

import (
	"context"
	"log"
	"net/http"
	"sync"

	godaily "github.com/ainsleyclark/godaily/pkg"
)

var (
	app   *godaily.App
	appMu sync.RWMutex
)

// SetApp sets the singleton App. Safe for concurrent use, allowing parallel tests
// to inject mocks without data races.
func SetApp(a *godaily.App) {
	appMu.Lock()
	defer appMu.Unlock()
	app = a
}

// GetApp returns the singleton App, bootstrapping it on first call.
func GetApp(ctx context.Context) *godaily.App {
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
type AppHandler func(w http.ResponseWriter, r *http.Request, ctx context.Context, a *godaily.App)

// Handle applies the standard API middleware chain to next, injecting the
// request context and bootstrapped App.
func Handle(next AppHandler) http.HandlerFunc {
	return Limiter.Limit(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		next(w, r, ctx, GetApp(ctx))
	})
}
