// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"context"
	"errors"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// ErrIssueNotDraft is returned when attempting to mutate the editable
// fields of an issue that is no longer in draft status.
var ErrIssueNotDraft = errors.New("issue is not in draft status")

// Issue defines an issue of go daily that contains a collection
// of news articles.
type Issue struct {
	ID      int64       `json:"id"`
	Slug    string      `json:"slug"`
	Subject string      `json:"subject"`
	Status  IssueStatus `json:"status"`
	Summary string      `json:"summary,omitzero"`
	SentAt  time.Time   `json:"sent_at"`
	Items   []news.Item `json:"items"`
}

// IssueListOptions filters and paginates List/Count queries.
// A nil Status returns all statuses; pagination follows store.ListOptions semantics.
type IssueListOptions struct {
	Status  *IssueStatus
	Page    int64
	PerPage int64
}

func (o IssueListOptions) listOpts() store.ListOptions {
	return store.ListOptions{Page: o.Page, PerPage: o.PerPage}
}

// Limit returns the SQL LIMIT value for this page.
func (o IssueListOptions) Limit() int64 { return o.listOpts().Limit() }

// Offset returns the SQL OFFSET value for the current page.
func (o IssueListOptions) Offset() int64 { return o.listOpts().Offset() }

//go:generate go run go.uber.org/mock/mockgen -package=mockdigest -destination=../../mocks/digest/IssueRepository.go . IssueRepository

// IssueRepository defines the methods for interacting with the Issue store.
type IssueRepository interface {
	Find(ctx context.Context, id int64) (Issue, error)
	FindBySlug(ctx context.Context, slug string) (Issue, error)
	List(ctx context.Context, opts IssueListOptions) ([]Issue, error)
	Latest(ctx context.Context, limit int) ([]Issue, error)
	Create(ctx context.Context, issue Issue) (Issue, error)
	Delete(ctx context.Context, id int64) (Issue, error)
	Update(ctx context.Context, issue Issue) (Issue, error)
	UpdateStatus(ctx context.Context, id int64, status IssueStatus, sentAt time.Time) (Issue, error)
	Count(ctx context.Context, opts IssueListOptions) (int64, error)
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
