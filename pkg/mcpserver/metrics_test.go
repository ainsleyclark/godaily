// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mcpserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

func TestMetricsSummary(t *testing.T) {
	t.Parallel()

	t.Run("Explicit dates", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		stats := engagement.SummaryStats{From: "2026-11-01", To: "2026-11-08", IssuesSent: 5, Delivered: 500, UniqueOpens: 250, OpenRate: 0.5, ClickRate: 0.1}
		h.Metrics.EXPECT().Summary(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, f engagement.MetricsFilter) (engagement.SummaryStats, error) {
			require.NotNil(t, f.From)
			require.NotNil(t, f.To)
			assert.Equal(t, "2026-11-01", f.From.Format(time.DateOnly))
			assert.Equal(t, "2026-11-08", f.To.Format(time.DateOnly))
			return stats, nil
		})

		out := decode[engagement.SummaryStats](t, h, "metrics_summary", map[string]any{"from": "2026-11-01", "to": "2026-11-08"})
		assert.Equal(t, stats, out)
	})

	t.Run("No window means all time", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		h.Metrics.EXPECT().Summary(gomock.Any(), engagement.MetricsFilter{}).Return(engagement.SummaryStats{}, nil)
		decode[engagement.SummaryStats](t, h, "metrics_summary", map[string]any{})
	})

	t.Run("Period all means all time", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		h.Metrics.EXPECT().Summary(gomock.Any(), engagement.MetricsFilter{}).Return(engagement.SummaryStats{}, nil)
		decode[engagement.SummaryStats](t, h, "metrics_summary", map[string]any{"period": "all"})
	})

	t.Run("Repository error", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		h.Metrics.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{}, errors.New("boom"))
		assert.Contains(t, toolError(t, h, "metrics_summary", map[string]any{"period": "week"}), "failed to fetch summary stats")
	})
}

func TestListIssueEngagement(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		args      map[string]any
		wantSort  string
		wantLimit int
	}{
		"Defaults":            {args: map[string]any{}, wantSort: "sent_at", wantLimit: 10},
		"Custom sort":         {args: map[string]any{"sort": "click_rate", "limit": 3}, wantSort: "click_rate", wantLimit: 3},
		"Limit capped at 100": {args: map[string]any{"limit": 1000}, wantSort: "sent_at", wantLimit: 100},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := setup(t)

			rows := []engagement.IssueEngagement{{IssueID: 42, Slug: "2026-11-18", SentAt: sentAt, Delivered: 100, ClickRate: 0.2}}
			h.Metrics.EXPECT().IssueList(gomock.Any(), engagement.MetricsFilter{Limit: test.wantLimit}, test.wantSort).Return(rows, nil)

			out := decode[IssueEngagementListOutput](t, h, "list_issue_engagement", test.args)
			assert.Equal(t, rows, out.Issues)
		})
	}
}

func TestGetIssueEngagement(t *testing.T) {
	t.Parallel()

	h := setup(t)
	h.Issues.EXPECT().FindBySlug(gomock.Any(), "2026-11-18").Return(sentIssue, nil)
	stats := engagement.IssueStats{IssueID: 42, Delivered: 100, UniqueClicks: 30, ClickRate: 0.3}
	links := []engagement.LinkClicks{{URL: "https://go.dev/blog", Title: "Go 1.27 freeze", Clicks: 20}}
	h.EmailEvents.EXPECT().IssueStats(gomock.Any(), int64(42)).Return(stats, nil)
	h.EmailEvents.EXPECT().TopLinks(gomock.Any(), int64(42), int64(topLinksLimit)).Return(links, nil)

	out := decode[IssueEngagementOutput](t, h, "get_issue_engagement", map[string]any{"key": "2026-11-18"})
	assert.Equal(t, IssueEngagementOutput{
		IssueID: 42,
		Slug:    "2026-11-18",
		Subject: sentIssue.Subject,
		SentAt:  sentAt,
		Stats:   stats,
		Links:   links,
	}, out)
}

func TestTopLists(t *testing.T) {
	t.Parallel()

	t.Run("top_items", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		rows := []engagement.ItemMetrics{{ItemID: 7, Title: "Go 1.27 freeze", URL: "https://go.dev", Tag: "official", Source: "go_blog", Clicks: 40}}
		h.Metrics.EXPECT().ItemList(gomock.Any(), engagement.MetricsFilter{Limit: 5}).Return(rows, nil)

		out := decode[TopItemsOutput](t, h, "top_items", map[string]any{"limit": 5})
		assert.Equal(t, rows, out.Items)
	})

	t.Run("top_tags", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		rows := []engagement.TagMetrics{{Tag: "official", Clicks: 40}}
		h.Metrics.EXPECT().TagList(gomock.Any(), engagement.MetricsFilter{Limit: 10}).Return(rows, nil)

		out := decode[TopTagsOutput](t, h, "top_tags", map[string]any{})
		assert.Equal(t, rows, out.Tags)
	})

	t.Run("top_sources with period", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		rows := []engagement.SourceMetrics{{Source: "hacker_news", Clicks: 90}}
		h.Metrics.EXPECT().SourceList(gomock.Any(), gomock.Cond(func(f engagement.MetricsFilter) bool {
			return f.From != nil && f.To != nil && f.Limit == 10 && f.To.Sub(*f.From) > 29*24*time.Hour
		})).Return(rows, nil)

		out := decode[TopSourcesOutput](t, h, "top_sources", map[string]any{"period": "month"})
		assert.Equal(t, rows, out.Sources)
	})

	t.Run("Empty result encodes as an empty list", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.Metrics.EXPECT().TagList(gomock.Any(), gomock.Any()).Return(nil, nil)

		out := decode[TopTagsOutput](t, h, "top_tags", map[string]any{})
		assert.Empty(t, out.Tags)
	})
}

