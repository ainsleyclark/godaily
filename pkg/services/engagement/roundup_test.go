// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package engagement

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
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	"github.com/ainsleyclark/godaily/pkg/mocks/slack"
	slacksdk "github.com/slack-go/slack"
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
	today := time.Date(fixedNow.Year(), fixedNow.Month(), fixedNow.Day(), 0, 0, 0, 0, time.UTC)
	wantCurrFrom := today.AddDate(0, 0, -7)
	wantPrevFrom := wantCurrFrom.AddDate(0, 0, -7)

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
		sender.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req slack.Request) error {
			captured = flattenRequest(req)
			return nil
		})

		require.NoError(t, svc.Roundup(context.Background()))

		assert.Contains(t, captured, "Weekly Roundup")
		assert.Contains(t, captured, "17 May – 24 May")
		assert.Contains(t, captured, "Issues sent")
		assert.Contains(t, captured, "Delivered")
		assert.Contains(t, captured, "1,243") // delivered count
		assert.Contains(t, captured, "↑")     // delivered went up vs prior
		assert.Contains(t, captured, "1,312") // subscriber active count
		assert.Contains(t, captured, "<https://go.dev/blog/go1.24|Go 1.24 released>")
		assert.Contains(t, captured, "Top tags*: ai (88)")
		assert.Contains(t, captured, "Top sources*: HN (120)")
		assert.Contains(t, captured, "Best issue:* 2026-05-22")
	})

	t.Run("Handles empty data gracefully", func(t *testing.T) {
		t.Parallel()
		svc, repo, sender := newService(t)
		svc.now = func() time.Time { return fixedNow }

		// Both windows empty.
		empty := engagement.SummaryStats{}
		emptySubs := engagement.SubscriberData{}
		for range 2 {
			repo.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(empty, nil)
			repo.EXPECT().SubscriberGrowth(gomock.Any(), gomock.Any(), "week").Return(emptySubs, nil)
			repo.EXPECT().ItemList(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().TagList(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().SourceList(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().IssueList(gomock.Any(), gomock.Any(), "click_rate").Return(nil, nil)
		}

		var captured string
		sender.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req slack.Request) error {
			captured = flattenRequest(req)
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

		for range 2 {
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

// flattenRequest concatenates the Request's Text + every section block's
// text and fields into one string for assertion convenience.
func flattenRequest(req slack.Request) string {
	var b strings.Builder
	b.WriteString(req.Text)
	for _, blk := range req.Blocks.BlockSet {
		switch v := blk.(type) {
		case *slacksdk.SectionBlock:
			if v.Text != nil {
				b.WriteString("\n")
				b.WriteString(v.Text.Text)
			}
			for _, f := range v.Fields {
				b.WriteString("\n")
				b.WriteString(f.Text)
			}
		case *slacksdk.HeaderBlock:
			if v.Text != nil {
				b.WriteString("\n")
				b.WriteString(v.Text.Text)
			}
		case *slacksdk.ContextBlock:
			for _, e := range v.ContextElements.Elements {
				if t, ok := e.(*slacksdk.TextBlockObject); ok {
					b.WriteString("\n")
					b.WriteString(t.Text)
				}
			}
		}
	}
	return b.String()
}
