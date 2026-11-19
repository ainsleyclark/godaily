// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mcpserver

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

// Enum values mirror the validation in pkg/api/handlers/metrics so the MCP
// and HTTP surfaces accept exactly the same arguments.
var (
	issueSorts = []string{
		"sent_at", "click_rate", "open_rate", "total_clicks",
		"unique_clicks", "total_opens", "unique_opens", "delivered",
	}
	trendMetrics = []string{
		"click_rate", "open_rate", "unique_clicks", "total_clicks",
		"unique_opens", "total_opens", "delivered",
	}
	trendBuckets      = []string{"hour", "day", "week"}
	subscriberBuckets = []string{"day", "week", "month"}
)

const (
	// topLinksLimit is the number of top-clicked links returned per issue.
	topLinksLimit = 10

	// issueTrendWindow is the default span of a single-issue trend, measured
	// from the issue's send time. Email engagement is heavily front-loaded,
	// so hourly buckets over the first two days show the post-send curve.
	issueTrendWindow = 48 * time.Hour

	// defaultSocialLimit is the number of social posts returned by default.
	defaultSocialLimit = 25
)

type (
	// WindowInput is the input for tools that only take a date window.
	WindowInput struct {
		Window
	}
	// RankedInput is the input for tools returning a top-N list over a window.
	RankedInput struct {
		Window
		Limit int `json:"limit,omitempty" jsonschema:"Maximum rows to return, at most 100. Defaults to 10."`
	}
	// IssueEngagementListInput is the input for list_issue_engagement.
	IssueEngagementListInput struct {
		Window
		Limit int    `json:"limit,omitempty" jsonschema:"Maximum issues to return, at most 100. Defaults to 10."`
		Sort  string `json:"sort,omitempty"  jsonschema:"Field to sort by, descending. Defaults to sent_at (newest first)."`
	}
	// IssueEngagementListOutput is the output for list_issue_engagement.
	IssueEngagementListOutput struct {
		Issues []engagement.IssueEngagement `json:"issues"`
	}
	// IssueEngagementInput is the input for get_issue_engagement.
	IssueEngagementInput struct {
		Key string `json:"key" jsonschema:"Issue identifier: a numeric ID, a date slug (YYYY-MM-DD), latest or draft."`
	}
	// IssueEngagementOutput is the output for get_issue_engagement.
	IssueEngagementOutput struct {
		IssueID int64                   `json:"issue_id"`
		Slug    string                  `json:"slug"`
		Subject string                  `json:"subject"`
		SentAt  time.Time               `json:"sent_at"`
		Stats   engagement.IssueStats   `json:"stats"`
		Links   []engagement.LinkClicks `json:"top_links"`
	}
	// TopItemsOutput is the output for top_items.
	TopItemsOutput struct {
		Items []engagement.ItemMetrics `json:"items"`
	}
	// TopTagsOutput is the output for top_tags.
	TopTagsOutput struct {
		Tags []engagement.TagMetrics `json:"tags"`
	}
	// TopSourcesOutput is the output for top_sources.
	TopSourcesOutput struct {
		Sources []engagement.SourceMetrics `json:"sources"`
	}
	// TrendInput is the input for engagement_trend.
	TrendInput struct {
		Window
		Metric string `json:"metric,omitempty" jsonschema:"Metric to chart. Defaults to click_rate."`
		Bucket string `json:"bucket,omitempty" jsonschema:"Bucket size. Defaults to day."`
	}
	// IssueTrendInput is the input for issue_engagement_trend.
	IssueTrendInput struct {
		Window
		Key    string `json:"key"              jsonschema:"Issue identifier: a numeric ID, a date slug (YYYY-MM-DD), latest or draft."`
		Metric string `json:"metric,omitempty" jsonschema:"Metric to chart. Defaults to unique_clicks."`
		Bucket string `json:"bucket,omitempty" jsonschema:"Bucket size. Defaults to hour."`
	}
	// SubscriberGrowthInput is the input for subscriber_growth.
	SubscriberGrowthInput struct {
		Window
		Bucket string `json:"bucket,omitempty" jsonschema:"Bucket size. Defaults to day."`
	}
	// SocialPostsInput is the input for social_post_metrics.
	SocialPostsInput struct {
		Window
		Limit int `json:"limit,omitempty" jsonschema:"Maximum posts to return, newest first, at most 100. Defaults to 25."`
	}
	// SocialPostsOutput is the output for social_post_metrics.
	SocialPostsOutput struct {
		Posts []engagement.SocialPostEngagement `json:"posts"`
		Total int                               `json:"total"`
	}
)

