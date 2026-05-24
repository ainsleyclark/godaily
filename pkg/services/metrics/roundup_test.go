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

package metrics

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/domain/engagement"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
)

var errBoom = errors.New("boom")

func newService(t *testing.T) (*Service, *mockengagement.MockMetricsRepository, *mockslack.MockSender) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mockengagement.NewMockMetricsRepository(ctrl)
	sender := mockslack.NewMockSender(ctrl)
	svc := New(repo, sender)
	return svc, repo, sender
}

func TestService_Gather(t *testing.T) {
	t.Parallel()

	from := time.Date(2026, 5, 17, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)

	t.Run("Returns populated snapshot", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := newService(t)

		summary := engagement.SummaryStats{IssuesSent: 7, Delivered: 1000, OpenRate: 0.5}
		subs := engagement.SubscriberData{
			Bucket: "week",
			Points: []engagement.SubscriberPoint{{ActiveAtEnd: 1312, NetChange: 19}},
		}
		items := []engagement.ItemMetrics{{ItemID: 1, Title: "Go 1.24", Clicks: 42}}
		tags := []engagement.TagMetrics{{Tag: "ai", Clicks: 88}}
		sources := []engagement.SourceMetrics{{Source: "HN", Clicks: 120}}
		bestIssues := []engagement.IssueEngagement{{Slug: "2026-05-22", ClickRate: 0.173}}

		repo.EXPECT().Summary(gomock.Any(), engagement.MetricsFilter{From: &from, To: &to}).Return(summary, nil)
		repo.EXPECT().SubscriberGrowth(gomock.Any(), engagement.MetricsFilter{From: &from, To: &to}, "week").Return(subs, nil)
		repo.EXPECT().ItemList(gomock.Any(), engagement.MetricsFilter{From: &from, To: &to, Limit: topItemsLimit}).Return(items, nil)
		repo.EXPECT().TagList(gomock.Any(), engagement.MetricsFilter{From: &from, To: &to, Limit: topTagsLimit}).Return(tags, nil)
		repo.EXPECT().SourceList(gomock.Any(), engagement.MetricsFilter{From: &from, To: &to, Limit: topSourcesLimit}).Return(sources, nil)
		repo.EXPECT().IssueList(gomock.Any(), engagement.MetricsFilter{From: &from, To: &to, Limit: 1}, "click_rate").Return(bestIssues, nil)

		snap, err := svc.Gather(context.Background(), from, to)
		require.NoError(t, err)
		assert.Equal(t, summary, snap.Summary)
		assert.Equal(t, subs, snap.Subs)
		assert.Equal(t, items, snap.Items)
		assert.Equal(t, tags, snap.Tags)
		assert.Equal(t, sources, snap.Sources)
		require.NotNil(t, snap.BestIssue)
		assert.Equal(t, "2026-05-22", snap.BestIssue.Slug)
	})

	t.Run("Best issue is nil when no issues", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := newService(t)

		repo.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{}, nil)
		repo.EXPECT().SubscriberGrowth(gomock.Any(), gomock.Any(), "week").Return(engagement.SubscriberData{}, nil)
		repo.EXPECT().ItemList(gomock.Any(), gomock.Any()).Return(nil, nil)
		repo.EXPECT().TagList(gomock.Any(), gomock.Any()).Return(nil, nil)
		repo.EXPECT().SourceList(gomock.Any(), gomock.Any()).Return(nil, nil)
		repo.EXPECT().IssueList(gomock.Any(), gomock.Any(), "click_rate").Return(nil, nil)

		snap, err := svc.Gather(context.Background(), from, to)
		require.NoError(t, err)
		assert.Nil(t, snap.BestIssue)
	})

	t.Run("Summary error propagates", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := newService(t)
		repo.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{}, errBoom)

		_, err := svc.Gather(context.Background(), from, to)
		require.Error(t, err)
	})
}

