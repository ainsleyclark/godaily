// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"context"
	"net/http"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/web/views/components"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/ainsleydev/webkit/pkg/webkit"
	"golang.org/x/sync/errgroup"
)

// homeFeedItems is the number of stories shown in the homepage live-feed
// section.
const homeFeedItems = 6

// Home handles the GoDaily homepage.
func Home(a *godaily.App) webkit.Handler {
	return func(c *webkit.Context) error {
		ctx := c.Request.Context()

		recent, err := a.Repository.Issues.Latest(ctx, 4)
		if err != nil {
			return c.RenderWithStatus(http.StatusInternalServerError, pages.Error(http.StatusInternalServerError))
		}

		var issue digest.Issue
		if len(recent) > 0 {
			issue = recent[0]
		}

		var flash string
		if c.Request.URL.Query().Get("confirmed") != "" {
			flash = "You're confirmed! Digest arrives weekday mornings."
		}

		// The feed is a best-effort extra; a failure renders the empty state
		// rather than taking down the homepage.
		feed, err := BuildHomeFeed(ctx, a.Repository.Items)
		if err != nil {
			feed = components.HomeFeedProps{SourceCount: len(news.Sources), SectionCount: len(news.SectionTags)}
		}

		return c.Render(pages.Home(pages.HomeData{
			LatestIssue:  issue,
			SampleIssue:  issue,
			RecentIssues: recent,
			Flash:        flash,
			Feed:         feed,
		}))
	}
}

// BuildHomeFeed assembles the homepage live-feed section: the hottest stories
// from the last week plus the archive totals. Shared by the live handler and
// the static-site generator so both render identical markup.
func BuildHomeFeed(ctx context.Context, items news.ItemRepository) (components.HomeFeedProps, error) {
	var (
		list  []news.Item
		total int64
	)

	from := time.Now().AddDate(0, 0, -7)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		list, err = items.List(gctx, news.ItemListOptions{
			Sort:    news.ItemSortHot,
			From:    &from,
			Page:    1,
			PerPage: homeFeedItems,
		})
		return err
	})
	g.Go(func() (err error) {
		total, err = items.Count(gctx)
		return err
	})
	if err := g.Wait(); err != nil {
		return components.HomeFeedProps{}, err
	}

	return components.HomeFeedProps{
		Items:        list,
		Total:        total,
		SourceCount:  len(news.Sources),
		SectionCount: len(news.SectionTags),
	}, nil
}
