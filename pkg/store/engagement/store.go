// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package engagement provides the MetricsRepository implementation backed by a SQL database.
package engagement

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/store/internal/sqlc"
)

// New creates a new metrics Store.
func New(db *sql.DB) *Store {
	return &Store{
		sqlc: sqlc.New(db),
		db:   db,
	}
}

// Store implements engagement.MetricsRepository.
type Store struct {
	sqlc *sqlc.Queries
	db   *sql.DB
}

var _ engagement.MetricsRepository = (*Store)(nil)

// nullableTime converts an optional time to the interface{} that sqlc.narg expects:
// nil when absent, RFC3339 string when present.
func nullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

// Summary returns headline engagement numbers for the given filter window.
func (s *Store) Summary(ctx context.Context, f engagement.MetricsFilter) (engagement.SummaryStats, error) {
	r, err := s.sqlc.MetricsSummary(ctx, sqlc.MetricsSummaryParams{
		From: nullableTime(f.From),
		To:   nullableTime(f.To),
	})
	if err != nil {
		return engagement.SummaryStats{}, err
	}

	stats := engagement.SummaryStats{
		From:                     formatDate(f.From),
		To:                       formatDate(f.To),
		IssuesSent:               r.IssuesSent,
		Delivered:                r.Delivered,
		UniqueOpens:              r.UniqueOpens,
		TotalOpens:               r.TotalOpens,
		UniqueClicks:             r.UniqueClicks,
		TotalClicks:              r.TotalClicks,
		Bounced:                  r.Bounced,
		Complained:               r.Complained,
		UniqueSubscribersEngaged: r.UniqueEngaged,
	}
	if r.Delivered > 0 {
		stats.OpenRate = float64(r.UniqueOpens) / float64(r.Delivered)
		stats.ClickRate = float64(r.UniqueClicks) / float64(r.Delivered)
	}
	return stats, nil
}

// issueSortExprs maps validated sort-key names to their SQL ORDER BY expressions.
// Raw SQL is required here because sqlc cannot parameterise ORDER BY expressions —
// only scalar values can be bound as parameters, not SQL fragments. The map is the
// sole injection guard: only keys present here are ever interpolated into the query.
var issueSortExprs = map[string]string{
	"click_rate":    "CAST(COUNT(DISTINCT CASE WHEN e.event_type='clicked' THEN e.subscriber_id END) AS REAL) / NULLIF(COUNT(CASE WHEN e.event_type='delivered' THEN 1 END), 0)",
	"open_rate":     "CAST(COUNT(DISTINCT CASE WHEN e.event_type='opened' THEN e.subscriber_id END) AS REAL) / NULLIF(COUNT(CASE WHEN e.event_type='delivered' THEN 1 END), 0)",
	"total_clicks":  "COUNT(CASE WHEN e.event_type='clicked' THEN 1 END)",
	"unique_clicks": "COUNT(DISTINCT CASE WHEN e.event_type='clicked' THEN e.subscriber_id END)",
	"total_opens":   "COUNT(CASE WHEN e.event_type='opened' THEN 1 END)",
	"unique_opens":  "COUNT(DISTINCT CASE WHEN e.event_type='opened' THEN e.subscriber_id END)",
	"delivered":     "COUNT(CASE WHEN e.event_type='delivered' THEN 1 END)",
	"sent_at":       "i.sent_at",
}

