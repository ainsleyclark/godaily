package handlers

import (
	godaily "github.com/ainsleyclark/godaily/internal"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

func Home(a *godaily.App) webkit.Handler {
	return func(c *webkit.Context) error {
		return c.Render(pages.Home())
	}
}
