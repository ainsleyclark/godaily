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

package engagement

import (
	"context"
	"time"
)

// MetricsFilter restricts which events are included in an aggregate query.
// Nil bounds mean open-ended (no restriction on that side).
type MetricsFilter struct {
	From  *time.Time
	To    *time.Time
	Limit int
}

// SummaryStats holds headline engagement numbers for a period.
type SummaryStats struct {
	From                     string  `json:"from"`
	To                       string  `json:"to"`
	IssuesSent               int64   `json:"issues_sent"`
	Delivered                int64   `json:"delivered"`
	UniqueOpens              int64   `json:"unique_opens"`
	TotalOpens               int64   `json:"total_opens"`
	UniqueClicks             int64   `json:"unique_clicks"`
	TotalClicks              int64   `json:"total_clicks"`
	Bounced                  int64   `json:"bounced"`
	Complained               int64   `json:"complained"`
	OpenRate                 float64 `json:"open_rate"`
	ClickRate                float64 `json:"click_rate"`
	UniqueSubscribersEngaged int64   `json:"unique_subscribers_engaged"`
}

// IssueEngagement holds aggregate email engagement for a single digest issue.
type IssueEngagement struct {
	IssueID      int64     `json:"issue_id"`
	Slug         string    `json:"slug"`
	SentAt       time.Time `json:"sent_at"`
	Delivered    int64     `json:"delivered"`
	UniqueOpens  int64     `json:"unique_opens"`
	TotalOpens   int64     `json:"total_opens"`
	UniqueClicks int64     `json:"unique_clicks"`
	TotalClicks  int64     `json:"total_clicks"`
	Bounced      int64     `json:"bounced"`
	Complained   int64     `json:"complained"`
	Delayed      int64     `json:"delayed"`
	Failed       int64     `json:"failed"`
	Suppressed   int64     `json:"suppressed"`
	OpenRate     float64   `json:"open_rate"`
	ClickRate    float64   `json:"click_rate"`
}

// ItemMetrics holds click counts for a single news item.
type ItemMetrics struct {
	ItemID int64  `json:"item_id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Tag    string `json:"tag"`
	Source string `json:"source"`
	Clicks int64  `json:"clicks"`
}

// TagMetrics holds click counts aggregated by item tag.
type TagMetrics struct {
	Tag    string `json:"tag"`
	Clicks int64  `json:"clicks"`
}

// SourceMetrics holds click counts aggregated by item source.
type SourceMetrics struct {
	Source string `json:"source"`
	Clicks int64  `json:"clicks"`
}

// TrendPoint is a single time-series bucket for an engagement metric.
type TrendPoint struct {
	BucketStart string  `json:"bucket_start"`
	Value       float64 `json:"value"`
	Delivered   int64   `json:"delivered"`
}

// TrendData is the time-series response for the trend endpoint.
type TrendData struct {
	Metric string       `json:"metric"`
	Bucket string       `json:"bucket"`
	Points []TrendPoint `json:"points"`
}

// SubscriberPoint is a single time-series bucket for subscriber growth.
type SubscriberPoint struct {
	BucketStart  string `json:"bucket_start"`
	New          int64  `json:"new"`
	Confirmed    int64  `json:"confirmed"`
	Unsubscribed int64  `json:"unsubscribed"`
	Lost         int64  `json:"lost"`
	NetChange    int64  `json:"net_change"`
	ActiveAtEnd  int64  `json:"active_at_end"`
}

// SubscriberData is the time-series response for the subscriber growth endpoint.
type SubscriberData struct {
	Bucket string            `json:"bucket"`
	Points []SubscriberPoint `json:"points"`
}

//go:generate go run go.uber.org/mock/mockgen -package=mockengagement -destination=../../mocks/domain/engagement/MetricsRepository.go . MetricsRepository

// MetricsRepository answers engagement analytics queries.
type MetricsRepository interface {
	// Summary returns headline engagement numbers for the given filter window.
	Summary(ctx context.Context, f MetricsFilter) (SummaryStats, error)

	// IssueList returns per-issue engagement stats, ordered by sort descending.
	IssueList(ctx context.Context, f MetricsFilter, sort string) ([]IssueEngagement, error)

	// ItemList returns the top-clicked news items, ordered by clicks descending.
	ItemList(ctx context.Context, f MetricsFilter) ([]ItemMetrics, error)

	// TagList returns clicks aggregated by item tag, ordered by clicks descending.
	TagList(ctx context.Context, f MetricsFilter) ([]TagMetrics, error)

	// SourceList returns clicks aggregated by item source, ordered by clicks descending.
	SourceList(ctx context.Context, f MetricsFilter) ([]SourceMetrics, error)

	// Trend returns a time-series for the requested metric, bucketed by day or week.
	Trend(ctx context.Context, f MetricsFilter, metric, bucket string) (TrendData, error)

	// SubscriberGrowth returns subscriber growth and churn bucketed over time.
	SubscriberGrowth(ctx context.Context, f MetricsFilter, bucket string) (SubscriberData, error)
}
