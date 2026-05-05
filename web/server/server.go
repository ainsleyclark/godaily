package server

import (
	"fmt"
	"net/http"

	kitmiddleware "github.com/ainsleydev/webkit/pkg/middleware"

	godaily "github.com/ainsleyclark/godaily/internal"
	"github.com/ainsleyclark/godaily/web/handlers"
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
	kit.Static("/assets/", "web/dist/") // From where main.go is
	kit.NotFound(func(c *webkit.Context) error { return c.String(http.StatusNotFound, "Not Found") })

	if env.IsDevelopment() {
		// Register on the raw chi mux so SSE bypasses webkit's middleware chain
		// (Logger's ResponseRecorder doesn't implement http.Flusher, which would
		// buffer the stream).
		kit.Mux().Get("/internal/reload/", handlers.ReloadHTTP)
	}

	return kit.Start(fmt.Sprintf(":%s", port))
}
