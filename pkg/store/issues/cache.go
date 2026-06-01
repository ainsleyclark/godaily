// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleydev/webkit/pkg/cache"
)

// CachingStore wraps an IssueRepository and caches Find and FindBySlug results.
type CachingStore struct {
	repo  digest.IssueRepository
	cache cache.Store
}

var _ digest.IssueRepository = (*CachingStore)(nil)

// NewCaching returns an IssueRepository that transparently caches reads.
func NewCaching(repo digest.IssueRepository, c cache.Store) *CachingStore {
	return &CachingStore{repo: repo, cache: c}
}

func (s *CachingStore) Find(ctx context.Context, id int64) (digest.Issue, error) {
	return s.cachedLookup(ctx, fmt.Sprintf("issue:id:%d", id), func() (digest.Issue, error) {
		return s.repo.Find(ctx, id)
	})
}

func (s *CachingStore) FindBySlug(ctx context.Context, slug string) (digest.Issue, error) {
	return s.cachedLookup(ctx, fmt.Sprintf("issue:slug:%s", slug), func() (digest.Issue, error) {
		return s.repo.FindBySlug(ctx, slug)
	})
}

func (s *CachingStore) List(ctx context.Context, opts digest.IssueListOptions) ([]digest.Issue, error) {
	return s.repo.List(ctx, opts)
}

func (s *CachingStore) Latest(ctx context.Context, limit int) ([]digest.Issue, error) {
	return s.repo.Latest(ctx, limit)
}

func (s *CachingStore) Create(ctx context.Context, issue digest.Issue) (digest.Issue, error) {
	return s.repo.Create(ctx, issue)
}

func (s *CachingStore) Delete(ctx context.Context, id int64) (digest.Issue, error) {
	issue, err := s.repo.Delete(ctx, id)
	if err != nil {
		return digest.Issue{}, err
	}
	_ = s.cache.Delete(ctx, fmt.Sprintf("issue:id:%d", id))
	_ = s.cache.Delete(ctx, fmt.Sprintf("issue:slug:%s", issue.Slug))
	return issue, nil
}

func (s *CachingStore) Update(ctx context.Context, issue digest.Issue) (digest.Issue, error) {
	updated, err := s.repo.Update(ctx, issue)
	if err != nil {
		return digest.Issue{}, err
	}
	_ = s.cache.Delete(ctx, fmt.Sprintf("issue:id:%d", updated.ID))
	_ = s.cache.Delete(ctx, fmt.Sprintf("issue:slug:%s", updated.Slug))
	return updated, nil
}

func (s *CachingStore) UpdateStatus(ctx context.Context, id int64, status digest.IssueStatus, sentAt time.Time) (digest.Issue, error) {
	issue, err := s.repo.UpdateStatus(ctx, id, status, sentAt)
	if err != nil {
		return digest.Issue{}, err
	}
	// Invalidate stale cache entries for this issue.
	_ = s.cache.Delete(ctx, fmt.Sprintf("issue:id:%d", id))
	_ = s.cache.Delete(ctx, fmt.Sprintf("issue:slug:%s", issue.Slug))
	return issue, nil
}

func (s *CachingStore) Count(ctx context.Context, opts digest.IssueListOptions) (int64, error) {
	return s.repo.Count(ctx, opts)
}

// cachedLookup gets an issue from cache (JSON bytes) or falls through to fetch.
func (s *CachingStore) cachedLookup(ctx context.Context, key string, fetch func() (digest.Issue, error)) (digest.Issue, error) {
	var raw []byte
	if err := s.cache.Get(ctx, key, &raw); err == nil {
		var issue digest.Issue
		if err = json.Unmarshal(raw, &issue); err == nil {
			return issue, nil
		}
	}
	issue, err := fetch()
	if err != nil {
		return digest.Issue{}, err
	}
	if data, err := json.Marshal(issue); err == nil {
		s.cache.Set(ctx, key, data, cache.Options{Expiration: cache.Forever})
	}
	return issue, nil
}
