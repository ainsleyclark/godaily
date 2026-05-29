// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

const (
	browsePerPage    = 30
	browseSearchMax  = 100
	browseTrendingN  = 5
	digestSendHour   = 6
	digestSendMinute = 30
)

// Browse handles the /browse/ archive page.
func Browse(a *godaily.App) webkit.Handler {
	return func(c *webkit.Context) error {
		props, err := BuildBrowseProps(c.Request.Context(), a.Repository.Issues, a.Repository.Items, c.Request.URL.Query())
		if err != nil {
			return c.RenderWithStatus(http.StatusInternalServerError, pages.Error(http.StatusInternalServerError))
		}
		return c.Render(pages.Browse(props))
	}
}

// BuildBrowseProps assembles the full props for the browse page from the
// given filter query. It is shared by the live handler, the static-site
// generator, and the /api/browse fragment endpoint so every surface renders
// identical markup.
func BuildBrowseProps(ctx context.Context, issues digest.IssueRepository, items news.ItemRepository, query url.Values) (pages.BrowseProps, error) {
	state := parseBrowseQuery(query)
	opts := browseOptions(state)

	list, err := items.List(ctx, opts)
	if err != nil {
		return pages.BrowseProps{}, err
	}

	total, err := items.Count(ctx)
	if err != nil {
		return pages.BrowseProps{}, err
	}

	sourceCounts, err := items.SourceCounts(ctx)
	if err != nil {
		return pages.BrowseProps{}, err
	}

	tagCounts, err := items.TagCounts(ctx)
	if err != nil {
		return pages.BrowseProps{}, err
	}

	matching := matchingCount(ctx, items, opts, list)
	digestPicks := digestPicksCount(ctx, items, opts)
	trending := trendingItems(ctx, items)

	var latestIssue int64
	if recent, err := issues.Latest(ctx, 1); err == nil && len(recent) > 0 {
		latestIssue = recent[0].ID
	}

	totalPages := int64(1)
	if matching > 0 {
		totalPages = (matching + browsePerPage - 1) / browsePerPage
	}

	page := state.Page
	if page < 1 {
		page = 1
	}

	return pages.BrowseProps{
		Items:        list,
		Trending:     trending,
		Total:        total,
		Matching:     matching,
		DigestPicks:  digestPicks,
		SourceCounts: sourceCounts,
		TagCounts:    tagCounts,
		State:        state,
		Page:         page,
		PerPage:      browsePerPage,
		TotalPages:   totalPages,
		NextDigest:   nextDigestIn(time.Now()),
		LatestIssue:  latestIssue,
	}, nil
}

func parseBrowseQuery(q map[string][]string) pages.BrowseFilterState {
	get := func(k string) string {
		if vs, ok := q[k]; ok && len(vs) > 0 {
			return strings.TrimSpace(vs[0])
		}
		return ""
	}

	state := pages.BrowseFilterState{
		Sort:  string(news.ItemSortHot),
		Range: "week",
	}

	if tab := get("tab"); tab != "" && tab != "all" {
		if validSectionTag(tab) {
			state.Tab = tab
		}
	}

	for _, raw := range q["source"] {
		if validSource(raw) && !containsSource(state.Sources, raw) {
			state.Sources = append(state.Sources, raw)
		}
	}

	if qv := get("q"); qv != "" {
		if len(qv) > browseSearchMax {
			qv = qv[:browseSearchMax]
		}
		state.Query = qv
	}

	switch get("sort") {
	case "new":
		state.Sort = string(news.ItemSortNew)
	case "top":
		state.Sort = string(news.ItemSortTop)
	case "hot", "":
		state.Sort = string(news.ItemSortHot)
	}

	switch get("range") {
	case "today", "week", "month", "year", "all":
		state.Range = get("range")
	}

	if get("digest") == "1" {
		state.Digest = true
	}

	if p, err := strconv.ParseInt(get("page"), 10, 64); err == nil && p > 0 {
		state.Page = p
	} else {
		state.Page = 1
	}

	return state
}

func browseOptions(state pages.BrowseFilterState) news.ItemListOptions {
	opts := news.ItemListOptions{
		Sort:    news.ItemSort(state.Sort),
		Search:  state.Query,
		Page:    state.Page,
		PerPage: browsePerPage,
	}
	if state.Tab != "" && state.Tab != "all" {
		opts.Tags = []news.Tag{news.Tag(state.Tab)}
	}
	for _, src := range state.Sources {
		opts.Sources = append(opts.Sources, news.Source(src))
	}
	if state.Digest {
		t := true
		opts.InDigest = &t
	}
	if from := rangeWindow(state.Range, time.Now()); from != nil {
		opts.From = from
	}
	return opts
}

func rangeWindow(r string, now time.Time) *time.Time {
	var from time.Time
	switch r {
	case "today":
		from = now.Add(-24 * time.Hour)
	case "week":
		from = now.AddDate(0, 0, -7)
	case "month":
		from = now.AddDate(0, -1, 0)
	case "year":
		from = now.AddDate(-1, 0, 0)
	default:
		return nil
	}
	return &from
}

// matchingCount returns the total number of items matching the filters (not
// just this page). If the items result is shorter than a full page and we're
// on page 1, we can derive it; otherwise count via a separate page-less
// query.
func matchingCount(ctx context.Context, items news.ItemRepository, opts news.ItemListOptions, page []news.Item) int64 {
	if opts.Page <= 1 && int64(len(page)) < opts.PerPage {
		return int64(len(page))
	}
	countOpts := opts
	countOpts.Page = 0
	countOpts.PerPage = 0
	all, err := items.List(ctx, countOpts)
	if err != nil {
		return int64(len(page))
	}
	return int64(len(all))
}

func digestPicksCount(ctx context.Context, items news.ItemRepository, opts news.ItemListOptions) int64 {
	t := true
	picksOpts := opts
	picksOpts.InDigest = &t
	picksOpts.Page = 0
	picksOpts.PerPage = 0
	picks, err := items.List(ctx, picksOpts)
	if err != nil {
		return 0
	}
	return int64(len(picks))
}

func trendingItems(ctx context.Context, items news.ItemRepository) []news.Item {
	from := time.Now().AddDate(0, 0, -7)
	out, err := items.List(ctx, news.ItemListOptions{
		Sort:    news.ItemSortTop,
		From:    &from,
		Page:    1,
		PerPage: browseTrendingN,
	})
	if err != nil {
		return nil
	}
	return out
}

func nextDigestIn(now time.Time) string {
	target := time.Date(now.Year(), now.Month(), now.Day(), digestSendHour, digestSendMinute, 0, 0, now.Location())
	if !target.After(now) {
		target = target.Add(24 * time.Hour)
	}
	d := target.Sub(now)
	h := int(d.Hours())
	m := int(d.Minutes()) - h*60
	return strconv.Itoa(h) + "h " + strconv.Itoa(m) + "m"
}

func validSectionTag(s string) bool {
	for _, t := range news.SectionTags {
		if string(t) == s {
			return true
		}
	}
	return false
}

func validSource(s string) bool {
	for _, src := range news.Sources {
		if string(src) == s {
			return true
		}
	}
	return false
}

func containsSource(srcs []string, s string) bool {
	for _, x := range srcs {
		if x == s {
			return true
		}
	}
	return false
}
