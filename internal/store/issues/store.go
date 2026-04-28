package issues

import (
	"context"
	"database/sql"
	"errors"

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

	return transformIssue(i), nil
}

func (s Store) FindBySlug(ctx context.Context, slug string) (news.Issue, error) {
	i, err := s.sqlc.IssueBySlug(ctx, slug)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return news.Issue{}, store.ErrNotFound
	} else if err != nil {
		return news.Issue{}, err
	}

	return transformIssue(i), nil
}

func (s Store) List(ctx context.Context) ([]news.Issue, error) {
	return nil, nil
}

func (s Store) Create(ctx context.Context, issue news.Issue) (news.Issue, error) {
	i, err := s.sqlc.IssueCreate(ctx, sqlc.IssueCreateParams{
		Slug:     issue.Slug,
		SentAt:   issue.SentAt,
		Subject:  issue.Subject,
		Summary:  sql.NullString{String: issue.Summary, Valid: true},
		HtmlBody: issue.HtmlBody,
		TextBody: issue.TextBody,
		Status:   issue.Status.String(),
	})

	if err != nil {
		return news.Issue{}, err
	}

	return transformIssue(i), nil
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

func transformIssue(i sqlc.Issue) news.Issue {
	return news.Issue{
		ID:       i.ID,
		Slug:     i.Slug,
		Subject:  i.Subject,
		Status:   news.IssueStatus(i.Status),
		HtmlBody: i.HtmlBody,
		TextBody: i.TextBody,
		Summary:  i.Summary.String,
		SentAt:   i.SentAt,
		Items:    nil,
	}
}
