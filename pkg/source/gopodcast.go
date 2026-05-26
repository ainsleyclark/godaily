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

// GoPodcast fetches episodes from go podcast() (Transistor.fm).
type GoPodcast struct {
	url string
}

var _ news.Fetcher = &GoPodcast{}

func init() {
	news.Register(news.SourceGoPodcast, func(cfg env.Config) news.Fetcher { return NewGoPodcast(cfg) })
}

const goPodcastURL = "https://feeds.transistor.fm/go-podcast"

// NewGoPodcast creates a go podcast() RSS client.
func NewGoPodcast(_ env.Config) *GoPodcast {
	return &GoPodcast{url: goPodcastURL}
}

// Fetch retrieves the latest episodes from the go podcast() RSS feed.
func (g GoPodcast) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := ingest.Fetch[goPodcastFeed](ctx, g.url, "go podcast", xml.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, feed.Channel.Items), nil
}

func (e goPodcastEpisode) ShouldInclude() bool   { return e.EpisodeType != "trailer" }
func (e goPodcastEpisode) EnrichmentURL() string { return "" }

func (e goPodcastEpisode) Transform() news.Item {
	pub, _ := time.Parse(time.RFC1123Z, e.PubDate)
	snippet := e.Summary
	if snippet == "" {
		snippet = e.Description
	}
	return news.Item{
		Source: news.SourceGoPodcast,
		Title:  e.Title,
		URL:    e.Link,
		Author: &news.Author{
			Name: e.Author,
		},
		ImageURL:  e.Image.Href,
		Snippet:   snippet,
		Tag:       news.TagPodcast,
		Score:     news.ScoreOf(news.SourceGoPodcast, news.TagPodcast, 0, false),
		Published: pub.UTC(),
	}
}

type (
	goPodcastFeed struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Items []goPodcastEpisode `xml:"item"`
		} `xml:"channel"`
	}
	goPodcastEpisode struct {
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