// IssueList returns per-issue engagement stats ordered by the given sort key descending.
//
// Raw SQL is required because the ORDER BY clause must contain a full aggregate
// expression (e.g. "CAST(COUNT(...) AS REAL) / NULLIF(...)") that is chosen at
// runtime. sqlc only supports binding scalar values as parameters — SQL fragments
// in ORDER BY are not parameterisable in any dialect.
func (s *Store) IssueList(ctx context.Context, f engagement.MetricsFilter, sortKey string) ([]engagement.IssueEngagement, error) {
	conds, args := timeConditions(f, "i.sent_at")

	orderExpr := issueSortExprs["sent_at"]
	if expr, ok := issueSortExprs[sortKey]; ok {
		orderExpr = expr
	}
	args = append(args, int64(f.Limit))

	query := fmt.Sprintf( /* #nosec G201 -- ORDER BY expression comes from issueSortExprs allowlist, not user input */
		`
SELECT
    i.id,
    i.slug,
    i.sent_at,
    COUNT(CASE          WHEN e.event_type = 'delivered'        THEN 1               END) AS delivered,
    COUNT(DISTINCT CASE WHEN e.event_type = 'opened'           THEN e.subscriber_id END) AS unique_opens,
    COUNT(CASE          WHEN e.event_type = 'opened'           THEN 1               END) AS total_opens,
    COUNT(DISTINCT CASE WHEN e.event_type = 'clicked'          THEN e.subscriber_id END) AS unique_clicks,
    COUNT(CASE          WHEN e.event_type = 'clicked'          THEN 1               END) AS total_clicks,
    COUNT(CASE          WHEN e.event_type = 'bounced'          THEN 1               END) AS bounced,
    COUNT(CASE          WHEN e.event_type = 'complained'       THEN 1               END) AS complained,
    COUNT(CASE          WHEN e.event_type = 'delivery_delayed' THEN 1               END) AS delayed,
    COUNT(CASE          WHEN e.event_type = 'failed'           THEN 1               END) AS failed,
    COUNT(CASE          WHEN e.event_type = 'suppressed'       THEN 1               END) AS suppressed
FROM email_events e
JOIN issues i ON i.id = e.issue_id
WHERE e.issue_id IS NOT NULL%s
GROUP BY i.id, i.slug, i.sent_at
ORDER BY %s DESC
LIMIT ?`, conds, orderExpr,
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []engagement.IssueEngagement
	for rows.Next() {
		var row engagement.IssueEngagement
		if err = rows.Scan(
			&row.IssueID, &row.Slug, &row.SentAt,
			&row.Delivered,
			&row.UniqueOpens, &row.TotalOpens,
			&row.UniqueClicks, &row.TotalClicks,
			&row.Bounced, &row.Complained,
			&row.Delayed, &row.Failed, &row.Suppressed,
		); err != nil {
			return nil, err
		}
		if row.Delivered > 0 {
			row.OpenRate = float64(row.UniqueOpens) / float64(row.Delivered)
			row.ClickRate = float64(row.UniqueClicks) / float64(row.Delivered)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ItemList returns the top-clicked news items enriched with item metadata.
func (s *Store) ItemList(ctx context.Context, f engagement.MetricsFilter) ([]engagement.ItemMetrics, error) {
	rows, err := s.sqlc.MetricsItemList(ctx, sqlc.MetricsItemListParams{
		From:  nullableTime(f.From),
		To:    nullableTime(f.To),
		Limit: int64(f.Limit),
	})
	if err != nil {
		return nil, err
	}

	out := make([]engagement.ItemMetrics, len(rows))
	for i, r := range rows {
		out[i] = engagement.ItemMetrics{
			ItemID: r.ID,
			Title:  r.Title,
			URL:    r.Url,
			Tag:    r.Tag,
			Source: r.Source,
			Clicks: r.Clicks,
		}
	}
	return out, nil
}

// TagList returns clicks aggregated by item tag, ordered by clicks descending.
func (s *Store) TagList(ctx context.Context, f engagement.MetricsFilter) ([]engagement.TagMetrics, error) {
	rows, err := s.sqlc.MetricsTagList(ctx, sqlc.MetricsTagListParams{
		From:  nullableTime(f.From),
		To:    nullableTime(f.To),
		Limit: int64(f.Limit),
	})
	if err != nil {
		return nil, err
	}

	out := make([]engagement.TagMetrics, len(rows))
	for i, r := range rows {
		out[i] = engagement.TagMetrics{Tag: r.Tag, Clicks: r.Clicks}
	}
	return out, nil
}

// SourceList returns clicks aggregated by item source, ordered by clicks descending.
func (s *Store) SourceList(ctx context.Context, f engagement.MetricsFilter) ([]engagement.SourceMetrics, error) {
	rows, err := s.sqlc.MetricsSourceList(ctx, sqlc.MetricsSourceListParams{
		From:  nullableTime(f.From),
		To:    nullableTime(f.To),
		Limit: int64(f.Limit),
	})
	if err != nil {
		return nil, err
	}

	out := make([]engagement.SourceMetrics, len(rows))
	for i, r := range rows {
		out[i] = engagement.SourceMetrics{Source: r.Source, Clicks: r.Clicks}
	}
	return out, nil
}

// trendBucket holds raw aggregated counts for a single time-series bucket.
type trendBucket struct {
	bucketStart  string
	delivered    int64
	uniqueOpens  int64
	totalOpens   int64
	uniqueClicks int64
	totalClicks  int64
}

// Trend returns a zero-filled time-series for the requested metric.
//
// Raw SQL is required because the GROUP BY and SELECT clauses both contain a
// date-bucketing expression (strftime or a week-start calculation) that is chosen
// at runtime based on the bucket parameter. sqlc cannot parameterise SQL fragments
// in SELECT or GROUP BY — only scalar values can be bound as query parameters.
func (s *Store) Trend(ctx context.Context, f engagement.MetricsFilter, metric, bucket string) (engagement.TrendData, error) {
	bucketExpr := trendBucketSQL(bucket)
	conds, args := timeConditions(f, "e.occurred_at")

	query := /* #nosec G202 -- bucketExpr is a hard-coded string from trendBucketSQL, conds uses only ? placeholders */ `
SELECT
    ` + bucketExpr + ` AS bucket_start,
    COUNT(CASE          WHEN e.event_type = 'delivered' THEN 1               END) AS delivered,
    COUNT(DISTINCT CASE WHEN e.event_type = 'opened'    THEN e.subscriber_id END) AS unique_opens,
    COUNT(CASE          WHEN e.event_type = 'opened'    THEN 1               END) AS total_opens,
    COUNT(DISTINCT CASE WHEN e.event_type = 'clicked'   THEN e.subscriber_id END) AS unique_clicks,
    COUNT(CASE          WHEN e.event_type = 'clicked'   THEN 1               END) AS total_clicks
FROM email_events e
WHERE e.issue_id IS NOT NULL` + conds + `
GROUP BY bucket_start
ORDER BY bucket_start ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return engagement.TrendData{}, err
	}
	defer rows.Close()

	byDate := make(map[string]trendBucket)
	for rows.Next() {
		var tb trendBucket
		if err = rows.Scan(&tb.bucketStart, &tb.delivered, &tb.uniqueOpens, &tb.totalOpens, &tb.uniqueClicks, &tb.totalClicks); err != nil {
			return engagement.TrendData{}, err
		}
		byDate[tb.bucketStart] = tb
	}
	if err = rows.Err(); err != nil {
		return engagement.TrendData{}, err
	}

	return engagement.TrendData{
		Metric: metric,
		Bucket: bucket,
		Points: buildTrendPoints(f, bucket, byDate, metric),
	}, nil
}

// IssueTrend returns a zero-filled time-series for a single issue.
//
// It mirrors Trend but scopes the aggregation to one issue via an issue_id
// predicate. Raw SQL is required for the same reason as Trend: the bucketing
// expression in SELECT and GROUP BY is chosen at runtime and cannot be bound
// as a sqlc parameter.
func (s *Store) IssueTrend(ctx context.Context, issueID int64, f engagement.MetricsFilter, metric, bucket string) (engagement.TrendData, error) {
	bucketExpr := trendBucketSQL(bucket)
	conds, timeArgs := timeConditions(f, "e.occurred_at")

	// issueID binds the first ? in the WHERE clause, so it must precede the
	// time-condition args.
	args := append([]any{issueID}, timeArgs...)

	query := /* #nosec G202 -- bucketExpr is a hard-coded string from trendBucketSQL, conds uses only ? placeholders */ `
SELECT
    ` + bucketExpr + ` AS bucket_start,
    COUNT(CASE          WHEN e.event_type = 'delivered' THEN 1               END) AS delivered,
    COUNT(DISTINCT CASE WHEN e.event_type = 'opened'    THEN e.subscriber_id END) AS unique_opens,
    COUNT(CASE          WHEN e.event_type = 'opened'    THEN 1               END) AS total_opens,
    COUNT(DISTINCT CASE WHEN e.event_type = 'clicked'   THEN e.subscriber_id END) AS unique_clicks,
    COUNT(CASE          WHEN e.event_type = 'clicked'   THEN 1               END) AS total_clicks
FROM email_events e
WHERE e.issue_id = ?` + conds + `
GROUP BY bucket_start
ORDER BY bucket_start ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return engagement.TrendData{}, err
	}
	defer rows.Close()

	byDate := make(map[string]trendBucket)
	for rows.Next() {
		var tb trendBucket
		if err = rows.Scan(&tb.bucketStart, &tb.delivered, &tb.uniqueOpens, &tb.totalOpens, &tb.uniqueClicks, &tb.totalClicks); err != nil {
			return engagement.TrendData{}, err
		}
		byDate[tb.bucketStart] = tb
	}
	if err = rows.Err(); err != nil {
		return engagement.TrendData{}, err
	}

	return engagement.TrendData{
		Metric: metric,
		Bucket: bucket,
		Points: buildTrendPoints(f, bucket, byDate, metric),
	}, nil
}

// subscriberBucket holds raw subscriber event counts for a single time bucket.
type subscriberBucket struct {
	bucketStart  string
	new          int64
	confirmed    int64
	unsubscribed int64
	lost         int64
}

// SubscriberGrowth returns subscriber growth and churn bucketed over time.
//
// Raw SQL is required for two reasons:
//  1. The bucket expression (day/week/month) is a runtime-chosen SQL fragment used
//     in both SELECT and GROUP BY — sqlc cannot parameterise SQL expressions.
//  2. The query uses UNION ALL across five distinct timestamp columns
//     (created_at, confirmed_at, unsubscribed_at, bounced_at, suppressed_at), each
//     with its own WHERE filter. sqlc has no mechanism to repeat a SQL fragment
//     across UNION branches at runtime.
func (s *Store) SubscriberGrowth(ctx context.Context, f engagement.MetricsFilter, bucket string) (engagement.SubscriberData, error) {
	bucketExpr := subsBucketExpr(bucket, "event_time")

	var timeParts []string
	var singleArgs []any
	if f.From != nil {
		timeParts = append(timeParts, "event_time >= ?")
		singleArgs = append(singleArgs, f.From.Format(time.RFC3339))
	}
	if f.To != nil {
		timeParts = append(timeParts, "event_time < ?")
		singleArgs = append(singleArgs, f.To.Format(time.RFC3339))
	}

	outerWhere := ""
	if len(timeParts) > 0 {
		outerWhere = "AND " + strings.Join(timeParts, " AND ")
	}

	// outerWhere references event_time, which is only a valid column name on the
	// derived-table alias ("events"), not on the subscribers table itself. Filter
	// in the outer WHERE so the reference is always valid.
	query := fmt.Sprintf( /* #nosec G201 -- bucketExpr is from subsBucketExpr (hard-coded), outerWhere uses only ? placeholders */
		`
SELECT
    %s                                                                      AS bucket_start,
    SUM(CASE WHEN event_type = 'new'          THEN 1 ELSE 0 END)           AS new,
    SUM(CASE WHEN event_type = 'confirmed'    THEN 1 ELSE 0 END)           AS confirmed,
    SUM(CASE WHEN event_type = 'unsubscribed' THEN 1 ELSE 0 END)           AS unsubscribed,
    SUM(CASE WHEN event_type = 'lost'         THEN 1 ELSE 0 END)           AS lost
FROM (
    SELECT created_at AS event_time, 'new' AS event_type
      FROM subscribers
    UNION ALL
    SELECT confirmed_at, 'confirmed'
      FROM subscribers WHERE confirmed_at IS NOT NULL
    UNION ALL
    SELECT unsubscribed_at, 'unsubscribed'
      FROM subscribers WHERE unsubscribed_at IS NOT NULL
    UNION ALL
    SELECT bounced_at, 'lost'
      FROM subscribers WHERE bounced_at IS NOT NULL
    UNION ALL
    SELECT suppressed_at, 'lost'
      FROM subscribers WHERE suppressed_at IS NOT NULL
) events
WHERE 1=1 %s
GROUP BY bucket_start
ORDER BY bucket_start ASC`,
		bucketExpr,
		outerWhere,
	)

	rows, err := s.db.QueryContext(ctx, query, singleArgs...)
	if err != nil {
		return engagement.SubscriberData{}, err
	}
	defer rows.Close()

	var buckets []subscriberBucket
	for rows.Next() {
		var sb subscriberBucket
		if err = rows.Scan(&sb.bucketStart, &sb.new, &sb.confirmed, &sb.unsubscribed, &sb.lost); err != nil {
			return engagement.SubscriberData{}, err
		}
		buckets = append(buckets, sb)
	}
	if err = rows.Err(); err != nil {
		return engagement.SubscriberData{}, err
	}

	// Seed running total from confirmed-minus-lost before the window.
	var baseline int64
	if f.From != nil {
		baseRow := s.db.QueryRowContext(
			ctx, `
SELECT COUNT(*) FROM subscribers
WHERE confirmed_at IS NOT NULL
  AND confirmed_at < ?
  AND (bounced_at      IS NULL OR bounced_at      >= ?)
  AND (suppressed_at   IS NULL OR suppressed_at   >= ?)
  AND (unsubscribed_at IS NULL OR unsubscribed_at >= ?)`,
			f.From.Format(time.RFC3339),
			f.From.Format(time.RFC3339),
			f.From.Format(time.RFC3339),
			f.From.Format(time.RFC3339),
		)
		_ = baseRow.Scan(&baseline)
	}

	running := baseline
	points := make([]engagement.SubscriberPoint, 0, len(buckets))
	for _, b := range buckets {
		netChange := b.confirmed - b.lost
		running += netChange
		points = append(points, engagement.SubscriberPoint{
			BucketStart:  b.bucketStart,
			New:          b.new,
			Confirmed:    b.confirmed,
			Unsubscribed: b.unsubscribed,
			Lost:         b.lost,
			NetChange:    netChange,
			ActiveAtEnd:  running,
		})
	}

	return engagement.SubscriberData{Bucket: bucket, Points: points}, nil
}

// timeConditions returns an AND fragment and positional args for optional from/to bounds.
//
// Both sides of the comparison are wrapped in SQLite's datetime() so the bound
// (an RFC3339 string) and the stored column compare as normalised UTC instants
// rather than as raw text. This matters because the libsql/Turso driver persists
// a bound time.Time as "2006-01-02 15:04:05.999999999-07:00" (space-separated,
// numeric offset), which does not sort lexically against an RFC3339 "…T…Z"
// bound. Without datetime() the comparison silently drops every row whose date
// equals the bound's date — e.g. a just-sent issue whose events all land on the
// from-boundary day, collapsing its trend to zero.
func timeConditions(f engagement.MetricsFilter, col string) (string, []any) {
	var parts []string
	var args []any
	if f.From != nil {
		parts = append(parts, "datetime("+col+") >= datetime(?)")
		args = append(args, f.From.Format(time.RFC3339))
	}
	if f.To != nil {
		parts = append(parts, "datetime("+col+") < datetime(?)")
		args = append(args, f.To.Format(time.RFC3339))
	}
	if len(parts) == 0 {
		return "", nil
	}
	return "\n  AND " + strings.Join(parts, "\n  AND "), args
}

// trendBucketSQL returns the SQLite expression that maps e.occurred_at to a bucket key.
//
// The hour bucket emits an RFC3339 UTC key (e.g. "2026-06-05T14:00:00Z") so it
// round-trips through the JavaScript Date parser on the dashboard and matches
// bucketKey exactly. SQLite normalises any stored "+HH:MM" offset to UTC before
// formatting, so the key is always UTC regardless of how the driver persisted
// occurred_at.
func trendBucketSQL(bucket string) string {
	switch bucket {
	case "week":
		return `date(e.occurred_at, '-' || CAST(((strftime('%w', e.occurred_at) + 6) % 7) AS TEXT) || ' days')`
	case "hour":
		return `strftime('%Y-%m-%dT%H:00:00Z', e.occurred_at)`
	default:
		return `strftime('%Y-%m-%d', e.occurred_at)`
	}
}

// subsBucketExpr returns the full SQLite date expression for bucketing the given column.
func subsBucketExpr(bucket, col string) string {
	switch bucket {
	case "week":
		return fmt.Sprintf(`date(%s, '-' || CAST(((strftime('%%w', %s) + 6) %% 7) AS TEXT) || ' days')`, col, col)
	case "month":
		return fmt.Sprintf(`strftime('%%Y-%%m-01', %s)`, col)
	default:
		return fmt.Sprintf(`strftime('%%Y-%%m-%%d', %s)`, col)
	}
}

// buildTrendPoints zero-fills missing buckets between f.From and f.To.
func buildTrendPoints(f engagement.MetricsFilter, bucket string, byDate map[string]trendBucket, metric string) []engagement.TrendPoint {
	if f.From == nil || f.To == nil {
		pts := make([]engagement.TrendPoint, 0, len(byDate))
		for _, tb := range byDate {
			pts = append(pts, engagement.TrendPoint{
				BucketStart: tb.bucketStart,
				Value:       trendValue(metric, tb),
				Delivered:   tb.delivered,
			})
		}
		sort.Slice(pts, func(i, j int) bool { return pts[i].BucketStart < pts[j].BucketStart })
		return pts
	}

	var pts []engagement.TrendPoint
	step := bucketStep(bucket)
	for cur := bucketFloor(*f.From, bucket); cur.Before(*f.To); cur = cur.Add(step) {
		key := bucketKey(cur, bucket)
		if len(pts) > 0 && pts[len(pts)-1].BucketStart == key {
			continue
		}
		tb := byDate[key]
		pts = append(pts, engagement.TrendPoint{
			BucketStart: key,
			Value:       trendValue(metric, tb),
			Delivered:   tb.delivered,
		})
	}
	return pts
}

func trendValue(metric string, tb trendBucket) float64 {
	switch metric {
	case "delivered":
		return float64(tb.delivered)
	case "unique_opens":
		return float64(tb.uniqueOpens)
	case "total_opens":
		return float64(tb.totalOpens)
	case "unique_clicks":
		return float64(tb.uniqueClicks)
	case "total_clicks":
		return float64(tb.totalClicks)
	case "open_rate":
		if tb.delivered == 0 {
			return 0
		}
		return float64(tb.uniqueOpens) / float64(tb.delivered)
	default: // click_rate
		if tb.delivered == 0 {
			return 0
		}
		return float64(tb.uniqueClicks) / float64(tb.delivered)
	}
}

func bucketStep(bucket string) time.Duration {
	switch bucket {
	case "week":
		return 7 * 24 * time.Hour
	case "hour":
		return time.Hour
	default:
		return 24 * time.Hour
	}
}

// bucketFloor truncates t to the start of its bucket so the zero-fill loop
// begins on an aligned boundary (the hour for hourly series, the day otherwise).
func bucketFloor(t time.Time, bucket string) time.Time {
	if bucket == "hour" {
		return t.UTC().Truncate(time.Hour)
	}
	return t.UTC().Truncate(24 * time.Hour)
}

func bucketKey(t time.Time, bucket string) string {
	switch bucket {
	case "week":
		wd := int(t.Weekday())
		if wd == 0 {
			wd = 7
		}
		return t.AddDate(0, 0, -(wd - 1)).Format("2006-01-02")
	case "hour":
		return t.UTC().Format("2006-01-02T15") + ":00:00Z"
	default:
		return t.Format("2006-01-02")
	}
}

func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
