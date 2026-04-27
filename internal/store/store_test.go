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

package store_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/internal/db"
	"github.com/ainsleyclark/godaily/internal/store"
)

// setup spins up an isolated, migrated SQLite database under t.TempDir()
// and returns a Store bound to it. The DB is closed when the test ends.
func setup(t *testing.T) *store.Store {
	t.Helper()

	url := "file:" + filepath.Join(t.TempDir(), "godaily.db")
	conn, err := db.New(t.Context(), url, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	require.NoError(t, db.Migrate(t.Context(), conn))
	return store.NewStore(conn)
}

func TestStore_CreateIssueAndNewsItems(t *testing.T) {
	s := setup(t)

	issue, err := s.CreateIssue(t.Context(), store.CreateIssueParams{
		Slug:     "2026-04-26",
		SentAt:   time.Date(2026, time.April, 26, 8, 0, 0, 0, time.UTC),
		Subject:  "GoDaily - April 26, 2026",
		HtmlBody: "<p>hi</p>",
		TextBody: "hi",
		Status:   "sent",
	})
	require.NoError(t, err)
	assert.NotZero(t, issue.ID)

	item, err := s.CreateNewsItem(t.Context(), store.CreateNewsItemParams{
		IssueID:  issue.ID,
		Source:   "hacker_news",
		Title:    "Generics are great",
		Url:      "https://example.com/x",
		Author:   sql.NullString{String: "gopher", Valid: true},
		Score:    sql.NullFloat64{Float64: 0.8, Valid: true},
		Position: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, issue.ID, item.IssueID)

	listed, err := s.ListNewsItemsByIssue(t.Context(), issue.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "Generics are great", listed[0].Title)
}

func TestStore_Subscribe(t *testing.T) {
	t.Run("Generates Tokens And Stores Lowercased Email", func(t *testing.T) {
		s := setup(t)

		sub, err := s.Subscribe(t.Context(), "  Foo@Example.COM ")
		require.NoError(t, err)
		assert.Equal(t, "foo@example.com", sub.Email)
		assert.NotEmpty(t, sub.ConfirmToken)
		assert.NotEmpty(t, sub.UnsubscribeToken)
		assert.NotEqual(t, sub.ConfirmToken, sub.UnsubscribeToken)
		assert.Nil(t, sub.ConfirmedAt)
		assert.Nil(t, sub.UnsubscribedAt)
	})

	t.Run("Empty Email Rejected", func(t *testing.T) {
		s := setup(t)

		_, err := s.Subscribe(t.Context(), "  ")
		assert.Error(t, err)
	})

	t.Run("Duplicate Email Surfaces DB Error", func(t *testing.T) {
		s := setup(t)

		_, err := s.Subscribe(t.Context(), "dupe@example.com")
		require.NoError(t, err)
		_, err = s.Subscribe(t.Context(), "dupe@example.com")
		assert.Error(t, err)
	})

	t.Run("Confirm And Unsubscribe Lifecycle", func(t *testing.T) {
		s := setup(t)

		sub, err := s.Subscribe(t.Context(), "lifecycle@example.com")
		require.NoError(t, err)

		require.NoError(t, s.ConfirmSubscriber(t.Context(), sub.ConfirmToken))

		got, err := s.GetSubscriberByEmail(t.Context(), sub.Email)
		require.NoError(t, err)
		require.NotNil(t, got.ConfirmedAt)

		require.NoError(t, s.UnsubscribeByToken(t.Context(), sub.UnsubscribeToken))

		got, err = s.GetSubscriberByEmail(t.Context(), sub.Email)
		require.NoError(t, err)
		require.NotNil(t, got.UnsubscribedAt)
	})
}

func TestStore_Tx(t *testing.T) {
	t.Run("Commits On Success", func(t *testing.T) {
		s := setup(t)

		err := s.Tx(t.Context(), func(q *store.Queries) error {
			_, err := q.CreateIssue(t.Context(), store.CreateIssueParams{
				Slug:     "2026-01-01",
				SentAt:   time.Now().UTC(),
				Subject:  "ok",
				HtmlBody: "h",
				TextBody: "t",
				Status:   "sent",
			})
			return err
		})
		require.NoError(t, err)

		got, err := s.GetIssueBySlug(t.Context(), "2026-01-01")
		require.NoError(t, err)
		assert.Equal(t, "ok", got.Subject)
	})

	t.Run("Rolls Back On Error", func(t *testing.T) {
		s := setup(t)

		boom := errors.New("boom")
		err := s.Tx(t.Context(), func(q *store.Queries) error {
			_, _ = q.CreateIssue(t.Context(), store.CreateIssueParams{
				Slug:     "2026-01-02",
				SentAt:   time.Now().UTC(),
				Subject:  "rolled back",
				HtmlBody: "h",
				TextBody: "t",
				Status:   "sent",
			})
			return boom
		})
		require.ErrorIs(t, err, boom)

		_, err = s.GetIssueBySlug(t.Context(), "2026-01-02")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}
