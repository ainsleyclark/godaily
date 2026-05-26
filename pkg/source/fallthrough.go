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

// Fallthrough fetches episodes from the Fallthrough podcast (Transistor.fm).
type Fallthrough struct {
	url string
}

var _ news.Fetcher = &Fallthrough{}

func init() {
	news.Register(news.SourceFallthrough, func(cfg env.Config) news.Fetcher { return NewFallthrough(cfg) })
}

const fallthroughURL = "https://feeds.transistor.fm/fallthrough"

// NewFallthrough creates a Fallthrough RSS client.
func NewFallthrough(_ env.Config) *Fallthrough {
	return &Fallthrough{url: fallthroughURL}
}

// Fetch retrieves the latest episodes from the Fallthrough RSS feed.
func (f Fallthrough) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := ingest.Fetch[fallthroughFeed](ctx, f.url, "fallthrough", xml.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, feed.Channel.Items), nil
}

func (e fallthroughEpisode) ShouldInclude() bool   { return e.EpisodeType != "trailer" }
func (e fallthroughEpisode) EnrichmentURL() string { return "" }

func (e fallthroughEpisode) Transform() news.Item {
	pub, _ := time.Parse(time.RFC1123Z, e.PubDate)
	snippet := e.Summary
	if snippet == "" {
		snippet = e.Description
	}
	return news.Item{
		Source: news.SourceFallthrough,
		Title:  e.Title,
		URL:    e.Link,
		Author: &news.Author{
			Name: e.Author,
		},
		ImageURL:  e.Image.Href,
		Snippet:   snippet,
		Tag:       news.TagPodcast,
		Score:     news.ScoreOf(news.SourceFallthrough, news.TagPodcast, 0, false),
		Published: pub.UTC(),
	}
}

type (
	fallthroughFeed struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Items []fallthroughEpisode `xml:"item"`
		} `xml:"channel"`
	}
	fallthroughEpisode struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Author      string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd author"`
		Summary     string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd summary"`
		Description string `xml:"description"`
		EpisodeType string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd episodeType"`
		PubDate     string `xml:"pubDate"`
		Image       struct {
			Href string `xml:"href,attr"`
		} `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	}
)
