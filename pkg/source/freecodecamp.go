// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/xml"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// FreeCodeCamp fetches Go-tagged tutorials from freeCodeCamp's Ghost RSS feed.
// freeCodeCamp does not publish a public REST API, but Ghost exposes a stable
// RSS endpoint per tag, so we scope to /tag/go to avoid the firehose of JS,
// Python, and React posts.
type FreeCodeCamp struct {
	url string
}

var _ news.Fetcher = &FreeCodeCamp{}

func init() {
	news.Register(news.SourceFreeCodeCamp, func(cfg env.Config) news.Fetcher { return NewFreeCodeCamp(cfg) })
}

const freeCodeCampURL = "https://www.freecodecamp.org/news/tag/go/rss/"

// NewFreeCodeCamp creates a freeCodeCamp RSS client scoped to Go-tagged posts.
func NewFreeCodeCamp(_ env.Config) *FreeCodeCamp {
	return &FreeCodeCamp{url: freeCodeCampURL}
}

// Fetch retrieves Go-tagged tutorials from the freeCodeCamp RSS feed.
func (f FreeCodeCamp) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := ingest.Fetch[fccFeed](ctx, f.url, "freecodecamp", xml.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, feed.Channel.Items), nil
}

type (
	fccFeed struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Items []fccItem `xml:"item"`
		} `xml:"channel"`
	}
	fccItem struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Creator     string `xml:"http://purl.org/dc/elements/1.1/ creator"`
		PubDate     string `xml:"pubDate"`
		Description string `xml:"description"`
	}
)

func (i fccItem) ShouldInclude() bool   { return strings.TrimSpace(i.Link) != "" }
func (i fccItem) EnrichmentURL() string { return i.Link }

func (i fccItem) Transform() news.Item {
	pub, _ := time.Parse(time.RFC1123, i.PubDate)
	if pub.IsZero() {
		pub, _ = time.Parse(time.RFC1123Z, i.PubDate)
	}
	return news.Item{
		Source:    news.SourceFreeCodeCamp,
		Title:     strings.TrimSpace(i.Title),
		URL:       i.Link,
		Author:    &news.Author{Name: strings.TrimSpace(i.Creator)},
		Snippet:   i.Description,
		Tag:       news.TagTutorial,
		Score:     news.ScoreOf(news.SourceFreeCodeCamp, news.TagTutorial, 0, false),
		Published: pub.UTC(),
	}
}
