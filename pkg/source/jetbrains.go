// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/xml"
	"net/http"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// JetBrains defines the type that implements news.Fetcher for the JetBrains
// GoLand WordPress blog feed.
type JetBrains struct {
	url string
}

var _ news.Fetcher = &JetBrains{}

func init() {
	news.Register(news.SourceJetBrains, func(cfg env.Config) news.Fetcher { return NewJetBrains(cfg) })
}

const jetbrainsURL = "https://blog.jetbrains.com/go/feed/"

// NewJetBrains creates a JetBrains GoLand blog RSS client.
func NewJetBrains(_ env.Config) *JetBrains {
	return &JetBrains{url: jetbrainsURL}
}

// Fetch retrieves the latest posts from the JetBrains GoLand blog. JetBrains'
// WordPress can 403 default Go user-agents, so we send our own (mirroring
// ardanlabs).
func (j JetBrains) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := ingest.Fetch[jetbrainsRSS](ctx, j.url, "jetbrains", xml.Unmarshal, http.Header{
		"User-Agent": {"godaily/1.0"},
	})
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, feed.Channel.Items), nil
}

func (i jetbrainsItem) ShouldInclude() bool   { return true }
func (i jetbrainsItem) EnrichmentURL() string { return i.Link }

func (i jetbrainsItem) Transform() news.Item {
	published, _ := time.Parse(time.RFC1123Z, i.PubDate)
	return news.Item{
		Source:    news.SourceJetBrains,
		Title:     i.Title,
		URL:       i.Link,
		Author:    &news.Author{Name: i.Creator},
		Snippet:   i.Description,
		Tag:       news.TagArticle,
		Score:     news.ScoreOf(news.SourceJetBrains, news.TagArticle, 0, false),
		Published: published.UTC(),
	}
}

type (
	jetbrainsRSS struct {
		XMLName xml.Name         `xml:"rss"`
		Channel jetbrainsChannel `xml:"channel"`
	}
	jetbrainsChannel struct {
		Items []jetbrainsItem `xml:"item"`
	}
	jetbrainsItem struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Creator     string `xml:"http://purl.org/dc/elements/1.1/ creator"`
		Description string `xml:"description"`
		PubDate     string `xml:"pubDate"`
	}
)
