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

	t.Log("Create")
	{
		got, err := s.Create(ctx, mock)
		require.NoError(t, err)
		assert.NotZero(t, got.ID)
		assert.Equal(t, mock.Slug, got.Slug)
		assert.Equal(t, mock.Subject, got.Subject)
		assert.Equal(t, mock.Status, got.Status)
		assert.Equal(t, mock.Summary, got.Summary)
		mock.ID = got.ID // For subsequent tests.
	}

	t.Log("Find")
	{
		got, err := s.Find(ctx, mock.ID)
		require.NoError(t, err)
		assert.Equal(t, mock.ID, got.ID)
		assert.Equal(t, mock.Slug, got.Slug)
		assert.Equal(t, mock.Subject, got.Subject)
	}

	t.Log("Find Not Found")
	{
		_, err := s.Find(ctx, 999_999)
		assert.ErrorIs(t, err, store.ErrNotFound)
	}

	t.Log("Find By Slug")
	{
		got, err := s.FindBySlug(ctx, mock.Slug)
		require.NoError(t, err)
		assert.Equal(t, mock.ID, got.ID)
		assert.Equal(t, mock.Slug, got.Slug)
	}

	t.Log("Find By Slug Not Found")
	{
		_, err := s.FindBySlug(ctx, "missing-slug")
		assert.ErrorIs(t, err, store.ErrNotFound)
	}

	t.Log("List")
	{
		got, err := s.List(ctx)
		require.NoError(t, err)
		assert.Nil(t, got)
	}

	t.Log("Count")
	{
		got, err := s.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), got)
	}

	t.Log("Create Duplicate Slug")
	{
		_, err := s.Create(ctx, mock)
		assert.Error(t, err)
	}
}
