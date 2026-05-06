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

package issues

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/ainsleyclark/godaily/internal/store/internal/sqlc"
)

// New creates a new reviews Store.
func New(db *sql.DB) *Store {
	return &Store{
		sqlc: sqlc.New(db),
		db:   db,
	}
}

// Store provides methods for interacting with review data
// in the database.
type Store struct {
	sqlc *sqlc.Queries
	db   *sql.DB
}

var _ news.IssueRepository = (*Store)(nil)

func (s Store) Find(ctx context.Context, id int64) (news.Issue, error) {
	i, err := s.sqlc.IssueByID(ctx, id)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return news.Issue{}, store.ErrNotFound
	} else if err != nil {
		return news.Issue{}, err
	}
	return s.withItems(ctx, i)
}

func (s Store) FindBySlug(ctx context.Context, slug string) (news.Issue, error) {
	i, err := s.sqlc.IssueBySlug(ctx, slug)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return news.Issue{}, store.ErrNotFound
	} else if err != nil {
		return news.Issue{}, err
	}
	return s.withItems(ctx, i)
}

func (s Store) withItems(ctx context.Context, i sqlc.Issue) (news.Issue, error) {
	rows, err := s.sqlc.ItemListByIssue(ctx, i.ID)
	if err != nil {
		return news.Issue{}, err
	}
	items := make([]sqlc.Item, len(rows))
	copy(items, rows)
	return issueFromRows(i, items), nil
}

func (s Store) List(ctx context.Context) ([]news.Issue, error) {
	rows, err := s.sqlc.IssueList(ctx, sqlc.IssueListParams{Limit: 10000, Offset: 0})
	if err != nil {
		return nil, err
	}
	out := make([]news.Issue, len(rows))
	for i, r := range rows {
		out[i] = issueFromRows(r, nil)
	}
	return out, nil
}

func (s Store) Create(ctx context.Context, issue news.Issue) (news.Issue, error) {
	i, err := s.sqlc.IssueCreate(ctx, sqlc.IssueCreateParams{
		Slug:    issue.Slug,
		SentAt:  issue.SentAt,
		Subject: issue.Subject,
		Summary: sql.NullString{String: issue.Summary, Valid: true},
		Status:  issue.Status.String(),
	})
	if err != nil {
		return news.Issue{}, err
	}

	return issueFromRows(i, nil), nil
}

func (s Store) UpdateStatus(ctx context.Context, id int64, status news.IssueStatus, sentAt time.Time) (news.Issue, error) {
	i, err := s.sqlc.IssueUpdateStatus(ctx, sqlc.IssueUpdateStatusParams{
		ID:     id,
		Status: status.String(),
		SentAt: sentAt,
	})
	if err != nil {
		return news.Issue{}, err
	}

	return issueFromRows(i, nil), nil
}

func (s Store) Count(ctx context.Context) (int64, error) {
	count, err := s.sqlc.IssueCount(ctx)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return count, nil
}

func issueFromRows(i sqlc.Issue, rawItems []sqlc.Item) news.Issue {
	out := news.Issue{
		ID:      i.ID,
		Slug:    i.Slug,
		Subject: i.Subject,
		Status:  news.IssueStatus(i.Status),
		Summary: i.Summary.String,
		SentAt:  i.SentAt,
		Items:   make([]news.Item, 0, len(rawItems)),
	}
	for _, it := range rawItems {
		out.Items = append(out.Items, transformItem(it))
	}
	return out
}

func transformItem(i sqlc.Item) news.Item {
	out := news.Item{
		ID:      i.ID,
		Source:  news.Source(i.Source),
		Title:   i.Title,
		URL:     i.Url,
		Snippet: i.Summary.String,
		Score:   i.Score.Float64,
	}
	if i.AuthorName.Valid || i.AuthorUsername.Valid || i.AuthorAvatarUrl.Valid || i.AuthorProfileUrl.Valid {
		out.Author = &news.Author{
			Name:       i.AuthorName.String,
			Username:   i.AuthorUsername.String,
			AvatarURL:  i.AuthorAvatarUrl.String,
			ProfileURL: i.AuthorProfileUrl.String,
		}
	}
	return out
}
