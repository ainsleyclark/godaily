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

package server

import (
	"fmt"
	"net/http"

	kitmiddleware "github.com/ainsleydev/webkit/pkg/middleware"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/web/handlers"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/ainsleydev/webkit/pkg/env"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Start boots the HTTP server on the given port.
func Start(a *godaily.App, port string) error {
	kit := webkit.New()

	kit.Plug(kitmiddleware.NonWWWRedirect)
	kit.Plug(kitmiddleware.TrailingSlashRedirect)
	kit.Plug(kitmiddleware.Logger)
	kit.Plug(kitmiddleware.Recover)
	kit.Plug(kitmiddleware.RequestID)
	kit.Plug(kitmiddleware.Gzip)
	kit.Plug(kitmiddleware.URL)

	kit.Get("/", handlers.Home(a))
	kit.Get("/thank-you/", handlers.ThankYou(a))
	kit.Get("/unsubscribed/", handlers.Unsubscribed())
	kit.Get("/issues/", handlers.Issues(a))
	kit.Get("/issues/{slug}/", handlers.Digest(a))
	kit.Static("/assets/", "web/dist/") // From where main.go is
	kit.NotFound(func(c *webkit.Context) error {
		return c.RenderWithStatus(http.StatusNotFound, pages.Error(http.StatusNotFound))
	})

	if env.IsDevelopment() {
		// Register on the raw chi mux so SSE bypasses webkit's middleware chain
		// (Logger's ResponseRecorder doesn't implement http.Flusher, which would
		// buffer the stream).
		kit.Mux().Get("/internal/reload/", handlers.ReloadHTTP)
	}

	return kit.Start(fmt.Sprintf(":%s", port))
}