func TestService_Roundup(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 5, 24, 15, 0, 0, 0, time.UTC)
	wantTo := fixedNow
	wantCurrFrom := fixedNow.Add(-roundupWindow)
	wantPrevFrom := wantCurrFrom.Add(-roundupWindow)

	t.Run("Formats message and sends to Slack", func(t *testing.T) {
		t.Parallel()
		svc, repo, sender := newService(t)
		svc.now = func() time.Time { return fixedNow }

		// Current window.
		repo.EXPECT().Summary(gomock.Any(), engagement.MetricsFilter{From: &wantCurrFrom, To: &wantTo}).Return(
			engagement.SummaryStats{IssuesSent: 7, Delivered: 1243, UniqueOpens: 612, UniqueClicks: 187, OpenRate: 0.492, ClickRate: 0.150}, nil,
		)
		repo.EXPECT().SubscriberGrowth(gomock.Any(), engagement.MetricsFilter{From: &wantCurrFrom, To: &wantTo}, "week").Return(
			engagement.SubscriberData{Points: []engagement.SubscriberPoint{{New: 28, Confirmed: 24, Unsubscribed: 5, NetChange: 19, ActiveAtEnd: 1312}}}, nil,
		)
		repo.EXPECT().ItemList(gomock.Any(), gomock.Any()).Return([]engagement.ItemMetrics{
			{ItemID: 1, Title: "Go 1.24 released", URL: "https://go.dev/blog/go1.24", Source: "go.dev", Clicks: 42},
		}, nil)
		repo.EXPECT().TagList(gomock.Any(), gomock.Any()).Return([]engagement.TagMetrics{{Tag: "ai", Clicks: 88}}, nil)
		repo.EXPECT().SourceList(gomock.Any(), gomock.Any()).Return([]engagement.SourceMetrics{{Source: "HN", Clicks: 120}}, nil)
		repo.EXPECT().IssueList(gomock.Any(), gomock.Any(), "click_rate").Return([]engagement.IssueEngagement{
			{Slug: "2026-05-22", ClickRate: 0.173, OpenRate: 0.531},
		}, nil)

		// Prior window — used only for deltas.
		repo.EXPECT().Summary(gomock.Any(), engagement.MetricsFilter{From: &wantPrevFrom, To: &wantCurrFrom}).Return(
			engagement.SummaryStats{IssuesSent: 7, Delivered: 1200, UniqueOpens: 598, UniqueClicks: 200, OpenRate: 0.480, ClickRate: 0.160}, nil,
		)
		repo.EXPECT().SubscriberGrowth(gomock.Any(), engagement.MetricsFilter{From: &wantPrevFrom, To: &wantCurrFrom}, "week").Return(
			engagement.SubscriberData{Points: []engagement.SubscriberPoint{{ActiveAtEnd: 1293}}}, nil,
		)
		repo.EXPECT().ItemList(gomock.Any(), gomock.Any()).Return(nil, nil)
		repo.EXPECT().TagList(gomock.Any(), gomock.Any()).Return(nil, nil)
		repo.EXPECT().SourceList(gomock.Any(), gomock.Any()).Return(nil, nil)
		repo.EXPECT().IssueList(gomock.Any(), gomock.Any(), "click_rate").Return(nil, nil)

		var captured string
		sender.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, msg string) error {
			captured = msg
			return nil
		})

		require.NoError(t, svc.Roundup(context.Background()))

		assert.Contains(t, captured, "GoDaily — Weekly Roundup")
		assert.Contains(t, captured, "17 May – 24 May")
		assert.Contains(t, captured, "Issues sent: 7")
		assert.Contains(t, captured, "Delivered: 1,243")
		assert.Contains(t, captured, "↑")             // delivered went up vs prior
		assert.Contains(t, captured, "Active: 1,312") // subscriber active count
		assert.Contains(t, captured, "<https://go.dev/blog/go1.24|Go 1.24 released>")
		assert.Contains(t, captured, "Top tags*: ai (88)")
		assert.Contains(t, captured, "Top sources*: HN (120)")
		assert.Contains(t, captured, "Best issue*: 2026-05-22")
	})

	t.Run("Handles empty data gracefully", func(t *testing.T) {
		t.Parallel()
		svc, repo, sender := newService(t)
		svc.now = func() time.Time { return fixedNow }

		// Both windows empty.
		empty := engagement.SummaryStats{}
		emptySubs := engagement.SubscriberData{}
		for i := 0; i < 2; i++ {
			repo.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(empty, nil)
			repo.EXPECT().SubscriberGrowth(gomock.Any(), gomock.Any(), "week").Return(emptySubs, nil)
			repo.EXPECT().ItemList(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().TagList(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().SourceList(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().IssueList(gomock.Any(), gomock.Any(), "click_rate").Return(nil, nil)
		}

		var captured string
		sender.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, msg string) error {
			captured = msg
			return nil
		})

		require.NoError(t, svc.Roundup(context.Background()))
		assert.Contains(t, captured, "No clicks recorded this week")
		assert.Contains(t, captured, "No subscriber activity this week")
		assert.NotContains(t, captured, "Best issue")
	})

	t.Run("Slack send error propagates", func(t *testing.T) {
		t.Parallel()
		svc, repo, sender := newService(t)
		svc.now = func() time.Time { return fixedNow }

		for i := 0; i < 2; i++ {
			repo.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{}, nil)
			repo.EXPECT().SubscriberGrowth(gomock.Any(), gomock.Any(), "week").Return(engagement.SubscriberData{}, nil)
			repo.EXPECT().ItemList(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().TagList(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().SourceList(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().IssueList(gomock.Any(), gomock.Any(), "click_rate").Return(nil, nil)
		}
		sender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(errBoom)

		err := svc.Roundup(context.Background())
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("Gather error short-circuits send", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := newService(t)
		svc.now = func() time.Time { return fixedNow }

		repo.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{}, errBoom)

		err := svc.Roundup(context.Background())
		require.Error(t, err)
	})
}

func TestDeltaCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		curr, prev       int64
		wantContainsAll  []string
		wantNotContains  []string
		wantExactlyEqual string
	}{
		{name: "Both zero", curr: 0, prev: 0, wantExactlyEqual: "(–)"},
		{name: "New from zero", curr: 5, prev: 0, wantExactlyEqual: "(new)"},
		{name: "Negative from zero", curr: -3, prev: 0, wantContainsAll: []string{"-3"}},
		{name: "Increase", curr: 110, prev: 100, wantContainsAll: []string{"↑", "+10.0%"}},
		{name: "Decrease", curr: 90, prev: 100, wantContainsAll: []string{"↓", "-10.0%"}, wantNotContains: []string{"+"}},
		{name: "Unchanged", curr: 100, prev: 100, wantExactlyEqual: "(–)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := deltaCount(tc.curr, tc.prev)
			if tc.wantExactlyEqual != "" {
				assert.Equal(t, tc.wantExactlyEqual, got)
			}
			for _, sub := range tc.wantContainsAll {
				assert.Contains(t, got, sub)
			}
			for _, sub := range tc.wantNotContains {
				assert.NotContains(t, got, sub)
			}
		})
	}
}

