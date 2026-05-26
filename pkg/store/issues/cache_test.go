// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleydev/webkit/pkg/cache"
	"github.com/ainsleydev/webkit/pkg/cache/cachefakes"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	errRepo   = errors.New("repo error")
	testSlug  = "go-weekly-1"
	testID    = int64(1)
	testIssue = digest.Issue{
		ID:      testID,
		Slug:    testSlug,
		Subject: "Go Weekly #1",
	}
)

func newCachingStore(t *testing.T) (*CachingStore, *mockdigest.MockIssueRepository, *cachefakes.MockStore) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockRepo := mockdigest.NewMockIssueRepository(ctrl)
	mockCache := cachefakes.NewMockStore(ctrl)
	return NewCaching(mockRepo, mockCache), mockRepo, mockCache
}

func TestCachingStore_Find(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		mock      func(repo *mockdigest.MockIssueRepository, c *cachefakes.MockStore)
		wantIssue digest.Issue
		wantErr   bool
	}{
		"Cache hit": {
			mock: func(repo *mockdigest.MockIssueRepository, c *cachefakes.MockStore) {
				encoded, _ := json.Marshal(testIssue)
				c.EXPECT().
					Get(gomock.Any(), "issue:id:1", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, out any) error {
						*out.(*[]byte) = encoded
						return nil
					})
			},
			wantIssue: testIssue,
		},
		"Cache miss - repo ok": {
			mock: func(repo *mockdigest.MockIssueRepository, c *cachefakes.MockStore) {
				encoded, _ := json.Marshal(testIssue)
				c.EXPECT().
					Get(gomock.Any(), "issue:id:1", gomock.Any()).
					Return(cache.ErrNotFound)
				repo.EXPECT().
					Find(gomock.Any(), testID).
					Return(testIssue, nil)
				c.EXPECT().
					Set(gomock.Any(), "issue:id:1", encoded, cache.Options{Expiration: cache.Forever})
			},
			wantIssue: testIssue,
		},
		"Cache miss - repo error": {
			mock: func(repo *mockdigest.MockIssueRepository, c *cachefakes.MockStore) {
				c.EXPECT().
					Get(gomock.Any(), "issue:id:1", gomock.Any()).
					Return(cache.ErrNotFound)
				repo.EXPECT().
					Find(gomock.Any(), testID).
					Return(digest.Issue{}, errRepo)
			},
			wantErr: true,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s, mockRepo, mockCache := newCachingStore(t)
			test.mock(mockRepo, mockCache)
			got, err := s.Find(context.Background(), testID)
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.wantIssue, got)
		})
	}
}

func TestCachingStore_FindBySlug(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		mock      func(repo *mockdigest.MockIssueRepository, c *cachefakes.MockStore)
		wantIssue digest.Issue
		wantErr   bool
	}{
		"Cache hit": {
			mock: func(repo *mockdigest.MockIssueRepository, c *cachefakes.MockStore) {
				encoded, _ := json.Marshal(testIssue)
				c.EXPECT().
					Get(gomock.Any(), "issue:slug:"+testSlug, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, out any) error {
						*out.(*[]byte) = encoded
						return nil
					})
			},
			wantIssue: testIssue,
		},
		"Cache miss - repo ok": {
			mock: func(repo *mockdigest.MockIssueRepository, c *cachefakes.MockStore) {
				encoded, _ := json.Marshal(testIssue)
				c.EXPECT().
					Get(gomock.Any(), "issue:slug:"+testSlug, gomock.Any()).
					Return(cache.ErrNotFound)
				repo.EXPECT().
					FindBySlug(gomock.Any(), testSlug).
					Return(testIssue, nil)
				c.EXPECT().
					Set(gomock.Any(), "issue:slug:"+testSlug, encoded, cache.Options{Expiration: cache.Forever})
			},
			wantIssue: testIssue,
		},
		"Cache miss - repo error": {
			mock: func(repo *mockdigest.MockIssueRepository, c *cachefakes.MockStore) {
				c.EXPECT().
					Get(gomock.Any(), "issue:slug:"+testSlug, gomock.Any()).
					Return(cache.ErrNotFound)
				repo.EXPECT().
					FindBySlug(gomock.Any(), testSlug).
					Return(digest.Issue{}, errRepo)
			},
			wantErr: true,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s, mockRepo, mockCache := newCachingStore(t)
			test.mock(mockRepo, mockCache)
			got, err := s.FindBySlug(context.Background(), testSlug)
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.wantIssue, got)
		})
	}
}

func TestCachingStore_UpdateStatus(t *testing.T) {
	t.Parallel()

	sentAt := time.Now()

	tt := map[string]struct {
		mock      func(repo *mockdigest.MockIssueRepository, c *cachefakes.MockStore)
		wantIssue digest.Issue
		wantErr   bool
	}{
		"OK - invalidates both cache keys": {
			mock: func(repo *mockdigest.MockIssueRepository, c *cachefakes.MockStore) {
				repo.EXPECT().
					UpdateStatus(gomock.Any(), testID, digest.IssueStatusSent, sentAt).
					Return(testIssue, nil)
				c.EXPECT().Delete(gomock.Any(), "issue:id:1").Return(nil)
				c.EXPECT().Delete(gomock.Any(), "issue:slug:"+testSlug).Return(nil)
			},
			wantIssue: testIssue,
		},
		"Repo error - no cache invalidation": {
			mock: func(repo *mockdigest.MockIssueRepository, c *cachefakes.MockStore) {
				repo.EXPECT().
					UpdateStatus(gomock.Any(), testID, digest.IssueStatusSent, sentAt).
					Return(digest.Issue{}, errRepo)
			},
			wantErr: true,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s, mockRepo, mockCache := newCachingStore(t)
			test.mock(mockRepo, mockCache)
			got, err := s.UpdateStatus(context.Background(), testID, digest.IssueStatusSent, sentAt)
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.wantIssue, got)
		})
	}
}
