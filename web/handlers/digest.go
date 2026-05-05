package handlers

import (
	"errors"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/internal"
	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Digest handles individual news issues.
func Digest(a *godaily.App) webkit.Handler {
	return func(c *webkit.Context) error {
		ctx := c.Context()

		issue, err := a.Repository.Issues.FindBySlug(ctx, c.Param("slug"))
		if err != nil && errors.Is(err, store.ErrNotFound) {
			return c.Render(pages.Error(http.StatusNotFound))
		} else if err != nil {
			return c.Render(pages.Error(http.StatusInternalServerError))
		}

		return c.Render(pages.Digest(issue))
	}
}