func TestEngagementTrend(t *testing.T) {
	t.Parallel()

	t.Run("Defaults", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		data := engagement.TrendData{Metric: "click_rate", Bucket: "day", Points: []engagement.TrendPoint{{BucketStart: "2026-11-18", Value: 0.2, Delivered: 100}}}
		h.Metrics.EXPECT().Trend(gomock.Any(), engagement.MetricsFilter{}, "click_rate", "day").Return(data, nil)

		out := decode[engagement.TrendData](t, h, "engagement_trend", map[string]any{})
		assert.Equal(t, data, out)
	})

	t.Run("Explicit metric and bucket", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.Metrics.EXPECT().Trend(gomock.Any(), gomock.Any(), "unique_opens", "week").Return(engagement.TrendData{Metric: "unique_opens", Bucket: "week"}, nil)

		decode[engagement.TrendData](t, h, "engagement_trend", map[string]any{"metric": "unique_opens", "bucket": "week", "period": "year"})
	})
}

func TestIssueEngagementTrend(t *testing.T) {
	t.Parallel()

	t.Run("Defaults to 48 hours after send", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		old := sentIssue
		old.SentAt = time.Date(2026, 10, 1, 8, 30, 0, 0, time.UTC)
		h.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(old, nil)
		h.Metrics.EXPECT().IssueTrend(gomock.Any(), int64(42), gomock.Any(), "unique_clicks", "hour").
			DoAndReturn(func(_ context.Context, _ int64, f engagement.MetricsFilter, metric, bucket string) (engagement.TrendData, error) {
				require.NotNil(t, f.From)
				require.NotNil(t, f.To)
				wantFrom := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC) // truncated to the hour
				assert.Equal(t, wantFrom, *f.From)
				assert.Equal(t, wantFrom.Add(48*time.Hour), *f.To)
				return engagement.TrendData{Metric: metric, Bucket: bucket}, nil
			})

		out := decode[engagement.TrendData](t, h, "issue_engagement_trend", map[string]any{"key": "42"})
		assert.Equal(t, "hour", out.Bucket)
	})

	t.Run("Recent issue is clamped to now", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		recent := sentIssue
		recent.SentAt = time.Now().UTC().Add(-2 * time.Hour)
		h.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(recent, nil)
		h.Metrics.EXPECT().IssueTrend(gomock.Any(), int64(42), gomock.Any(), "click_rate", "day").
			DoAndReturn(func(_ context.Context, _ int64, f engagement.MetricsFilter, metric, bucket string) (engagement.TrendData, error) {
				require.NotNil(t, f.To)
				assert.WithinDuration(t, time.Now().UTC(), *f.To, time.Minute)
				return engagement.TrendData{}, nil
			})

		decode[engagement.TrendData](t, h, "issue_engagement_trend", map[string]any{"key": "42", "metric": "click_rate", "bucket": "day"})
	})

	t.Run("Explicit window is respected", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(sentIssue, nil)
		h.Metrics.EXPECT().IssueTrend(gomock.Any(), int64(42), gomock.Cond(func(f engagement.MetricsFilter) bool {
			return f.From != nil && f.From.Format(time.DateOnly) == "2026-11-18" && f.To != nil && f.To.Format(time.DateOnly) == "2026-11-25"
		}), "unique_clicks", "day").Return(engagement.TrendData{}, nil)

		decode[engagement.TrendData](t, h, "issue_engagement_trend", map[string]any{"key": "42", "from": "2026-11-18", "to": "2026-11-25", "bucket": "day"})
	})

	t.Run("Invalid window fails before lookup", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		assert.Contains(t, toolError(t, h, "issue_engagement_trend", map[string]any{"key": "42", "to": "nope"}), "invalid to date")
	})
}

func TestSubscriberGrowth(t *testing.T) {
	t.Parallel()

	h := setup(t)
	data := engagement.SubscriberData{Bucket: "week", Points: []engagement.SubscriberPoint{{BucketStart: "2026-11-16", New: 10, Confirmed: 8, Unsubscribed: 1, NetChange: 7, ActiveAtEnd: 1200}}}
	h.Metrics.EXPECT().SubscriberGrowth(gomock.Any(), gomock.Any(), "week").Return(data, nil)

	out := decode[engagement.SubscriberData](t, h, "subscriber_growth", map[string]any{"bucket": "week", "period": "month"})
	assert.Equal(t, data, out)
}

func TestSocialPostMetrics(t *testing.T) {
	t.Parallel()

	posts := make([]engagement.SocialPostEngagement, 30)
	issueID := int64(42)
	for i := range posts {
		posts[i] = engagement.SocialPostEngagement{ID: int64(i + 1), IssueID: &issueID, Kind: "featured", Platform: "bluesky", Text: "post", PostedAt: sentAt, Likes: int64(i)}
	}

	t.Run("Default limit truncates and reports total", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.SocialMetrics.EXPECT().List(gomock.Any(), engagement.MetricsFilter{}).Return(posts, nil)

		out := decode[SocialPostsOutput](t, h, "social_post_metrics", map[string]any{})
		assert.Len(t, out.Posts, defaultSocialLimit)
		assert.Equal(t, 30, out.Total)
		assert.Equal(t, int64(1), out.Posts[0].ID)
	})

	t.Run("Custom limit", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.SocialMetrics.EXPECT().List(gomock.Any(), gomock.Any()).Return(posts, nil)

		out := decode[SocialPostsOutput](t, h, "social_post_metrics", map[string]any{"limit": 2, "period": "week"})
		assert.Len(t, out.Posts, 2)
	})
}