func (t *tools) registerMetrics(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "metrics_summary",
		Description: "Headline email engagement for a window: issues sent, deliveries, unique and total opens and clicks, bounces, complaints, open/click rates and unique engaged subscribers.",
		InputSchema: inputSchema[WindowInput](windowEnums),
		Annotations: readOnly("Engagement summary"),
	}, t.metricsSummary)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_issue_engagement",
		Description: "Per-issue email engagement (deliveries, opens, clicks, rates, bounces) for sent issues in a window. Sort by a metric to find the best or worst performing issues.",
		InputSchema: inputSchema[IssueEngagementListInput](withEnums(windowEnums, map[string][]string{"sort": issueSorts})),
		Annotations: readOnly("Per-issue engagement"),
	}, t.listIssueEngagement)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_issue_engagement",
		Description: "Email engagement stats for one issue plus its top 10 most-clicked links. " + issueKeyDescription,
		Annotations: readOnly("Issue engagement"),
	}, t.getIssueEngagement)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "top_items",
		Description: "The most-clicked news items across all issues in a window, with title, URL, tag and source.",
		InputSchema: inputSchema[RankedInput](windowEnums),
		Annotations: readOnly("Top clicked items"),
	}, t.topItems)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "top_tags",
		Description: "Email link clicks aggregated by item tag in a window, most clicked first.",
		InputSchema: inputSchema[RankedInput](windowEnums),
		Annotations: readOnly("Top tags"),
	}, t.topTags)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "top_sources",
		Description: "Email link clicks aggregated by item source (Hacker News, Reddit, ...) in a window, most clicked first.",
		InputSchema: inputSchema[RankedInput](windowEnums),
		Annotations: readOnly("Top sources"),
	}, t.topSources)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "engagement_trend",
		Description: "Time series of an email engagement metric across all issues, bucketed by hour, day or week.",
		InputSchema: inputSchema[TrendInput](withEnums(windowEnums, map[string][]string{
			"metric": trendMetrics,
			"bucket": trendBuckets,
		})),
		Annotations: readOnly("Engagement trend"),
	}, t.engagementTrend)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "issue_engagement_trend",
		Description: "Time series of an engagement metric for a single issue. Without a window it covers the first 48 hours after the issue was sent, which captures the post-send surge. " + issueKeyDescription,
		InputSchema: inputSchema[IssueTrendInput](withEnums(windowEnums, map[string][]string{
			"metric": trendMetrics,
			"bucket": trendBuckets,
		})),
		Annotations: readOnly("Issue engagement trend"),
	}, t.issueEngagementTrend)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "subscriber_growth",
		Description: "Subscriber growth and churn over time: new, confirmed, unsubscribed and lost subscribers per bucket, net change and active subscribers at the end of each bucket. Aggregate counts only; no email addresses.",
		InputSchema: inputSchema[SubscriberGrowthInput](withEnums(windowEnums, map[string][]string{"bucket": subscriberBuckets})),
		Annotations: readOnly("Subscriber growth"),
	}, t.subscriberGrowth)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "social_post_metrics",
		Description: "GoDaily's social media posts (Bluesky, LinkedIn, Mastodon) in a window with their latest likes, reposts, comments and impressions.",
		InputSchema: inputSchema[SocialPostsInput](windowEnums),
		Annotations: readOnly("Social post metrics"),
	}, t.socialPostMetrics)
}

