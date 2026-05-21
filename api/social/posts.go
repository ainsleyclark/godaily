package api

import (
	"context"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
)

// HandleSocialPosts is the Vercel serverless function entry point for GET /api/social/posts.
// Requires an issue_id query parameter.
func HandleSocialPosts(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		issueID := api.QueryInt(r, "issue_id", 0)
		if issueID < 1 {
			api.Error(w, http.StatusBadRequest, "issue_id is required")
			return
		}

		posts, err := a.Repository.SocialPosts.ListForIssue(ctx, issueID)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "failed to list social posts")
			return
		}

		api.JSON(w, http.StatusOK, posts)
	})(w, r)
}
