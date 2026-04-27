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

	"github.com/ainsleyclark/godaily/internal/ingest"
	"github.com/ainsleyclark/godaily/internal/news"
)

// Medium defines the type that implements news.Fetcher for Medium's golang tag feed.
type Medium struct {
	url string
}

var _ news.Fetcher = &Medium{}

func init() {
	news.Register(news.SourceMedium, NewMedium())
}

const mediumURL = "https://medium.com/feed/tag/golang"

// NewMedium creates a Medium client targeting the golang tag RSS feed.
func NewMedium() *Medium {
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
		Source:    news.SourceMedium,
		Title:     i.Title,
		URL:       i.Link,
		Author:    i.Creator,
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
