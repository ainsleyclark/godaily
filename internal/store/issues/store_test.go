package issues_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/ainsleyclark/godaily/internal/store/internal/dbtest"
	"github.com/ainsleyclark/godaily/internal/store/issues"
)

func TestIssues_Store(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()
	s := issues.New(db)

	mock := news.Issue{
		Slug:     "2026-04-28",
		Subject:  "GoDaily - April 28, 2026",
		Status:   news.IssueStatusSent,
		HtmlBody: "<p>hi</p>",
		TextBody: "hi",
		Summary:  "a summary",
		SentAt:   time.Date(2026, time.April, 28, 8, 0, 0, 0, time.UTC),
	}

	t.Run("Create", func(t *testing.T) {
		t.Log("Happy path")
		{
			got, err := s.Create(ctx, mock)
			require.NoError(t, err)
			assert.NotZero(t, got.ID)
			assert.Equal(t, mock.Slug, got.Slug)
			assert.Equal(t, mock.Subject, got.Subject)
			assert.Equal(t, mock.Status, got.Status)
			assert.Equal(t, mock.Summary, got.Summary)
			mock.ID = got.ID
		}

		t.Log("Rejects duplicate slug")
		{
			_, err := s.Create(ctx, mock)
			assert.Error(t, err)
		}
	})

	t.Run("Find", func(t *testing.T) {
		t.Log("Happy path")
		{
			got, err := s.Find(ctx, mock.ID)
			require.NoError(t, err)
			assert.Equal(t, mock.ID, got.ID)
			assert.Equal(t, mock.Slug, got.Slug)
			assert.Equal(t, mock.Subject, got.Subject)
		}

		t.Log("Not found")
		{
			_, err := s.Find(ctx, 999)
			require.Error(t, err)
			assert.Equal(t, store.ErrNotFound, err)
		}
	})

	t.Run("FindBySlug", func(t *testing.T) {
		t.Log("Happy path")
		{
			got, err := s.FindBySlug(ctx, mock.Slug)
			require.NoError(t, err)
			assert.Equal(t, mock.ID, got.ID)
			assert.Equal(t, mock.Slug, got.Slug)
		}

		t.Log("Not found")
		{
			_, err := s.FindBySlug(ctx, "wrong")
			require.Error(t, err)
			assert.Equal(t, store.ErrNotFound, err)
		}
	})

	t.Run("List", func(t *testing.T) {
		got, err := s.List(ctx)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("Count", func(t *testing.T) {
		got, err := s.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), got)
	})

	// MUST be last: closing the DB makes every subsequent query fail.
	t.Run("Query Error On Closed DB", func(t *testing.T) {
		require.NoError(t, db.Close())

		t.Log("Find")
		{
			_, err := s.Find(ctx, 1)
			assert.Error(t, err)
			assert.NotErrorIs(t, err, store.ErrNotFound)
		}

		t.Log("FindBySlug")
		{
			_, err := s.FindBySlug(ctx, "x")
			assert.Error(t, err)
			assert.NotErrorIs(t, err, store.ErrNotFound)
		}

		t.Log("Create")
		{
			_, err := s.Create(ctx, news.Issue{Slug: "x", Subject: "x", Status: news.IssueStatusSent, HtmlBody: "x", TextBody: "x"})
			assert.Error(t, err)
		}

		t.Log("Count")
		{
			_, err := s.Count(ctx)
			assert.Error(t, err)
		}
	})
}