func (t *tools) metricsSummary(ctx context.Context, _ *mcp.CallToolRequest, in WindowInput) (*mcp.CallToolResult, engagement.SummaryStats, error) {
	from, to, err := in.bounds()
	if err != nil {
		return nil, engagement.SummaryStats{}, err
	}
	stats, err := t.deps.Metrics.Summary(ctx, engagement.MetricsFilter{From: from, To: to})
	if err != nil {
		return nil, engagement.SummaryStats{}, internalError(ctx, "fetch summary stats", err)
	}
	return nil, stats, nil
}

func (t *tools) listIssueEngagement(ctx context.Context, _ *mcp.CallToolRequest, in IssueEngagementListInput) (*mcp.CallToolResult, IssueEngagementListOutput, error) {
	from, to, err := in.bounds()
	if err != nil {
		return nil, IssueEngagementListOutput{}, err
	}
	sort := in.Sort
	if sort == "" {
		sort = "sent_at"
	}
	rows, err := t.deps.Metrics.IssueList(ctx, engagement.MetricsFilter{
		From:  from,
		To:    to,
		Limit: clampLimit(in.Limit, defaultLimit),
	}, sort)
	if err != nil {
		return nil, IssueEngagementListOutput{}, internalError(ctx, "fetch issue metrics", err)
	}
	return nil, IssueEngagementListOutput{Issues: rows}, nil
}

func (t *tools) getIssueEngagement(ctx context.Context, _ *mcp.CallToolRequest, in IssueEngagementInput) (*mcp.CallToolResult, IssueEngagementOutput, error) {
	issue, err := t.resolveIssue(ctx, in.Key)
	if err != nil {
		return nil, IssueEngagementOutput{}, err
	}
	stats, err := t.deps.EmailEvents.IssueStats(ctx, issue.ID)
	if err != nil {
		return nil, IssueEngagementOutput{}, internalError(ctx, "fetch issue stats", err)
	}
	links, err := t.deps.EmailEvents.TopLinks(ctx, issue.ID, topLinksLimit)
	if err != nil {
		return nil, IssueEngagementOutput{}, internalError(ctx, "fetch top links", err)
	}
	return nil, IssueEngagementOutput{
		IssueID: issue.ID,
		Slug:    issue.Slug,
		Subject: issue.Subject,
		SentAt:  issue.SentAt,
		Stats:   stats,
		Links:   links,
	}, nil
}

func (t *tools) topItems(ctx context.Context, _ *mcp.CallToolRequest, in RankedInput) (*mcp.CallToolResult, TopItemsOutput, error) {
	f, err := in.filter()
	if err != nil {
		return nil, TopItemsOutput{}, err
	}
	rows, err := t.deps.Metrics.ItemList(ctx, f)
	if err != nil {
		return nil, TopItemsOutput{}, internalError(ctx, "fetch item metrics", err)
	}
	return nil, TopItemsOutput{Items: rows}, nil
}

func (t *tools) topTags(ctx context.Context, _ *mcp.CallToolRequest, in RankedInput) (*mcp.CallToolResult, TopTagsOutput, error) {
	f, err := in.filter()
	if err != nil {
		return nil, TopTagsOutput{}, err
	}
	rows, err := t.deps.Metrics.TagList(ctx, f)
	if err != nil {
		return nil, TopTagsOutput{}, internalError(ctx, "fetch tag metrics", err)
	}
	return nil, TopTagsOutput{Tags: rows}, nil
}

func (t *tools) topSources(ctx context.Context, _ *mcp.CallToolRequest, in RankedInput) (*mcp.CallToolResult, TopSourcesOutput, error) {
	f, err := in.filter()
	if err != nil {
		return nil, TopSourcesOutput{}, err
	}
	rows, err := t.deps.Metrics.SourceList(ctx, f)
	if err != nil {
		return nil, TopSourcesOutput{}, internalError(ctx, "fetch source metrics", err)
	}
	return nil, TopSourcesOutput{Sources: rows}, nil
}

