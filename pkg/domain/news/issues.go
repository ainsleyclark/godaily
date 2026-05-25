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

package news

import (
	"context"
	"time"
)

// Issue defines an issue of go daily that contains a collection
// of news articles.
type Issue struct {
	ID      int64       `json:"id"`
	Slug    string      `json:"slug"`
	Subject string      `json:"subject"`
	Status  IssueStatus `json:"status"`
	Summary string      `json:"summary,omitzero"`
	SentAt  time.Time   `json:"sent_at"`
	Items   []Item      `json:"items"`
}

//go:generate go run go.uber.org/mock/mockgen -package=mocknews -destination=../../mocks/news/IssueRepository.go . IssueRepository

// IssueRepository defines the methods for interacting with the Issue store.
type IssueRepository interface {
	Find(ctx context.Context, id int64) (Issue, error)
	FindBySlug(ctx context.Context, slug string) (Issue, error)
	List(ctx context.Context, opts ListOptions) ([]Issue, error)
	ListByStatus(ctx context.Context, status IssueStatus, opts ListOptions) ([]Issue, error)
	Latest(ctx context.Context, limit int) ([]Issue, error)
	Create(ctx context.Context, issue Issue) (Issue, error)
	Delete(ctx context.Context, id int64) (Issue, error)
	UpdateStatus(ctx context.Context, id int64, status IssueStatus, sentAt time.Time) (Issue, error)
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status IssueStatus) (int64, error)
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
