package api

import (
	"context"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

type emailStatsResponse struct {
	Stats    engagement.IssueStats   `json:"stats"`
	TopLinks []engagement.LinkClicks `json:"top_links"`
}

// HandleEmailStats is the Vercel serverless function entry point for GET /api/email/stats.
// Requires issue_id and accepts optional link_limit query params.
func HandleEmailStats(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		issueID := api.QueryInt(r, "issue_id", 0)
		if issueID < 1 {
			api.Error(w, http.StatusBadRequest, "issue_id is required")
			return
		}

		stats, err := a.Repository.EmailEvents.IssueStats(ctx, issueID)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "failed to fetch email stats")
			return
		}

		limit := api.QueryInt(r, "link_limit", 10)
		if limit < 1 || limit > 100 {
			limit = 10
		}

		topLinks, err := a.Repository.EmailEvents.TopLinks(ctx, issueID, limit)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "failed to fetch top links")
			return
		}

		api.JSON(w, http.StatusOK, emailStatsResponse{Stats: stats, TopLinks: topLinks})
	})(w, r)
}
