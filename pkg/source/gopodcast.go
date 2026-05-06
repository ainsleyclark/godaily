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

package source

import (
	"context"
	"encoding/xml"
	"time"

	"github.com/ainsleyclark/godaily/pkg/ingest"
	"github.com/ainsleyclark/godaily/pkg/news"
)

// GoPodcast fetches episodes from go podcast() (Transistor.fm).
type GoPodcast struct {
	url string
}

var _ news.Fetcher = &GoPodcast{}

func init() {
	news.Register(news.SourceGoPodcast, NewGoPodcast())
}

const goPodcastURL = "https://feeds.transistor.fm/go-podcast"

// NewGoPodcast creates a go podcast() RSS client.
func NewGoPodcast() *GoPodcast {
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