func (t *tools) engagementTrend(ctx context.Context, _ *mcp.CallToolRequest, in TrendInput) (*mcp.CallToolResult, engagement.TrendData, error) {
	from, to, err := in.bounds()
	if err != nil {
		return nil, engagement.TrendData{}, err
	}
	metric := orDefault(in.Metric, "click_rate")
	bucket := orDefault(in.Bucket, "day")
	data, err := t.deps.Metrics.Trend(ctx, engagement.MetricsFilter{From: from, To: to}, metric, bucket)
	if err != nil {
		return nil, engagement.TrendData{}, internalError(ctx, "fetch trend data", err)
	}
	return nil, data, nil
}

func (t *tools) issueEngagementTrend(ctx context.Context, _ *mcp.CallToolRequest, in IssueTrendInput) (*mcp.CallToolResult, engagement.TrendData, error) {
	from, to, err := in.bounds()
	if err != nil {
		return nil, engagement.TrendData{}, err
	}
	issue, err := t.resolveIssue(ctx, in.Key)
	if err != nil {
		return nil, engagement.TrendData{}, err
	}
	// Match GET /metrics/issues/{slug}/trend: default to the 48 hours after
	// send, clamped to now so the series isn't padded with future buckets.
	if from == nil && to == nil {
		sent := issue.SentAt.UTC().Truncate(time.Hour)
		end := sent.Add(issueTrendWindow)
		if now := time.Now().UTC(); end.After(now) {
			end = now
		}
		from, to = &sent, &end
	}
	metric := orDefault(in.Metric, "unique_clicks")
	bucket := orDefault(in.Bucket, "hour")
	data, err := t.deps.Metrics.IssueTrend(ctx, issue.ID, engagement.MetricsFilter{From: from, To: to}, metric, bucket)
	if err != nil {
		return nil, engagement.TrendData{}, internalError(ctx, "fetch issue trend data", err)
	}
	return nil, data, nil
}

func (t *tools) subscriberGrowth(ctx context.Context, _ *mcp.CallToolRequest, in SubscriberGrowthInput) (*mcp.CallToolResult, engagement.SubscriberData, error) {
	from, to, err := in.bounds()
	if err != nil {
		return nil, engagement.SubscriberData{}, err
	}
	data, err := t.deps.Metrics.SubscriberGrowth(ctx, engagement.MetricsFilter{From: from, To: to}, orDefault(in.Bucket, "day"))
	if err != nil {
		return nil, engagement.SubscriberData{}, internalError(ctx, "fetch subscriber data", err)
	}
	return nil, data, nil
}

func (t *tools) socialPostMetrics(ctx context.Context, _ *mcp.CallToolRequest, in SocialPostsInput) (*mcp.CallToolResult, SocialPostsOutput, error) {
	from, to, err := in.bounds()
	if err != nil {
		return nil, SocialPostsOutput{}, err
	}
	posts, err := t.deps.SocialMetrics.List(ctx, engagement.MetricsFilter{From: from, To: to})
	if err != nil {
		return nil, SocialPostsOutput{}, internalError(ctx, "fetch social metrics", err)
	}
	total := len(posts)
	if limit := clampLimit(in.Limit, defaultSocialLimit); len(posts) > limit {
		posts = posts[:limit]
	}
	return nil, SocialPostsOutput{Posts: posts, Total: total}, nil
}

// filter resolves a RankedInput into a metrics filter.
func (in RankedInput) filter() (engagement.MetricsFilter, error) {
	from, to, err := in.bounds()
	if err != nil {
		return engagement.MetricsFilter{}, err
	}
	return engagement.MetricsFilter{From: from, To: to, Limit: clampLimit(in.Limit, defaultLimit)}, nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
