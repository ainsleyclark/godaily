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

package emailevents_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/store/emailevents"
	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtest"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/subscribers"
)

func mustCreate(t *testing.T, ctx context.Context, s engagement.EmailEventRepository, e engagement.EmailEvent) {
	t.Helper()
	_, err := s.Create(ctx, e)
	require.NoError(t, err)
}

func TestEmailEvents_Store(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	issue, err := issues.New(db).Create(ctx, digest.Issue{
		Slug:    "2026-05-20",
		Subject: "GoDaily - May 20, 2026",
		Status:  digest.IssueStatusSent,
		SentAt:  time.Date(2026, time.May, 20, 8, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	subs := subscribers.New(db)
	subA, err := subs.Create(ctx, "a@example.com")
	require.NoError(t, err)
	subB, err := subs.Create(ctx, "b@example.com")
	require.NoError(t, err)

	s := emailevents.New(db)

	t.Run("Create persists row and defaults OccurredAt", func(t *testing.T) {
		got, err := s.Create(ctx, engagement.EmailEvent{
			IssueID:      &issue.ID,
			SubscriberID: &subA.ID,
			Email:        "a@example.com",
			Type:         engagement.EmailEventTypeDelivered,
			ProviderID:   "re_abc123",
			EventID:      "evt_delivered_a",
		})
		require.NoError(t, err)
		assert.NotZero(t, got.ID)
		assert.False(t, got.OccurredAt.IsZero(), "OccurredAt should default to now")
		require.NotNil(t, got.IssueID)
		assert.Equal(t, issue.ID, *got.IssueID)
		assert.Equal(t, "re_abc123", got.ProviderID)
	})

	t.Run("Create accepts nil issue and subscriber", func(t *testing.T) {
		got, err := s.Create(ctx, engagement.EmailEvent{
			Email:   "stranger@example.com",
			Type:    engagement.EmailEventTypeBounced,
			EventID: "evt_orphan",
		})
		require.NoError(t, err)
		assert.Nil(t, got.IssueID)
		assert.Nil(t, got.SubscriberID)
	})

	t.Run("Duplicate event ID is rejected", func(t *testing.T) {
		_, err := s.Create(ctx, engagement.EmailEvent{
			Email:   "a@example.com",
			Type:    engagement.EmailEventTypeDelivered,
			EventID: "evt_delivered_a",
		})
		assert.Error(t, err)
	})

	t.Run("ExistsByEventID", func(t *testing.T) {
		t.Log("Known event")
		{
			got, err := s.ExistsByEventID(ctx, "evt_delivered_a")
			require.NoError(t, err)
			assert.True(t, got)
		}

		t.Log("Unknown event")
		{
			got, err := s.ExistsByEventID(ctx, "evt_nope")
			require.NoError(t, err)
			assert.False(t, got)
		}
	})

	t.Run("IssueStats aggregates engagement", func(t *testing.T) {
		mustCreate(t, ctx, s, engagement.EmailEvent{IssueID: &issue.ID, SubscriberID: &subB.ID, Email: "b@example.com", Type: engagement.EmailEventTypeDelivered, EventID: "evt_delivered_b"})
		mustCreate(t, ctx, s, engagement.EmailEvent{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeOpened, EventID: "evt_open_a1"})
		mustCreate(t, ctx, s, engagement.EmailEvent{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeOpened, EventID: "evt_open_a2"})
		mustCreate(t, ctx, s, engagement.EmailEvent{IssueID: &issue.ID, SubscriberID: &subB.ID, Email: "b@example.com", Type: engagement.EmailEventTypeOpened, EventID: "evt_open_b1"})
		mustCreate(t, ctx, s, engagement.EmailEvent{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeClicked, URL: "https://go.dev", EventID: "evt_click_a1"})

		got, err := s.IssueStats(ctx, issue.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), got.Delivered, "delivered: subA + subB")
		assert.Equal(t, int64(2), got.UniqueOpens, "unique opens: subA + subB")
		assert.Equal(t, int64(3), got.TotalOpens, "total opens: a1 + a2 + b1")
		assert.Equal(t, int64(1), got.UniqueClicks)
		assert.Equal(t, int64(1), got.TotalClicks)
		assert.InDelta(t, 1.0, got.OpenRate, 0.0001)
		assert.InDelta(t, 0.5, got.ClickRate, 0.0001)
	})

	t.Run("IssueStats is zero for an unknown issue", func(t *testing.T) {
		got, err := s.IssueStats(ctx, 9999)
		require.NoError(t, err)
		assert.Zero(t, got.Delivered)
		assert.Zero(t, got.OpenRate)
	})

	t.Run("TopLinks ranks clicks", func(t *testing.T) {
		mustCreate(t, ctx, s, engagement.EmailEvent{IssueID: &issue.ID, SubscriberID: &subB.ID, Email: "b@example.com", Type: engagement.EmailEventTypeClicked, URL: "https://go.dev", EventID: "evt_click_b1"})
		mustCreate(t, ctx, s, engagement.EmailEvent{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeClicked, URL: "https://pkg.go.dev", EventID: "evt_click_a2"})

		got, err := s.TopLinks(ctx, issue.ID, 10)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "https://go.dev", got[0].URL)
		assert.Equal(t, int64(2), got[0].Clicks)
		assert.Equal(t, int64(1), got[1].Clicks)
	})

	t.Run("TopLinks respects the limit", func(t *testing.T) {
		got, err := s.TopLinks(ctx, issue.ID, 1)
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})

	// MUST be last: closing the DB makes every subsequent query fail.
	t.Run("Query error on closed DB", func(t *testing.T) {
		require.NoError(t, db.Close())
		_, err := s.Create(ctx, engagement.EmailEvent{Email: "x@example.com", Type: engagement.EmailEventTypeOpened, EventID: "evt_closed"})
		assert.Error(t, err)
	})
}
