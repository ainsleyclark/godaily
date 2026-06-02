// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package items

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleydev/webkit/pkg/cache"
)

// countsTTL bounds how stale the sidebar source/tag aggregates may be. They
// are full-table GROUP BY scans that change slowly (only when collection runs),
// so a short TTL trades negligible staleness for skipping the scan on the vast
// majority of browse hits.
const countsTTL = 5 * time.Minute

const (
	cacheKeySourceCounts = "items:source_counts"
	cacheKeyTagCounts    = "items:tag_counts"
)

// CachingStore wraps an ItemRepository and caches the slow, slowly-changing
// SourceCounts and TagCounts aggregates with a short TTL. Every other method
// passes straight through.
type CachingStore struct {
	repo  news.ItemRepository
	cache cache.Store
}

var _ news.ItemRepository = (*CachingStore)(nil)

// NewCaching returns an ItemRepository that transparently caches the browse
// sidebar aggregates.
func NewCaching(repo news.ItemRepository, c cache.Store) *CachingStore {
	return &CachingStore{repo: repo, cache: c}
}

func (s *CachingStore) Find(ctx context.Context, id int64) (news.Item, error) {
	return s.repo.Find(ctx, id)
}

func (s *CachingStore) List(ctx context.Context, opts news.ItemListOptions) ([]news.Item, error) {
	return s.repo.List(ctx, opts)
}

func (s *CachingStore) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}

func (s *CachingStore) CountMatching(ctx context.Context, opts news.ItemListOptions) (int64, error) {
	return s.repo.CountMatching(ctx, opts)
}

// SourceCounts returns the cached source aggregates, falling through to the
// underlying store on a miss.
func (s *CachingStore) SourceCounts(ctx context.Context) ([]news.SourceCount, error) {
	var cached []news.SourceCount
	if s.getCached(ctx, cacheKeySourceCounts, &cached) {
		return cached, nil
	}
	out, err := s.repo.SourceCounts(ctx)
	if err != nil {
		return nil, err
	}
	s.setCached(ctx, cacheKeySourceCounts, out)
	return out, nil
}

// TagCounts returns the cached tag aggregates, falling through to the
// underlying store on a miss.
func (s *CachingStore) TagCounts(ctx context.Context) ([]news.TagCount, error) {
	var cached []news.TagCount
	if s.getCached(ctx, cacheKeyTagCounts, &cached) {
		return cached, nil
	}
	out, err := s.repo.TagCounts(ctx)
	if err != nil {
		return nil, err
	}
	s.setCached(ctx, cacheKeyTagCounts, out)
	return out, nil
}

func (s *CachingStore) Create(ctx context.Context, issueID *int64, position int, item news.Item) (news.Item, error) {
	return s.repo.Create(ctx, issueID, position, item)
}

func (s *CachingStore) DeleteByIssue(ctx context.Context, issueID int64) error {
	return s.repo.DeleteByIssue(ctx, issueID)
}

func (s *CachingStore) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *CachingStore) LinkToIssue(ctx context.Context, issueID, itemID int64) error {
	return s.repo.LinkToIssue(ctx, issueID, itemID)
}

func (s *CachingStore) UnlinkFromIssue(ctx context.Context, issueID, itemID int64) error {
	return s.repo.UnlinkFromIssue(ctx, issueID, itemID)
}

func (s *CachingStore) ReorderInIssue(ctx context.Context, issueID int64, orderedItemIDs []int64) error {
	return s.repo.ReorderInIssue(ctx, issueID, orderedItemIDs)
}

// getCached unmarshals the JSON value at key into dst, returning true on a hit.
func (s *CachingStore) getCached(ctx context.Context, key string, dst any) bool {
	var raw []byte
	if err := s.cache.Get(ctx, key, &raw); err != nil {
		return false
	}
	return json.Unmarshal(raw, dst) == nil
}

// setCached stores v as JSON under key with the aggregates TTL.
func (s *CachingStore) setCached(ctx context.Context, key string, v any) {
	if data, err := json.Marshal(v); err == nil {
		s.cache.Set(ctx, key, data, cache.Options{Expiration: countsTTL})
	}
}
