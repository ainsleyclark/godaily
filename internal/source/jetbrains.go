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
	"time"

	"github.com/ainsleyclark/godaily/internal/ingest"
	"github.com/ainsleyclark/godaily/internal/news"
)

// JetBrains defines the type that implements news.Fetcher for the JetBrains
// GoLand WordPress blog feed.
type JetBrains struct {
	url string
}

var _ news.Fetcher = &JetBrains{}

func init() {
	news.Register(news.SourceJetBrains, NewJetBrains())
}

const jetbrainsURL = "https://blog.jetbrains.com/go/feed/"

// NewJetBrains creates a JetBrains GoLand blog RSS client.
func NewJetBrains() *JetBrains {
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
