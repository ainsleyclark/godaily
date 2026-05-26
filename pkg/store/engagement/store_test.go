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

package engagement_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store/emailevents"
	metricsstore "github.com/ainsleyclark/godaily/pkg/store/engagement"
	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtest"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/items"
	"github.com/ainsleyclark/godaily/pkg/store/subscribers"
)

// insertSubscriberAt inserts a subscriber with a controlled created_at, bypassing
// the store's token-generation logic to allow precise timestamp control for growth
// window tests.
func insertSubscriberAt(t *testing.T, ctx context.Context, db *sql.DB, email string, createdAt time.Time, confirmedAt, unsubscribedAt, bouncedAt, suppressedAt *time.Time) {
	t.Helper()

	nullTime := func(tp *time.Time) any {
		if tp == nil {
			return nil
		}
		return tp.Format(time.RFC3339)
	}

	_, err := db.ExecContext(
		ctx,
		`INSERT INTO subscribers
		    (email, unsubscribe_token, created_at, confirmed_at, unsubscribed_at, bounced_at, suppressed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		email, "tok-"+email,
		createdAt.Format(time.RFC3339),
		nullTime(confirmedAt),
		nullTime(unsubscribedAt),
		nullTime(bouncedAt),
		nullTime(suppressedAt),
	)
	require.NoError(t, err)
}

func ptr(t time.Time) *time.Time { return &t }

func TestMetrics_Store(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	s := metricsstore.New(db)

	now := time.Now().UTC()
	from := now.Add(-7 * 24 * time.Hour)
	filter := engagement.MetricsFilter{From: &from, To: &now}

	// Issue sent 2 days ago (within the window).
	issue, err := issues.New(db).Create(ctx, digest.Issue{
		Slug:    "2026-05-22",
		Subject: "GoDaily - May 22, 2026",
		Status:  digest.IssueStatusSent,
		SentAt:  now.Add(-2 * 24 * time.Hour),
	})
	require.NoError(t, err)

	subs := subscribers.New(db)
	subA, err := subs.Create(ctx, "a@example.com")
	require.NoError(t, err)
	subB, err := subs.Create(ctx, "b@example.com")
	require.NoError(t, err)

	item, err := items.New(db).Create(ctx, &issue.ID, 1, news.Item{
		Source: "The Go Blog",
		Tag:    "language",
		Title:  "Go 1.26 released",
		URL:    "https://go.dev/blog/go1.26",
	})
	require.NoError(t, err)

	item2, err := items.New(db).Create(ctx, &issue.ID, 2, news.Item{
		Source: "Ardan Labs",
		Tag:    "tutorial",
		Title:  "Go concurrency patterns",
		URL:    "https://ardanlabs.com/concurrency",
	})
	require.NoError(t, err)

	ee := emailevents.New(db)
	for _, ev := range []engagement.EmailEvent{
		{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeDelivered, EventID: "del-a"},
		{IssueID: &issue.ID, SubscriberID: &subB.ID, Email: "b@example.com", Type: engagement.EmailEventTypeDelivered, EventID: "del-b"},
		{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeOpened, EventID: "open-a1"},
		{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeOpened, EventID: "open-a2"},
		{IssueID: &issue.ID, SubscriberID: &subB.ID, Email: "b@example.com", Type: engagement.EmailEventTypeOpened, EventID: "open-b1"},
		{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeClicked, URL: item.URL, ItemID: &item.ID, EventID: "click-a1"},
		{IssueID: &issue.ID, SubscriberID: &subB.ID, Email: "b@example.com", Type: engagement.EmailEventTypeClicked, URL: item.URL, ItemID: &item.ID, EventID: "click-b1"},
		{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeClicked, URL: item2.URL, ItemID: &item2.ID, EventID: "click-a2"},
	} {
		_, err := ee.Create(ctx, ev)
		require.NoError(t, err)
	}

	t.Run("Summary aggregates headline stats", func(t *testing.T) {
		got, err := s.Summary(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, int64(2), got.Delivered)
		assert.Equal(t, int64(2), got.UniqueOpens, "subA + subB each opened")
		assert.Equal(t, int64(3), got.TotalOpens, "open-a1 + open-a2 + open-b1")
		assert.Equal(t, int64(2), got.UniqueClicks, "subA + subB each clicked")
		assert.Equal(t, int64(3), got.TotalClicks, "click-a1 + click-b1 + click-a2")
		assert.InDelta(t, 1.0, got.OpenRate, 0.001, "both delivered subscribers opened")
		assert.InDelta(t, 1.0, got.ClickRate, 0.001, "both delivered subscribers clicked")
	})

	t.Run("Summary with no filter returns all events", func(t *testing.T) {
		got, err := s.Summary(ctx, engagement.MetricsFilter{})
		require.NoError(t, err)
		assert.Equal(t, int64(2), got.Delivered)
	})

	t.Run("IssueList orders by click_rate", func(t *testing.T) {
		f := filter
		f.Limit = 5
		got, err := s.IssueList(ctx, f, "click_rate")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, issue.ID, got[0].IssueID)
		assert.Equal(t, issue.Slug, got[0].Slug)
		assert.Equal(t, int64(2), got[0].Delivered)
		assert.Equal(t, int64(2), got[0].UniqueClicks)
		assert.InDelta(t, 1.0, got[0].ClickRate, 0.001)
	})

	t.Run("IssueList respects limit", func(t *testing.T) {
		f := filter
		f.Limit = 0
		got, err := s.IssueList(ctx, f, "click_rate")
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("ItemList ranks by clicks", func(t *testing.T) {
		f := filter
		f.Limit = 5
		got, err := s.ItemList(ctx, f)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, item.ID, got[0].ItemID, "item1 has 2 clicks, should rank first")
		assert.Equal(t, int64(2), got[0].Clicks)
		assert.Equal(t, item2.ID, got[1].ItemID)
		assert.Equal(t, int64(1), got[1].Clicks)
	})

	t.Run("ItemList respects limit", func(t *testing.T) {
		f := filter
		f.Limit = 1
		got, err := s.ItemList(ctx, f)
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})

	t.Run("TagList aggregates clicks by tag", func(t *testing.T) {
		f := filter
		f.Limit = 5
		got, err := s.TagList(ctx, f)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "language", got[0].Tag, "language tag has 2 clicks")
		assert.Equal(t, int64(2), got[0].Clicks)
		assert.Equal(t, "tutorial", got[1].Tag)
		assert.Equal(t, int64(1), got[1].Clicks)
	})

	t.Run("SourceList aggregates clicks by source", func(t *testing.T) {
		f := filter
		f.Limit = 5
		got, err := s.SourceList(ctx, f)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "The Go Blog", got[0].Source)
		assert.Equal(t, int64(2), got[0].Clicks)
		assert.Equal(t, "Ardan Labs", got[1].Source)
		assert.Equal(t, int64(1), got[1].Clicks)
	})

	t.Run("SubscriberGrowth day bucket with time filter", func(t *testing.T) {
		inWindow := now.Add(-3 * 24 * time.Hour)
		confirmedAt := ptr(inWindow.Add(time.Hour))
		unsubAt := ptr(inWindow.Add(2 * time.Hour))
		bouncedAt := ptr(inWindow.Add(3 * time.Hour))
		outOfWindow := now.Add(-10 * 24 * time.Hour)

		// alice: created+confirmed in-window, then unsubscribed.
		insertSubscriberAt(t, ctx, db, "alice@example.com", inWindow, confirmedAt, unsubAt, nil, nil)
		// bounce: created+confirmed in-window, then bounced.
		insertSubscriberAt(t, ctx, db, "bounce@example.com", inWindow, confirmedAt, nil, bouncedAt, nil)
		// old: created before the window — must not appear in in-window counts.
		insertSubscriberAt(t, ctx, db, "old@example.com", outOfWindow, nil, nil, nil, nil)

		got, err := s.SubscriberGrowth(ctx, filter, "day")
		require.NoError(t, err)
		assert.Equal(t, "day", got.Bucket)
		assert.NotEmpty(t, got.Points)

		var totalNew, totalConfirmed, totalUnsub, totalLost int64
		for _, p := range got.Points {
			totalNew += p.New
			totalConfirmed += p.Confirmed
			totalUnsub += p.Unsubscribed
			totalLost += p.Lost
		}
		assert.Equal(t, int64(4), totalNew, "subA + subB + alice + bounce all created in-window; old is excluded")
		assert.Equal(t, int64(2), totalConfirmed, "alice + bounce confirmed in-window")
		assert.Equal(t, int64(1), totalUnsub, "alice unsubscribed in-window")
		assert.Equal(t, int64(1), totalLost, "bounce lost in-window")
	})

	t.Run("SubscriberGrowth week bucket", func(t *testing.T) {
		got, err := s.SubscriberGrowth(ctx, filter, "week")
		require.NoError(t, err)
		assert.Equal(t, "week", got.Bucket)
		assert.NotEmpty(t, got.Points)
	})

	t.Run("SubscriberGrowth no filter returns all subscribers", func(t *testing.T) {
		got, err := s.SubscriberGrowth(ctx, engagement.MetricsFilter{}, "day")
		require.NoError(t, err)

		var totalNew int64
		for _, p := range got.Points {
			totalNew += p.New
		}
		// subA + subB (created via store above) + alice + bounce + old = 5
		assert.Equal(t, int64(5), totalNew)
	})

	// MUST be last: closing the DB makes every subsequent query fail.
	t.Run("Query error on closed DB", func(t *testing.T) {
		require.NoError(t, db.Close())

		t.Log("Summary")
		{
			_, err := s.Summary(ctx, filter)
			assert.Error(t, err)
		}

		t.Log("IssueList")
		{
			_, err := s.IssueList(ctx, filter, "click_rate")
			assert.Error(t, err)
		}

		t.Log("ItemList")
		{
			_, err := s.ItemList(ctx, filter)
			assert.Error(t, err)
		}

		t.Log("TagList")
		{
			_, err := s.TagList(ctx, filter)
			assert.Error(t, err)
		}

		t.Log("SourceList")
		{
			_, err := s.SourceList(ctx, filter)
			assert.Error(t, err)
		}

		t.Log("SubscriberGrowth")
		{
			_, err := s.SubscriberGrowth(ctx, filter, "day")
			assert.Error(t, err)
		}
	})
}
