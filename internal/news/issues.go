package news

import (
	"context"
	"time"
)

// Issue defines an issue of go daily that contains a collection
// of news articles.
type Issue struct {
	ID       int64       `json:"id"`
	Slug     string      `json:"slug"`
	Subject  string      `json:"subject"`
	Status   IssueStatus `json:"status"`
	HtmlBody string      `json:"html_body"`
	TextBody string      `json:"text_body"`
	Summary  string      `json:"summary,omitzero"`
	SentAt   time.Time   `json:"sent_at"`
	Items    []Item      `json:"items"`
}

// IssueRepository defines the methods for interacting with the Issue store.
type IssueRepository interface {
	Find(ctx context.Context, id int64) (Issue, error)
	FindBySlug(ctx context.Context, slug string) (Issue, error)
	List(ctx context.Context) ([]Issue, error)
	Create(ctx context.Context, issue Issue) (Issue, error)
	Count(ctx context.Context) (int64, error)
}

// IssueStatus defines the state of an issue.
type IssueStatus string

// IssueStatus constants.
const (
	IssueStatusDraft IssueStatus = "draft"
	IssueStatusSent  IssueStatus = "sent"
	IssueStatusError IssueStatus = "error"
)

// String implements fmt.Stringer on the IssueStatus type.
func (s IssueStatus) String() string {
	return string(s)
}
