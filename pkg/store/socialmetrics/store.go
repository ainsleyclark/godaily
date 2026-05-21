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

// Package socialmetrics persists per-post social engagement counts (likes,
// reposts, comments, impressions) for every platform GoDaily posts to.
package socialmetrics

import (
	"context"
	"database/sql"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/store/internal/sqlc"
)

var _ engagement.SocialMetricRepository = (*Store)(nil)

// New creates a new social metrics Store.
func New(db *sql.DB) *Store {
	return &Store{
		sqlc: sqlc.New(db),
	}
}

// Store provides methods for interacting with social_metrics in the database.
type Store struct {
	sqlc *sqlc.Queries
}

// Upsert inserts or replaces the metrics for a (social_post_id, platform) pair.
// FetchedAt defaults to now when zero.
func (s Store) Upsert(ctx context.Context, m engagement.SocialMetric) error {
	fetchedAt := m.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	return s.sqlc.SocialMetricUpsert(ctx, sqlc.SocialMetricUpsertParams{
		SocialPostID: m.SocialPostID,
		Platform:     m.Platform,
		Likes:        m.Likes,
		Reposts:      m.Reposts,
		Comments:     m.Comments,
		Impressions:  m.Impressions,
		FetchedAt:    fetchedAt,
	})
}

// ListBySocialPostID returns all metric rows for a given social post.
func (s Store) ListBySocialPostID(ctx context.Context, socialPostID int64) ([]engagement.SocialMetric, error) {
	rows, err := s.sqlc.SocialMetricListBySocialPostID(ctx, socialPostID)
	if err != nil {
		return nil, err
	}
	out := make([]engagement.SocialMetric, 0, len(rows))
	for _, r := range rows {
		out = append(out, transform(r))
	}
	return out, nil
}

func transform(r sqlc.SocialMetric) engagement.SocialMetric {
	return engagement.SocialMetric{
		ID:           r.ID,
		SocialPostID: r.SocialPostID,
		Platform:     r.Platform,
		Likes:        r.Likes,
		Reposts:      r.Reposts,
		Comments:     r.Comments,
		Impressions:  r.Impressions,
		FetchedAt:    r.FetchedAt,
	}
}
