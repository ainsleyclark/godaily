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
	"fmt"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleydev/webkit/pkg/cache"
)

// CachingStore wraps an IssueRepository and caches Find and FindBySlug results.
type CachingStore struct {
	repo  news.IssueRepository
	cache cache.Store
}

var _ news.IssueRepository = (*CachingStore)(nil)

// NewCaching returns an IssueRepository that transparently caches reads.
func NewCaching(repo news.IssueRepository, c cache.Store) *CachingStore {
	return &CachingStore{repo: repo, cache: c}
}

func (s *CachingStore) Find(ctx context.Context, id int64) (news.Issue, error) {
	key := fmt.Sprintf("issue:id:%d", id)
	var issue news.Issue
	if err := s.cache.Get(ctx, key, &issue); err == nil {
		return issue, nil
	}
	issue, err := s.repo.Find(ctx, id)
	if err != nil {
		return news.Issue{}, err
	}
	s.cache.Set(ctx, key, issue, cache.Options{Expiration: cache.Forever})
	return issue, nil
}

func (s *CachingStore) FindBySlug(ctx context.Context, slug string) (news.Issue, error) {
	key := fmt.Sprintf("issue:slug:%s", slug)
	var issue news.Issue
	if err := s.cache.Get(ctx, key, &issue); err == nil {
		return issue, nil
	}
	issue, err := s.repo.FindBySlug(ctx, slug)
	if err != nil {
		return news.Issue{}, err
	}
	s.cache.Set(ctx, key, issue, cache.Options{Expiration: cache.Forever})
	return issue, nil
}

func (s *CachingStore) List(ctx context.Context) ([]news.Issue, error) {
	return s.repo.List(ctx)
}

func (s *CachingStore) Create(ctx context.Context, issue news.Issue) (news.Issue, error) {
	return s.repo.Create(ctx, issue)
}

func (s *CachingStore) UpdateStatus(ctx context.Context, id int64, status news.IssueStatus, sentAt time.Time) (news.Issue, error) {
	issue, err := s.repo.UpdateStatus(ctx, id, status, sentAt)
	if err != nil {
		return news.Issue{}, err
	}
	// Invalidate stale cache entries for this issue.
	_ = s.cache.Delete(ctx, fmt.Sprintf("issue:id:%d", id))
	_ = s.cache.Delete(ctx, fmt.Sprintf("issue:slug:%s", issue.Slug))
	return issue, nil
}

func (s *CachingStore) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}
