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

package digest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/db"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/news"
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
	req    email.SendEmailRequest
	err    error
}

func (m *mockEmail) Send(_ context.Context, req email.SendEmailRequest) error {
	m.called = true
	m.req = req
	return m.err
}

type mockSlack struct {
	msgs []string
}

func (m *mockSlack) MustSend(_ context.Context, message string) {
	m.msgs = append(m.msgs, message)
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
func newSubsMock(t *testing.T) *mocknews.MockSubscriberRepository {
	t.Helper()
	m := mocknews.NewMockSubscriberRepository(gomock.NewController(t))
	m.EXPECT().ListActive(gomock.Any()).Return(nil, nil).AnyTimes()
	return m
}

// errItemRepo is an ItemRepository that always returns errItemRepoErr from ListByIssue.
type errItemRepo struct {
	err error
}

func (e errItemRepo) Find(_ context.Context, _ int64) (news.Item, error) {
	return news.Item{}, nil
}

func (e errItemRepo) ListByIssue(_ context.Context, _ int64) ([]news.Item, error) {
	return nil, e.err
}

func (e errItemRepo) Create(_ context.Context, _ int64, _ int, _ news.Item) (news.Item, error) {
	return news.Item{}, nil
}
func (e errItemRepo) DeleteByIssue(_ context.Context, _ int64) error { return nil }

func newTestStores(t *testing.T) (*issues.Store, *items.Store) {
	t.Helper()
	url := "file:" + filepath.Join(t.TempDir(), "godaily.db")
	conn, err := db.New(t.Context(), url, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, db.Up(t.Context(), conn))
	return issues.New(conn), items.New(conn)
}
