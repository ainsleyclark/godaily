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

package items

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtypes"
	"github.com/ainsleyclark/godaily/pkg/store/internal/sqlc"
)

// New creates a new items Store.
func New(db *sql.DB) *Store {
	return &Store{
		sqlc: sqlc.New(db),
		db:   db,
	}
}

// Store provides methods for interacting with item data
// in the database.
type Store struct {
	sqlc *sqlc.Queries
	db   *sql.DB
}

var _ news.ItemRepository = (*Store)(nil)

func (s Store) Find(ctx context.Context, id int64) (news.Item, error) {
	i, err := s.sqlc.ItemByID(ctx, id)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return news.Item{}, store.ErrNotFound
	} else if err != nil {
		return news.Item{}, err
	}

	return transformItem(i), nil
}

func (s Store) List(ctx context.Context, opts news.ItemListOptions) ([]news.Item, error) {
	if opts.IssueID != nil {
		return s.listByIssue(ctx, *opts.IssueID)
	}
	if opts.From != nil && opts.To != nil {
		return s.listByDateRange(ctx, *opts.From, *opts.To)
	}
	return nil, fmt.Errorf("items.List: at least one filter (IssueID or From/To) must be set")
}

func (s Store) listByIssue(ctx context.Context, issueID int64) ([]news.Item, error) {
	rows, err := s.sqlc.ItemListByIssue(ctx, sql.NullInt64{Int64: issueID, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]news.Item, 0, len(rows))
	for _, r := range rows {
		out = append(out, transformItem(r))
	}
	return out, nil
}

func (s Store) listByDateRange(ctx context.Context, from, to time.Time) ([]news.Item, error) {
	rows, err := s.sqlc.ItemListByDateRange(ctx, sqlc.ItemListByDateRangeParams{
		From: &from,
		To:   &to,
	})
	if err != nil {
		return nil, err
	}
	out := make([]news.Item, 0, len(rows))
	for _, r := range rows {
		out = append(out, transformItem(r))
	}
	return out, nil
}

func (s Store) Create(ctx context.Context, issueID *int64, position int, item news.Item) (news.Item, error) {
	name, username, avatar, profile := authorFields(item.Author)

	var nid sql.NullInt64
	if issueID != nil {
		nid = sql.NullInt64{Int64: *issueID, Valid: true}
	}

	var published *time.Time
	if !item.Published.IsZero() {
		t := item.Published
		published = &t
	}

	created, err := s.sqlc.ItemCreate(ctx, sqlc.ItemCreateParams{
		IssueID:          nid,
		Source:           item.Source.String(),
		Tag:              string(item.Tag),
		Title:            item.Title,
		Url:              item.URL,
		OriginalUrl:      dbtypes.NullString(item.OriginalURL),
		AuthorName:       name,
		AuthorUsername:   username,
		AuthorAvatarUrl:  avatar,
		AuthorProfileUrl: profile,
		Score:            sql.NullFloat64{Float64: item.Score, Valid: true},
		Summary:          dbtypes.NullString(item.Snippet),
		Position:         int64(position),
		Published:        published,
	})
	if err != nil {
		return news.Item{}, err
	}

	return transformItem(created), nil
}

func (s Store) DeleteByIssue(ctx context.Context, issueID int64) error {
	return s.sqlc.ItemDeleteByIssue(ctx, sql.NullInt64{Int64: issueID, Valid: true})
}

func transformItem(i sqlc.Item) news.Item {
	out := news.Item{
		ID:          i.ID,
		Source:      news.Source(i.Source),
		Tag:         news.Tag(i.Tag),
		Title:       i.Title,
		URL:         i.Url,
		OriginalURL: i.OriginalUrl.String,
		Snippet:     i.Summary.String,
		Score:       i.Score.Float64,
	}
	if i.Published != nil {
		out.Published = *i.Published
	}
	if a := authorFromRow(i); a != nil {
		out.Author = a
	}
	return out
}

func authorFromRow(i sqlc.Item) *news.Author {
	if !i.AuthorName.Valid && !i.AuthorUsername.Valid && !i.AuthorAvatarUrl.Valid && !i.AuthorProfileUrl.Valid {
		return nil
	}
	return &news.Author{
		Name:       i.AuthorName.String,
		Username:   i.AuthorUsername.String,
		AvatarURL:  i.AuthorAvatarUrl.String,
		ProfileURL: i.AuthorProfileUrl.String,
	}
}

func authorFields(a *news.Author) (name, username, avatar, profile sql.NullString) {
	if a == nil {
		return
	}
	return dbtypes.NullString(a.Name), dbtypes.NullString(a.Username), dbtypes.NullString(a.AvatarURL), dbtypes.NullString(a.ProfileURL)
}
