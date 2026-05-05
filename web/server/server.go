package server

import (
	"fmt"

	godaily "github.com/ainsleyclark/godaily/internal"
	"github.com/ainsleyclark/godaily/web/handlers"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Start boots the HTTP server on the given port.
func Start(a *godaily.App, port string) error {
	kit := webkit.New()

	kit.Get("/", handlers.Home(a))
	kit.Static("/assets/", "web/dist/") // From where main.go is

	return kit.Start(fmt.Sprintf(":%s", port))
}
