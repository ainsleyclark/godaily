// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/xml"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// Medium defines the type that implements news.Fetcher for Medium's golang tag feed.
type Medium struct {
	url string
}

var _ news.Fetcher = &Medium{}

func init() {
	news.Register(news.SourceMedium, func(cfg env.Config) news.Fetcher { return NewMedium(cfg) })
}

const mediumURL = "https://medium.com/feed/tag/golang"

// NewMedium creates a Medium client targeting the golang tag RSS feed.
func NewMedium(_ env.Config) *Medium {
	return &Medium{url: mediumURL}
}

// Fetch retrieves the latest Go-tagged articles from Medium's RSS feed.
func (m Medium) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := ingest.Fetch[mediumRSS](ctx, m.url, "medium", xml.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, feed.Channel.Items), nil
}

func (i mediumItem) ShouldInclude() bool   { return true }
func (i mediumItem) EnrichmentURL() string { return i.Link }

func (i mediumItem) Transform() news.Item {
	published, _ := time.Parse(time.RFC1123, i.PubDate)
	return news.Item{
		Source: news.SourceMedium,
		Title:  i.Title,
		URL:    i.Link,
		Author: &news.Author{
			Name: i.Creator,
		},
		Snippet:   i.Description,
		Tag:       news.TagArticle,
		Score:     news.ScoreOf(news.SourceMedium, news.TagArticle, 0, false),
		Published: published.UTC(),
	}
}

type mediumRSS struct {
	XMLName xml.Name      `xml:"rss"`
	Channel mediumChannel `xml:"channel"`
}

type mediumChannel struct {
	Items []mediumItem `xml:"item"`
}

type mediumItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Creator     string   `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Description string   `xml:"description"`
	PubDate     string   `xml:"pubDate"`
	Categories  []string `xml:"category"`
}
