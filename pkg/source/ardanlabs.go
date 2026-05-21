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
	"net/http"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// ArdanLabs fetches episodes from the Ardan Labs Podcast (Buzzsprout).
type ArdanLabs struct {
	url string
}

var _ news.Fetcher = &ArdanLabs{}

func init() {
	news.Register(news.SourceArdanLabs, func(cfg env.Config) news.Fetcher { return NewArdanLabs(cfg) })
}

const ardanLabsURL = "https://feeds.buzzsprout.com/1466944.rss"

// NewArdanLabs creates an Ardan Labs Podcast RSS client.
func NewArdanLabs(_ env.Config) *ArdanLabs {
	return &ArdanLabs{url: ardanLabsURL}
}

// Fetch retrieves the latest episodes from the Ardan Labs Podcast RSS feed.
// Buzzsprout omits per-item <link> and <itunes:image>, so each episode inherits
// the channel artwork and derives its public URL from the enclosure URL.
// Buzzsprout returns 403 to default Go user-agents, so we send our own.
func (a ArdanLabs) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := ingest.Fetch[ardanLabsFeed](ctx, a.url, "ardan labs", xml.Unmarshal, http.Header{
		"User-Agent": {"godaily/1.0"},
	})
	if err != nil {
		return nil, err
	}
	channelImage := feed.Channel.Image.Href
	for i := range feed.Channel.Items {
		feed.Channel.Items[i].channelImage = channelImage
	}
	return ingest.TransformAll(ctx, feed.Channel.Items), nil
}

func (e ardanLabsEpisode) ShouldInclude() bool   { return e.EpisodeType != "trailer" }
func (e ardanLabsEpisode) EnrichmentURL() string { return "" }

func (e ardanLabsEpisode) Transform() news.Item {
	pub, _ := time.Parse(time.RFC1123Z, e.PubDate)
	snippet := e.Summary
	if snippet == "" {
		snippet = e.Description
	}
	return news.Item{
		Source: news.SourceArdanLabs,
		Title:  strings.TrimSpace(e.Title),
		URL:    buzzsproutEpisodeURL(e.Enclosure.URL),
		Author: &news.Author{
			Name: e.Author,
		},
		ImageURL:  e.channelImage,
		Snippet:   snippet,
		Tag:       news.TagPodcast,
		Score:     news.ScoreOf(news.SourceArdanLabs, news.TagPodcast, 0, false),
		Published: pub.UTC(),
	}
}

// buzzsproutEpisodeURL converts the .mp3 enclosure URL into the user-facing
// episode page URL by stripping the trailing extension. Buzzsprout serves the
// HTML page at the same path without the extension.
func buzzsproutEpisodeURL(enclosure string) string {
	return strings.TrimSuffix(enclosure, ".mp3")
}

type (
	ardanLabsFeed struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Image struct {
				Href string `xml:"href,attr"`
			} `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
			Items []ardanLabsEpisode `xml:"item"`
		} `xml:"channel"`
	}
	ardanLabsEpisode struct {
		Title       string `xml:"title"`
		Summary     string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd summary"`
		Description string `xml:"description"`
		Author      string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd author"`
		PubDate     string `xml:"pubDate"`
		EpisodeType string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd episodeType"`
		Enclosure   struct {
			URL string `xml:"url,attr"`
		} `xml:"enclosure"`

		// channelImage is populated post-parse by Fetch since Buzzsprout only
		// emits artwork at the channel level, not per item.
		channelImage string `xml:"-"`
	}
)
