// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"net/http"

	kitmiddleware "github.com/ainsleydev/webkit/pkg/middleware"

	"github.com/ainsleyclark/godaily/pkg/api/mux"

	"github.com/ainsleydev/webkit/pkg/env"
	"github.com/ainsleydev/webkit/pkg/webkit"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/web/handlers"
	"github.com/ainsleyclark/godaily/web/views/pages"
)

// Handler returns the configured HTTP handler for the web server without
// starting it. Useful for composing with additional routes in tests.
func Handler(a *godaily.App) http.Handler {
	return newKit(a)
}

// Start boots the HTTP server on the given port.
func Start(a *godaily.App, port string) error {
	return newKit(a).Start(fmt.Sprintf(":%s", port))
}

func newKit(a *godaily.App) *webkit.Kit {
	kit := webkit.New()

	kit.Plug(kitmiddleware.NonWWWRedirect)
	kit.Plug(kitmiddleware.TrailingSlashRedirect)
	kit.Plug(kitmiddleware.Logger)
	kit.Plug(kitmiddleware.Recover)
	kit.Plug(kitmiddleware.RequestID)
	kit.Plug(kitmiddleware.Gzip)
	kit.Plug(kitmiddleware.URL)

	kit.Get("/", handlers.Home(a))
	kit.Get("/thank-you/", handlers.ThankYou())
	kit.Get("/confirmed/", handlers.Confirmed(a))
	kit.Get("/unsubscribed/", handlers.Unsubscribed())
	kit.Get("/privacy/", handlers.Privacy())
	kit.Get("/issues/", handlers.Issues(a))
	kit.Get("/issues/{slug}/", handlers.Digest(a))
	kit.Get("/browse/", handlers.Browse(a))
	kit.Get("/browse/{tag}/", handlers.BrowseTag(a))
	kit.Static("/assets/", "web/dist/") // From where main.go is
	kit.NotFound(func(c *webkit.Context) error {
		return c.RenderWithStatus(http.StatusNotFound, pages.Error(http.StatusNotFound))
	})

	// Mount API routes on the raw chi mux so they bypass webkit's middleware chain.
	kit.Mux().Mount("/api", http.StripPrefix("/api", mux.Handler(a)))

	if env.IsDevelopment() {
		// Register on the raw chi mux so SSE bypasses webkit's middleware chain
		// (Logger's ResponseRecorder doesn't implement http.Flusher, which would
		// buffer the stream).
		kit.Mux().Get("/internal/reload/", handlers.ReloadHTTP)
	}

	return kit
}
