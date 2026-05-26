// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/store/internal/sqlc"
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

var _ digest.IssueRepository = (*Store)(nil)

func (s Store) Find(ctx context.Context, id int64) (digest.Issue, error) {
	i, err := s.sqlc.IssueByID(ctx, id)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return digest.Issue{}, store.ErrNotFound
	} else if err != nil {
		return digest.Issue{}, err
	}
	return s.withItems(ctx, i)
}

func (s Store) FindBySlug(ctx context.Context, slug string) (digest.Issue, error) {
	i, err := s.sqlc.IssueBySlug(ctx, slug)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return digest.Issue{}, store.ErrNotFound
	} else if err != nil {
		return digest.Issue{}, err
	}
	return s.withItems(ctx, i)
}

func (s Store) withItems(ctx context.Context, i sqlc.Issue) (digest.Issue, error) {
	rows, err := s.sqlc.ItemListByIssue(ctx, sql.NullInt64{Int64: i.ID, Valid: true})
	if err != nil {
		return digest.Issue{}, err
	}
	items := make([]sqlc.Item, len(rows))
	copy(items, rows)
	return issueFromRows(i, items), nil
}

func (s Store) List(ctx context.Context, opts store.ListOptions) ([]digest.Issue, error) {
	rows, err := s.sqlc.IssueList(ctx, sqlc.IssueListParams{Limit: opts.Limit(), Offset: opts.Offset()})
	if err != nil {
		return nil, err
	}
	out := make([]digest.Issue, len(rows))
	for i, r := range rows {
		out[i] = issueFromRows(r, nil)
	}
	return out, nil
}

func (s Store) Latest(ctx context.Context, limit int) ([]digest.Issue, error) {
	if limit <= 0 {
		return nil, nil
	}

	rows, err := s.sqlc.IssueList(ctx, sqlc.IssueListParams{
		Limit:  int64(limit),
		Offset: 0,
	})
	if err != nil {
		return nil, err
	}

	out := make([]digest.Issue, 0, len(rows))
	for _, r := range rows {
		issue, err := s.withItems(ctx, r)
		if err != nil {
			return nil, err
		}
		out = append(out, issue)
	}

	return out, nil
}

func (s Store) Create(ctx context.Context, issue digest.Issue) (digest.Issue, error) {
	i, err := s.sqlc.IssueCreate(ctx, sqlc.IssueCreateParams{
		Slug:    issue.Slug,
		SentAt:  issue.SentAt,
		Subject: issue.Subject,
		Summary: sql.NullString{String: issue.Summary, Valid: true},
		Status:  issue.Status.String(),
	})
	if err != nil {
		return digest.Issue{}, err
	}

	return issueFromRows(i, nil), nil
}

func (s Store) Delete(ctx context.Context, id int64) (digest.Issue, error) {
	i, err := s.sqlc.IssueByID(ctx, id)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return digest.Issue{}, store.ErrNotFound
	} else if err != nil {
		return digest.Issue{}, err
	}
	_, err = s.db.ExecContext(ctx, "DELETE FROM issues WHERE id = ?", id)
	if err != nil {
		return digest.Issue{}, err
	}
	return issueFromRows(i, nil), nil
}

func (s Store) UpdateStatus(ctx context.Context, id int64, status digest.IssueStatus, sentAt time.Time) (digest.Issue, error) {
	i, err := s.sqlc.IssueUpdateStatus(ctx, sqlc.IssueUpdateStatusParams{
		ID:     id,
		Status: status.String(),
		SentAt: sentAt,
	})
	if err != nil {
		return digest.Issue{}, err
	}

	return issueFromRows(i, nil), nil
}

func (s Store) ListByStatus(ctx context.Context, status digest.IssueStatus, opts store.ListOptions) ([]digest.Issue, error) {
	rows, err := s.db.QueryContext(
		ctx,
		"SELECT id, slug, sent_at, subject, COALESCE(summary,''), status FROM issues WHERE status = ? ORDER BY sent_at DESC LIMIT ? OFFSET ?",
		status.String(), opts.Limit(), opts.Offset(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []digest.Issue
	for rows.Next() {
		var (
			i      digest.Issue
			sentAt time.Time
		)
		if err := rows.Scan(&i.ID, &i.Slug, &sentAt, &i.Subject, &i.Summary, &i.Status); err != nil {
			return nil, err
		}
		i.SentAt = sentAt
		i.Items = []news.Item{}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s Store) CountByStatus(ctx context.Context, status digest.IssueStatus) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM issues WHERE status = ?", status.String()).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
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

func issueFromRows(i sqlc.Issue, rawItems []sqlc.Item) digest.Issue {
	out := digest.Issue{
		ID:      i.ID,
		Slug:    i.Slug,
		Subject: i.Subject,
		Status:  digest.IssueStatus(i.Status),
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
		Tag:     news.Tag(i.Tag),
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
