// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package items

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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

// List runs a filtered/sorted/paginated query over items. Every field on
// ItemListOptions is optional; a zero ItemListOptions returns every row
// ordered by published DESC.
//
// Default ordering depends on which filters are set when opts.Sort is empty:
//   - IssueID set → position ASC (preserves digest item order)
//   - From/To set → score DESC
//   - otherwise   → published DESC
func (s Store) List(ctx context.Context, opts news.ItemListOptions) ([]news.Item, error) {
	var (
		clauses []string
		args    []any
	)

	if opts.IssueID != nil {
		clauses = append(clauses, "issue_id = ?")
		args = append(args, *opts.IssueID)
	}

	if opts.From != nil {
		clauses = append(clauses, "published >= ?")
		args = append(args, *opts.From)
	}
	if opts.To != nil {
		clauses = append(clauses, "published < ?")
		args = append(args, *opts.To)
	}

	if len(opts.Sources) > 0 {
		placeholders := make([]string, len(opts.Sources))
		for i, src := range opts.Sources {
			placeholders[i] = "?"
			args = append(args, string(src))
		}
		clauses = append(clauses, "source IN ("+strings.Join(placeholders, ", ")+")")
	}

	if len(opts.Tags) > 0 {
		placeholders := make([]string, len(opts.Tags))
		for i, tag := range opts.Tags {
			placeholders[i] = "?"
			args = append(args, string(tag))
		}
		clauses = append(clauses, "tag IN ("+strings.Join(placeholders, ", ")+")")
	}

	if term := strings.TrimSpace(opts.Search); term != "" {
		like := "%" + term + "%"
		clauses = append(clauses, "(title LIKE ? OR summary LIKE ?)")
		args = append(args, like, like)
	}

	if opts.InDigest != nil {
		if *opts.InDigest {
			clauses = append(clauses, "issue_id IS NOT NULL")
		} else {
			clauses = append(clauses, "issue_id IS NULL")
		}
	}

	var sb strings.Builder
	sb.WriteString(`SELECT id, issue_id, source, title, url, tag,
		author_name, author_username, author_avatar_url, author_profile_url,
		score, summary, position, original_url, published
		FROM items`)
	if len(clauses) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(clauses, " AND "))
	}
	sb.WriteString(" ORDER BY ")
	sb.WriteString(orderByClause(opts))

	page := store.ListOptions{Page: opts.Page, PerPage: opts.PerPage}
	sb.WriteString(" LIMIT ? OFFSET ?")
	args = append(args, page.Limit(), page.Offset())

	rows, err := s.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	out := make([]news.Item, 0)
	for rows.Next() {
		var i sqlc.Item
		if err := rows.Scan(
			&i.ID, &i.IssueID, &i.Source, &i.Title, &i.Url, &i.Tag,
			&i.AuthorName, &i.AuthorUsername, &i.AuthorAvatarUrl, &i.AuthorProfileUrl,
			&i.Score, &i.Summary, &i.Position, &i.OriginalUrl, &i.Published,
		); err != nil {
			return nil, err
		}
		out = append(out, transformItem(i))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func orderByClause(opts news.ItemListOptions) string {
	switch opts.Sort {
	case news.ItemSortTop:
		return "score DESC, id DESC"
	case news.ItemSortHot:
		return "(score / (julianday('now') - julianday(published) + 2)) DESC, id DESC"
	case news.ItemSortNew:
		return "published DESC, id DESC"
	}
	// No explicit sort — preserve legacy defaults based on which filters are set.
	switch {
	case opts.IssueID != nil:
		return "position ASC"
	case opts.From != nil || opts.To != nil:
		return "score DESC, id DESC"
	default:
		return "published DESC, id DESC"
	}
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

// Count returns the total number of items in the store.
func (s Store) Count(ctx context.Context) (int64, error) {
	return s.sqlc.ItemCount(ctx)
}

// SourceCounts returns the number of items grouped by source, ordered by count DESC.
func (s Store) SourceCounts(ctx context.Context) ([]news.SourceCount, error) {
	rows, err := s.sqlc.ItemSourceCounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]news.SourceCount, 0, len(rows))
	for _, r := range rows {
		out = append(out, news.SourceCount{Source: news.Source(r.Source), Count: r.Count})
	}
	return out, nil
}

// TagCounts returns the number of items grouped by tag, ordered by count DESC.
func (s Store) TagCounts(ctx context.Context) ([]news.TagCount, error) {
	rows, err := s.sqlc.ItemTagCounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]news.TagCount, 0, len(rows))
	for _, r := range rows {
		out = append(out, news.TagCount{Tag: news.Tag(r.Tag), Count: r.Count})
	}
	return out, nil
}

// FindByURLInIssue resolves a clicked URL back to an item within an issue,
// matching against either the canonical or the original URL. It is a
// best-effort lookup: a missing row returns (0, false, nil), not an error.
func (s Store) FindByURLInIssue(ctx context.Context, issueID int64, url string) (int64, bool, error) {
	id, err := s.sqlc.ItemFindByURLInIssue(ctx, sqlc.ItemFindByURLInIssueParams{
		IssueID: sql.NullInt64{Int64: issueID, Valid: true},
		Url:     url,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
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
		InDigest:    i.IssueID.Valid,
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