func TestDeltaPoint(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "(–)", deltaPoint(0, 0))
	assert.Contains(t, deltaPoint(0.50, 0.48), "↑")
	assert.Contains(t, deltaPoint(0.50, 0.48), "pp")
	assert.Contains(t, deltaPoint(0.48, 0.50), "↓")
}

func TestHumanCount(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		in   int64
		want string
	}{
		"under 1k":            {in: 42, want: "42"},
		"thousands separator": {in: 1234, want: "1,234"},
		"just under 10k":      {in: 9999, want: "9,999"},
		"compact at 10k":      {in: 12345, want: "12.3k"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, humanCount(tc.in))
		})
	}
}

func TestFormatRoundup_LengthSanity(t *testing.T) {
	t.Parallel()
	// Sanity check: even with full top-N lists, message stays well under Slack's 4000 char limit.
	curr := Snapshot{
		From: time.Date(2026, 5, 17, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC),
		Summary: engagement.SummaryStats{
			IssuesSent: 7, Delivered: 1500, UniqueOpens: 700, UniqueClicks: 200,
			OpenRate: 0.5, ClickRate: 0.15,
		},
		Subs: engagement.SubscriberData{Points: []engagement.SubscriberPoint{{ActiveAtEnd: 1500, NetChange: 25, New: 30, Confirmed: 28, Unsubscribed: 5}}},
		Items: []engagement.ItemMetrics{
			{Title: strings.Repeat("X", 100), URL: "https://example.com/" + strings.Repeat("y", 80), Source: "src", Clicks: 50},
			{Title: strings.Repeat("X", 100), URL: "https://example.com/" + strings.Repeat("y", 80), Source: "src", Clicks: 40},
			{Title: strings.Repeat("X", 100), URL: "https://example.com/" + strings.Repeat("y", 80), Source: "src", Clicks: 30},
			{Title: strings.Repeat("X", 100), URL: "https://example.com/" + strings.Repeat("y", 80), Source: "src", Clicks: 20},
			{Title: strings.Repeat("X", 100), URL: "https://example.com/" + strings.Repeat("y", 80), Source: "src", Clicks: 10},
		},
		Tags:      []engagement.TagMetrics{{Tag: "a", Clicks: 1}, {Tag: "b", Clicks: 2}, {Tag: "c", Clicks: 3}},
		Sources:   []engagement.SourceMetrics{{Source: "x", Clicks: 1}, {Source: "y", Clicks: 2}, {Source: "z", Clicks: 3}},
		BestIssue: &engagement.IssueEngagement{Slug: "2026-05-22", ClickRate: 0.15, OpenRate: 0.5},
	}
	msg := formatRoundup(curr, Snapshot{})
	assert.Less(t, len(msg), 4000, "message must fit in Slack's 4000-char limit")
}
