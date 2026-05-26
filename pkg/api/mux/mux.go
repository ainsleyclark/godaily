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

// Package mux wires all API handler functions into a single http.Handler.
// It lives outside api/ so Vercel's serverless function glob ("api/**/*.go")
// does not pick it up as a function entry point.
package mux

import (
	"errors"
	"net/http"
	"strings"

	godaily "github.com/ainsleyclark/godaily/pkg"
	pkgapi "github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/api/plugs"
	handlers "github.com/ainsleyclark/godaily/pkg/api/handlers"
	digesthandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/digest"
	issuehandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/issues"
	itemhandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/items"
	metricshandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/metrics"
	socialhandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/social"
	webhookhandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/webhooks"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Handler returns an http.Handler for all API routes.
// Routes are relative to /api — callers should mount or strip the /api prefix
// before dispatching here.
func Handler(app *godaily.App) http.Handler {
	kit := webkit.New()

	kit.ErrorHandler = func(c *webkit.Context, err error) error {
		var e *webkit.Error
		if errors.As(err, &e) {
			pkgapi.Error(c.Response, e.Code, e.Message)
			return nil
		}
		pkgapi.Error(c.Response, http.StatusInternalServerError, err.Error())
		return nil
	}

	kit.Plug(plugs.RateLimit(pkgapi.Limiter))
	auth := plugs.Auth(app.Config.APISecret)

	digestH := digesthandlers.New(app)
	metricsH := metricshandlers.New(app)
	socialH := socialhandlers.New(app)
	issuesH := issuehandlers.New(app)
	itemsH := itemhandlers.New(app)
	webhookH := webhookhandlers.New(app)

	kit.Get("/healthz", handlers.Healthz)

	kit.Group("/digest", func(k *webkit.Kit) { digestH.Routes(k, auth) })
	kit.Group("/metrics", func(k *webkit.Kit) { metricsH.Routes(k, auth) })
	kit.Group("/social", func(k *webkit.Kit) { socialH.Routes(k, auth) })
	kit.Group("/issues", func(k *webkit.Kit) { issuesH.Routes(k, auth) })
	kit.Group("/items", func(k *webkit.Kit) { itemsH.Routes(k, auth) })
	kit.Group("/webhooks", func(k *webkit.Kit) { webhookH.Routes(k) })

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Vercel's trailingSlash:true setting appends a slash to every URL.
		// Strip it here so chi's patterns resolve correctly.
		if p := r.URL.Path; p != "/" && strings.HasSuffix(p, "/") {
			r2 := r.Clone(r.Context())
			r2.URL.Path = p[:len(p)-1]
			r = r2
		}
		kit.ServeHTTP(w, r)
	})
}
