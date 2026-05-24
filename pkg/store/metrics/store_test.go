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

package metrics_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store/emailevents"
	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtest"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/items"
	metricsstore "github.com/ainsleyclark/godaily/pkg/store/metrics"
	"github.com/ainsleyclark/godaily/pkg/store/subscribers"
)

// insertSubscriber inserts a subscriber row directly with a controlled created_at.
// It bypasses the store to avoid the token-generation logic, which is not relevant here.
func insertSubscriber(t *testing.T, ctx context.Context, db *sql.DB, email string, createdAt time.Time, confirmedAt, unsubscribedAt, bouncedAt, suppressedAt *time.Time) int64 {
	t.Helper()

	nullTime := func(tp *time.Time) interface{} {
		if tp == nil {
			return nil
		}
		return tp.Format(time.RFC3339)
	}

	res, err := db.ExecContext(ctx,
		`INSERT INTO subscribers
		    (email, unsubscribe_token, created_at, confirmed_at, unsubscribed_at, bounced_at, suppressed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		email,
		"tok-"+email,
		createdAt.Format(time.RFC3339),
		nullTime(confirmedAt),
		nullTime(unsubscribedAt),
		nullTime(bouncedAt),
		nullTime(suppressedAt),
	)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

// TestSubscriberGrowthBuggyQueryFails proves that the pre-fix UNION query,
// which placed "event_time" (a SELECT alias) inside each branch's WHERE clause,
// fails with a "no such column" error. "event_time" is not a real column on
// the subscribers table; branches 2-5 have no such alias at all.
func TestSubscriberGrowthBuggyQueryFails(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	from := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Format(time.RFC3339)

	// This is the query as it existed before the fix.
	buggyQuery := `
SELECT
    strftime('%Y-%m-%d', event_time)                                        AS bucket_start,
    SUM(CASE WHEN event_type = 'new'          THEN 1 ELSE 0 END)           AS new,
    SUM(CASE WHEN event_type = 'confirmed'    THEN 1 ELSE 0 END)           AS confirmed,
    SUM(CASE WHEN event_type = 'unsubscribed' THEN 1 ELSE 0 END)           AS unsubscribed,
    SUM(CASE WHEN event_type = 'lost'         THEN 1 ELSE 0 END)           AS lost
FROM (
    SELECT created_at AS event_time, 'new' AS event_type
      FROM subscribers WHERE 1=1 AND event_time >= ? AND event_time < ?
    UNION ALL
    SELECT confirmed_at, 'confirmed'
      FROM subscribers WHERE confirmed_at IS NOT NULL AND event_time >= ? AND event_time < ?
    UNION ALL
    SELECT unsubscribed_at, 'unsubscribed'
      FROM subscribers WHERE unsubscribed_at IS NOT NULL AND event_time >= ? AND event_time < ?
    UNION ALL
    SELECT bounced_at, 'lost'
      FROM subscribers WHERE bounced_at IS NOT NULL AND event_time >= ? AND event_time < ?
    UNION ALL
    SELECT suppressed_at, 'lost'
      FROM subscribers WHERE suppressed_at IS NOT NULL AND event_time >= ? AND event_time < ?
) events
GROUP BY bucket_start
ORDER BY bucket_start ASC`

	args := []any{from, to, from, to, from, to, from, to, from, to}
	rows, err := db.QueryContext(ctx, buggyQuery, args...)
	if err == nil {
		// If QueryContext doesn't fail immediately, the error surfaces on the
		// first rows.Next() call (deferred prepare/execute cycle in the driver).
		for rows.Next() {
		}
		err = rows.Err()
		rows.Close()
	}

	require.Error(t, err, "buggy query must fail: event_time is not a column in subscribers")
	assert.True(t,
		strings.Contains(err.Error(), "event_time") || strings.Contains(err.Error(), "no such column"),
		"error should mention the unknown column, got: %v", err,
	)
}

func TestStore_SubscriberGrowth(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	s := metricsstore.New(db)

	now := time.Now().UTC()
	window := 7 * 24 * time.Hour
	from := now.Add(-window)

	inWindow := now.Add(-3 * 24 * time.Hour)
	outOfWindow := now.Add(-10 * 24 * time.Hour)
	confirmedAt := ptr(inWindow.Add(time.Hour))
	unsubAt := ptr(inWindow.Add(2 * time.Hour))

	// Subscriber created and confirmed within window, later unsubscribed.
	insertSubscriber(t, ctx, db, "alice@example.com", inWindow, confirmedAt, unsubAt, nil, nil)
	// Subscriber created outside window — must not appear in the result counts.
	insertSubscriber(t, ctx, db, "old@example.com", outOfWindow, nil, nil, nil, nil)
	// Subscriber bounced within window.
	bouncedAt := ptr(inWindow.Add(3 * time.Hour))
	insertSubscriber(t, ctx, db, "bounce@example.com", inWindow, confirmedAt, nil, bouncedAt, nil)

	filter := engagement.MetricsFilter{From: &from, To: &now}

	t.Run("With time filter returns no error", func(t *testing.T) {
		got, err := s.SubscriberGrowth(ctx, filter, "day")
		require.NoError(t, err)
		assert.Equal(t, "day", got.Bucket)
		assert.NotEmpty(t, got.Points, "should have at least one bucket for the in-window events")

		var totalNew, totalConfirmed, totalUnsub, totalLost int64
		for _, p := range got.Points {
			totalNew += p.New
			totalConfirmed += p.Confirmed
			totalUnsub += p.Unsubscribed
			totalLost += p.Lost
		}
		assert.Equal(t, int64(2), totalNew, "alice + bounce are in-window new subscribers")
		assert.Equal(t, int64(2), totalConfirmed, "alice + bounce were confirmed in-window")
		assert.Equal(t, int64(1), totalUnsub, "alice unsubscribed in-window")
		assert.Equal(t, int64(1), totalLost, "bounce was bounced in-window")
	})

	t.Run("Without filter returns all subscribers", func(t *testing.T) {
		got, err := s.SubscriberGrowth(ctx, engagement.MetricsFilter{}, "day")
		require.NoError(t, err)

		var totalNew int64
		for _, p := range got.Points {
			totalNew += p.New
		}
		assert.Equal(t, int64(3), totalNew, "all three subscribers should appear with no filter")
	})

	t.Run("Week bucket returns no error", func(t *testing.T) {
		got, err := s.SubscriberGrowth(ctx, filter, "week")
		require.NoError(t, err)
		assert.Equal(t, "week", got.Bucket)
	})
}

func TestStore_Summary(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	s := metricsstore.New(db)

	now := time.Now().UTC()
	from := now.Add(-7 * 24 * time.Hour)

	issue, err := issues.New(db).Create(ctx, news.Issue{
		Slug:    "2026-05-20",
		Subject: "GoDaily - May 20, 2026",
		Status:  news.IssueStatusSent,
		SentAt:  now.Add(-2 * 24 * time.Hour),
	})
	require.NoError(t, err)

	subs := subscribers.New(db)
	subA, err := subs.Create(ctx, "a@example.com")
	require.NoError(t, err)
	subB, err := subs.Create(ctx, "b@example.com")
	require.NoError(t, err)

	ee := emailevents.New(db)
	for _, ev := range []engagement.EmailEvent{
		{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeDelivered, EventID: "evt-del-a"},
		{IssueID: &issue.ID, SubscriberID: &subB.ID, Email: "b@example.com", Type: engagement.EmailEventTypeDelivered, EventID: "evt-del-b"},
		{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeOpened, EventID: "evt-open-a"},
		{IssueID: &issue.ID, SubscriberID: &subA.ID, Email: "a@example.com", Type: engagement.EmailEventTypeClicked, EventID: "evt-click-a"},
	} {
		_, err := ee.Create(ctx, ev)
		require.NoError(t, err)
	}

	filter := engagement.MetricsFilter{From: &from, To: &now}

	got, err := s.Summary(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(2), got.Delivered)
	assert.Equal(t, int64(1), got.UniqueOpens)
	assert.Equal(t, int64(1), got.UniqueClicks)
	assert.InDelta(t, 0.5, got.OpenRate, 0.001)
	assert.InDelta(t, 0.5, got.ClickRate, 0.001)
}

func TestStore_IssueList(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	s := metricsstore.New(db)

	now := time.Now().UTC()
	from := now.Add(-7 * 24 * time.Hour)

	issue, err := issues.New(db).Create(ctx, news.Issue{
		Slug:    "2026-05-20",
		Subject: "GoDaily - May 20, 2026",
		Status:  news.IssueStatusSent,
		SentAt:  now.Add(-2 * 24 * time.Hour),
	})
	require.NoError(t, err)

	sub, err := subscribers.New(db).Create(ctx, "a@example.com")
	require.NoError(t, err)

	ee := emailevents.New(db)
	for _, ev := range []engagement.EmailEvent{
		{IssueID: &issue.ID, SubscriberID: &sub.ID, Email: "a@example.com", Type: engagement.EmailEventTypeDelivered, EventID: "il-del"},
		{IssueID: &issue.ID, SubscriberID: &sub.ID, Email: "a@example.com", Type: engagement.EmailEventTypeClicked, EventID: "il-click"},
	} {
		_, err := ee.Create(ctx, ev)
		require.NoError(t, err)
	}

	filter := engagement.MetricsFilter{From: &from, To: &now, Limit: 5}

	got, err := s.IssueList(ctx, filter, "click_rate")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, issue.ID, got[0].IssueID)
	assert.Equal(t, int64(1), got[0].Delivered)
	assert.Equal(t, int64(1), got[0].UniqueClicks)
	assert.InDelta(t, 1.0, got[0].ClickRate, 0.001)
}

func TestStore_ItemList(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()

	s := metricsstore.New(db)

	now := time.Now().UTC()
	from := now.Add(-7 * 24 * time.Hour)

	issue, err := issues.New(db).Create(ctx, news.Issue{
		Slug:    "2026-05-20",
		Subject: "GoDaily - May 20, 2026",
		Status:  news.IssueStatusSent,
		SentAt:  now.Add(-2 * 24 * time.Hour),
	})
	require.NoError(t, err)

	item, err := items.New(db).Create(ctx, &issue.ID, 1, news.Item{
		Source: "The Go Blog",
		Tag:    "language",
		Title:  "Go 1.26 is released",
		URL:    "https://go.dev/blog/go1.26",
	})
	require.NoError(t, err)

	sub, err := subscribers.New(db).Create(ctx, "a@example.com")
	require.NoError(t, err)

	ee := emailevents.New(db)
	_, err = ee.Create(ctx, engagement.EmailEvent{
		IssueID: &issue.ID, SubscriberID: &sub.ID, ItemID: &item.ID,
		Email: "a@example.com", Type: engagement.EmailEventTypeClicked, URL: item.URL, EventID: "item-click",
	})
	require.NoError(t, err)

	filter := engagement.MetricsFilter{From: &from, To: &now, Limit: 5}

	got, err := s.ItemList(ctx, filter)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, item.ID, got[0].ItemID)
	assert.Equal(t, int64(1), got[0].Clicks)
}

func ptr(t time.Time) *time.Time { return &t }
