// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mux wires all API handler functions into a single http.Handler.
// It lives outside api/ so Vercel's serverless function glob ("api/**/*.go")
// does not pick it up as a function entry point.
package mux

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api/handlers"
	authhandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/auth"
	digesthandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/digest"
	issuehandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/issues"
	itemhandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/items"
	metricshandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/metrics"
	socialhandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/social"
	webhookhandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/webhooks"
	"github.com/ainsleyclark/godaily/pkg/api/plugs"
	"github.com/ainsleydev/webkit/pkg/middleware"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Handler returns an http.Handler for all API routes.
// Routes are relative to /api — callers should mount or strip the /api prefix
// before dispatching here.
func Handler(app *godaily.App) http.Handler {
	kit := webkit.New()
	// CORS runs as chi middleware (not a webkit.Plug) so OPTIONS preflight
	// requests for routes that only register GET reach it before chi's 405.
	kit.Mux().Use(plugs.CORS())
	kit.Plug(plugs.RateLimit(plugs.Limiter, app.Config.APISecret))
	kit.Plug(middleware.Logger)

	kit.ErrorHandler = func(c *webkit.Context, err error) error {
		slog.ErrorContext(c.Context(), "Request failed: "+err.Error())
		var e *webkit.Error
		if errors.As(err, &e) {
			return c.JSON(e.Code, map[string]any{
				"code":    e.Code,
				"message": e.Message,
			})
			// pkgapi.Error(c.Response, e.Code, e.Message)
		}
		return c.JSON(http.StatusInternalServerError, err.Error())
		// pkgapi.Error(c.Response, http.StatusInternalServerError, err.Error())
	}

	auth := plugs.Auth(app.Config.APISecret)

	digestH := digesthandlers.New(app)
	metricsH := metricshandlers.New(app)
	socialH := socialhandlers.New(app)
	issuesH := issuehandlers.New(app)
	itemsH := itemhandlers.New(app)
	webhookH := webhookhandlers.New(app)
	authH := authhandlers.New(app)

	kit.Get("/healthz", handlers.HealthZ)

	kit.Group("/auth", func(k *webkit.Kit) { authH.Routes(k) })

	kit.Post("/subscribe", digestH.Subscribe)
	kit.Get("/confirm", digestH.Confirm)
	kit.Get("/unsubscribe", digestH.Unsubscribe)
	kit.Post("/unsubscribe", digestH.Unsubscribe)

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
