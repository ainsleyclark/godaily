// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/db"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/mocks/audience"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/items"
)

type mockFetcher struct {
	items []news.Item
	err   error
}

func (m mockFetcher) Fetch(_ context.Context) ([]news.Item, error) {
	return m.items, m.err
}

type mockEmail struct {
	called bool
	req    email.SendEmailRequest   // the most recent request
	reqs   []email.SendEmailRequest // every request, in send order
	err    error
}

func (m *mockEmail) Send(_ context.Context, req email.SendEmailRequest) error {
	m.called = true
	m.req = req
	m.reqs = append(m.reqs, req)
	return m.err
}

func (m *mockEmail) SendBatch(_ context.Context, reqs []*email.SendEmailRequest) error {
	for _, r := range reqs {
		m.called = true
		m.req = *r
		m.reqs = append(m.reqs, *r)
	}
	return m.err
}

type mockSlack struct {
	msgs []string
}

func (m *mockSlack) Send(_ context.Context, _ slack.Request) error {
	return nil
}

func (m *mockSlack) MustSend(_ context.Context, req slack.Request) {
	m.msgs = append(m.msgs, req.Text)
}

// allRegistered returns a registry populated with mock fetchers for
// every source in news.Sources.
func allRegistered() map[news.Source]news.Fetcher {
	reg := map[news.Source]news.Fetcher{}
	for _, s := range news.Sources {
		reg[s] = mockFetcher{}
	}
	return reg
}

// newSubsMock returns a MockSubscriberRepository whose ListActive returns an empty list.
// AnyTimes allows tests that return early before ListActive is called to pass without failure.
func newSubsMock(t *testing.T) *mockaudience.MockSubscriberRepository {
	t.Helper()
	m := mockaudience.NewMockSubscriberRepository(gomock.NewController(t))
	m.EXPECT().ListActive(gomock.Any()).Return(nil, nil).AnyTimes()
	return m
}

// errItemRepo is an ItemRepository that always returns errItemRepoErr from List.
type errItemRepo struct {
	err error
}

func (e errItemRepo) Find(_ context.Context, _ int64) (news.Item, error) {
	return news.Item{}, nil
}

func (e errItemRepo) List(_ context.Context, _ news.ItemListOptions) ([]news.Item, error) {
	return nil, e.err
}

func (e errItemRepo) Create(_ context.Context, _ *int64, _ int, _ news.Item) (news.Item, error) {
	return news.Item{}, nil
}
func (e errItemRepo) DeleteByIssue(_ context.Context, _ int64) error { return nil }
func (e errItemRepo) Delete(_ context.Context, _ int64) error        { return nil }
func (e errItemRepo) Count(_ context.Context) (int64, error)         { return 0, e.err }
func (e errItemRepo) CountMatching(_ context.Context, _ news.ItemListOptions) (int64, error) {
	return 0, e.err
}

func (e errItemRepo) SourceCounts(_ context.Context) ([]news.SourceCount, error) {
	return nil, e.err
}
func (e errItemRepo) TagCounts(_ context.Context) ([]news.TagCount, error) { return nil, e.err }
func (e errItemRepo) UnlinkFromIssue(_ context.Context, _, _ int64) error  { return e.err }
func (e errItemRepo) ReorderInIssue(_ context.Context, _ int64, _ []int64) error {
	return e.err
}

func newTestStores(t *testing.T) (*issues.Store, *items.Store) {
	t.Helper()
	url := "file:" + filepath.Join(t.TempDir(), "godaily.db")
	conn, err := db.New(t.Context(), url, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, db.Up(t.Context(), conn))
	return issues.New(conn), items.New(conn)
}
