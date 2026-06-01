// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtest"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
)

func TestIssues_Store(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()
	s := issues.New(db)

	mock := digest.Issue{
		Slug:    "2026-04-28",
		Subject: "GoDaily - April 28, 2026",
		Status:  digest.IssueStatusSent,
		Summary: "a summary",
		SentAt:  time.Date(2026, time.April, 28, 8, 0, 0, 0, time.UTC),
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
		t.Log("No filter returns all issues")
		{
			got, err := s.List(ctx, digest.IssueListOptions{})
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, mock.Slug, got[0].Slug)
			assert.Equal(t, mock.Subject, got[0].Subject)
		}

		t.Log("Filters by status when set")
		{
			sent := digest.IssueStatusSent
			got, err := s.List(ctx, digest.IssueListOptions{Status: &sent})
			require.NoError(t, err)
			require.Len(t, got, 1)
		}

		t.Log("Filters out non-matching status")
		{
			draft := digest.IssueStatusDraft
			got, err := s.List(ctx, digest.IssueListOptions{Status: &draft})
			require.NoError(t, err)
			assert.Empty(t, got)
		}
	})

	t.Run("Latest", func(t *testing.T) {
		t.Log("Returns most recent sent issues")
		{
			got, err := s.Latest(ctx, 5)
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, mock.Slug, got[0].Slug)
			assert.Equal(t, mock.Subject, got[0].Subject)
		}

		t.Log("Zero or negative limit returns nil")
		{
			got, err := s.Latest(ctx, 0)
			require.NoError(t, err)
			assert.Nil(t, got)
		}
	})

	t.Run("Count", func(t *testing.T) {
		t.Log("No filter counts all issues")
		{
			got, err := s.Count(ctx, digest.IssueListOptions{})
			require.NoError(t, err)
			assert.Equal(t, int64(1), got)
		}

		t.Log("Filters by status when set")
		{
			sent := digest.IssueStatusSent
			got, err := s.Count(ctx, digest.IssueListOptions{Status: &sent})
			require.NoError(t, err)
			assert.Equal(t, int64(1), got)
		}
	})

	t.Run("Update", func(t *testing.T) {
		draft, err := s.Create(ctx, digest.Issue{
			Slug:    "2026-05-01",
			Subject: "Original",
			Summary: "Original summary",
			Status:  digest.IssueStatusDraft,
			SentAt:  mock.SentAt,
		})
		require.NoError(t, err)

		t.Log("Happy path updates subject and summary on a draft")
		{
			got, err := s.Update(ctx, digest.Issue{
				ID:      draft.ID,
				Subject: "New subject",
				Summary: "New summary",
			})
			require.NoError(t, err)
			assert.Equal(t, draft.ID, got.ID)
			assert.Equal(t, "New subject", got.Subject)
			assert.Equal(t, "New summary", got.Summary)
			assert.Equal(t, digest.IssueStatusDraft, got.Status)
			assert.NotNil(t, got.Items)
		}

		t.Log("Rejects update on a non-draft issue")
		{
			_, err := s.Update(ctx, digest.Issue{
				ID:      mock.ID,
				Subject: "Should not apply",
				Summary: "",
			})
			require.Error(t, err)
			assert.ErrorIs(t, err, digest.ErrIssueNotDraft)
		}

		t.Log("Returns ErrNotFound for unknown ID")
		{
			_, err := s.Update(ctx, digest.Issue{ID: 9999, Subject: "x"})
			require.Error(t, err)
			assert.ErrorIs(t, err, store.ErrNotFound)
		}
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		t.Log("Happy path")
		{
			got, err := s.UpdateStatus(ctx, mock.ID, digest.IssueStatusError, mock.SentAt)
			require.NoError(t, err)
			assert.Equal(t, digest.IssueStatusError, got.Status)
			assert.Equal(t, mock.ID, got.ID)
		}

		t.Log("Unknown ID returns sql.ErrNoRows via RETURNING *")
		{
			_, err := s.UpdateStatus(ctx, 999, digest.IssueStatusSent, mock.SentAt)
			assert.Error(t, err)
		}
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
			_, err := s.Create(ctx, digest.Issue{Slug: "x", Subject: "x", Status: digest.IssueStatusSent})
			assert.Error(t, err)
		}

		t.Log("UpdateStatus")
		{
			_, err := s.UpdateStatus(ctx, 1, digest.IssueStatusSent, mock.SentAt)
			assert.Error(t, err)
		}

		t.Log("Update")
		{
			_, err := s.Update(ctx, digest.Issue{ID: 1, Subject: "x"})
			assert.Error(t, err)
		}

		t.Log("Count")
		{
			_, err := s.Count(ctx, digest.IssueListOptions{})
			assert.Error(t, err)
		}
	})
}
