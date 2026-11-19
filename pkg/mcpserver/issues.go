// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

const (
	// issueKeyLatest resolves to the most recently sent issue.
	issueKeyLatest = "latest"
	// issueKeyDraft resolves to the most recent draft issue.
	issueKeyDraft = "draft"
)

// issueKeyDescription documents the polymorphic issue key argument.
const issueKeyDescription = `Issue identifier: a numeric ID, a date slug (YYYY-MM-DD), "latest" for the most recently sent issue, or "draft" for the most recent draft.`

type (
	// ListIssuesInput is the input for list_issues.
	ListIssuesInput struct {
		Status  string `json:"status,omitempty"   jsonschema:"Only return issues with this status. Omit for all statuses."`
		Page    int64  `json:"page,omitempty"     jsonschema:"1-based page number. Defaults to 1."`
		PerPage int64  `json:"per_page,omitempty" jsonschema:"Issues per page, at most 100. Defaults to 20."`
	}
	// IssueSummary is an issue without its items, as returned by list_issues.
	IssueSummary struct {
		ID      int64     `json:"id"`
		Slug    string    `json:"slug"`
		Subject string    `json:"subject"`
		Status  string    `json:"status"`
		Summary string    `json:"summary,omitempty"`
		SentAt  time.Time `json:"sent_at"`
	}
	// ListIssuesOutput is the output for list_issues.
	ListIssuesOutput struct {
		Issues  []IssueSummary `json:"issues"`
		Page    int64          `json:"page"`
		PerPage int64          `json:"per_page"`
		Total   int64          `json:"total"`
	}

	// GetIssueInput is the input for get_issue.
	GetIssueInput struct {
		Key string `json:"key" jsonschema:"Issue identifier: a numeric ID, a date slug (YYYY-MM-DD), latest or draft."`
	}

	// IssueCandidatesInput is the input for get_issue_candidates.
	IssueCandidatesInput struct {
		Key   string `json:"key"             jsonschema:"Issue identifier: a numeric ID, a date slug (YYYY-MM-DD), latest or draft."`
		Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of candidates to return, at most 100. Defaults to 25."`
	}
	// ItemsOutput wraps a list of news items.
	ItemsOutput struct {
		Items []news.Item `json:"items"`
	}
)

func (t *tools) registerIssues(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_issues",
		Description: "List GoDaily newsletter issues, newest first, without their items. Use get_issue to fetch an issue's full contents.",
		InputSchema: inputSchema[ListIssuesInput](map[string][]string{
			"status": {digest.IssueStatusDraft.String(), digest.IssueStatusSent.String(), digest.IssueStatusError.String()},
		}),
		Annotations: readOnly("List issues"),
	}, t.listIssues)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_issue",
		Description: "Fetch a single GoDaily issue including its subject, summary, status, send time and ordered news items. " + issueKeyDescription,
		Annotations: readOnly("Get issue"),
	}, t.getIssue)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_issue_candidates",
		Description: "List the news items collected in an issue's build window that were NOT included in the issue, ranked by score. Useful for seeing what was left out. " + issueKeyDescription,
		Annotations: readOnly("Get issue candidates"),
	}, t.issueCandidates)
}

func (t *tools) listIssues(ctx context.Context, _ *mcp.CallToolRequest, in ListIssuesInput) (*mcp.CallToolResult, ListIssuesOutput, error) {
	page, perPage := pagination(in.Page, in.PerPage)
	opts := digest.IssueListOptions{Page: page, PerPage: perPage}
	if in.Status != "" {
		status := digest.IssueStatus(in.Status)
		opts.Status = &status
	}

	total, err := t.deps.Issues.Count(ctx, opts)
	if err != nil {
		return nil, ListIssuesOutput{}, internalError(ctx, "count issues", err)
	}
	issues, err := t.deps.Issues.List(ctx, opts)
	if err != nil {
		return nil, ListIssuesOutput{}, internalError(ctx, "list issues", err)
	}

	out := ListIssuesOutput{
		Issues:  make([]IssueSummary, len(issues)),
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}
	for i, issue := range issues {
		out.Issues[i] = IssueSummary{
			ID:      issue.ID,
			Slug:    issue.Slug,
			Subject: issue.Subject,
			Status:  issue.Status.String(),
			Summary: issue.Summary,
			SentAt:  issue.SentAt,
		}
	}
	return nil, out, nil
}

func (t *tools) getIssue(ctx context.Context, _ *mcp.CallToolRequest, in GetIssueInput) (*mcp.CallToolResult, digest.Issue, error) {
	issue, err := t.resolveIssue(ctx, in.Key)
	if err != nil {
		return nil, digest.Issue{}, err
	}
	return nil, issue, nil
}

func (t *tools) issueCandidates(ctx context.Context, _ *mcp.CallToolRequest, in IssueCandidatesInput) (*mcp.CallToolResult, ItemsOutput, error) {
	issue, err := t.resolveIssue(ctx, in.Key)
	if err != nil {
		return nil, ItemsOutput{}, err
	}

	start, end := digest.BuildWindow(issue.SentAt)
	notInDigest := false
	items, err := t.deps.Items.List(ctx, news.ItemListOptions{
		From:     &start,
		To:       &end,
		InDigest: &notInDigest,
		Sort:     news.ItemSortTop,
		Page:     1,
		PerPage:  int64(clampLimit(in.Limit, 25)),
	})
	if err != nil {
		return nil, ItemsOutput{}, internalError(ctx, "list candidate items", err)
	}
	return nil, ItemsOutput{Items: items}, nil
}

// resolveIssue looks an issue up by numeric ID, date slug, or one of the
// "latest" / "draft" aliases. The returned issue always includes its items.
func (t *tools) resolveIssue(ctx context.Context, key string) (digest.Issue, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return digest.Issue{}, errors.New("key is required: pass an issue ID, a YYYY-MM-DD slug, latest or draft")
	}

	var (
		issue digest.Issue
		err   error
	)
	switch {
	case strings.EqualFold(key, issueKeyLatest):
		var latest []digest.Issue
		latest, err = t.deps.Issues.Latest(ctx, 1)
		if err == nil && len(latest) == 0 {
			return digest.Issue{}, errors.New("no issues have been sent yet")
		}
		if err == nil {
			issue = latest[0]
		}
	case strings.EqualFold(key, issueKeyDraft):
		status := digest.IssueStatusDraft
		var drafts []digest.Issue
		drafts, err = t.deps.Issues.List(ctx, digest.IssueListOptions{Status: &status, Page: 1, PerPage: 1})
		if err == nil && len(drafts) == 0 {
			return digest.Issue{}, errors.New("there is no draft issue")
		}
		if err == nil {
			// List omits items; re-fetch to include them.
			issue, err = t.deps.Issues.Find(ctx, drafts[0].ID)
		}
	default:
		if id, parseErr := strconv.ParseInt(key, 10, 64); parseErr == nil && id > 0 {
			issue, err = t.deps.Issues.Find(ctx, id)
		} else {
			issue, err = t.deps.Issues.FindBySlug(ctx, key)
		}
	}

	if errors.Is(err, store.ErrNotFound) {
		return digest.Issue{}, fmt.Errorf("issue %q not found", key)
	}
	if err != nil {
		return digest.Issue{}, internalError(ctx, "fetch issue", err)
	}
	return issue, nil
}

// internalError logs the underlying cause and returns a generic error for
// the tool result. Database errors are not surfaced verbatim to the client.
func internalError(ctx context.Context, op string, err error) error {
	slog.ErrorContext(ctx, "MCP tool failed to "+op, "err", err)
	return fmt.Errorf("failed to %s, please try again later", op)
}
