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
	"encoding/json"
	"fmt"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
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

func (s *CachingStore) List(ctx context.Context, opts store.ListOptions) ([]digest.Issue, error) {
	return s.repo.List(ctx, opts)
}

func (s *CachingStore) ListByStatus(ctx context.Context, status digest.IssueStatus, opts store.ListOptions) ([]digest.Issue, error) {
	return s.repo.ListByStatus(ctx, status, opts)
}

func (s *CachingStore) CountByStatus(ctx context.Context, status digest.IssueStatus) (int64, error) {
	return s.repo.CountByStatus(ctx, status)
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

func (s *CachingStore) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
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
