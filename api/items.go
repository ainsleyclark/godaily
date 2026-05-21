package api

import (
	"context"
	"net/http"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

// HandleItems is the Vercel serverless function entry point for GET /api/items.
// Supports optional filtering with issue_id, from and to query parameters.
func HandleItems(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		var opts news.ItemListOptions

		issueID := api.QueryInt(r, "issue_id", 0)
		if issueID > 0 {
			opts.IssueID = &issueID
		}

		if raw := r.URL.Query().Get("from"); raw != "" {
			from, err := time.Parse(time.DateOnly, raw)
			if err != nil {
				api.Error(w, http.StatusBadRequest, "from must be in YYYY-MM-DD format")
				return
			}
			opts.From = &from
		}

		if raw := r.URL.Query().Get("to"); raw != "" {
			to, err := time.Parse(time.DateOnly, raw)
			if err != nil {
				api.Error(w, http.StatusBadRequest, "to must be in YYYY-MM-DD format")
				return
			}
			opts.To = &to
		}

		items, err := a.Repository.Items.List(ctx, opts)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "failed to list items")
			return
		}

		api.JSON(w, http.StatusOK, items)
	})(w, r)
}
