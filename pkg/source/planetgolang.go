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

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// PlanetGolang defines the type that implements news.Fetcher for the Planet Golang aggregator.
type PlanetGolang struct {
	url string
}

var _ news.Fetcher = &PlanetGolang{}

func init() {
	news.Register(news.SourcePlanetGolang, func(cfg env.Config) news.Fetcher { return NewPlanetGolang(cfg) })
}

const planetGolangURL = "https://www.planetgolang.dev/index.xml"

// NewPlanetGolang creates a Planet Golang RSS client.
func NewPlanetGolang(_ env.Config) *PlanetGolang {
	return &PlanetGolang{url: planetGolangURL}
}

// Fetch retrieves the latest articles from the Planet Golang RSS feed.
func (p PlanetGolang) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := ingest.Fetch[planetGolangRSS](ctx, p.url, "planet golang", xml.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, feed.Channel.Items), nil
}

func (i planetGolangItem) ShouldInclude() bool   { return true }
func (i planetGolangItem) EnrichmentURL() string { return i.Link }

func (i planetGolangItem) Transform() news.Item {
	published, _ := time.Parse(time.RFC1123Z, i.PubDate)
	return news.Item{
		Source: news.SourcePlanetGolang,
		Title:  i.Title,
		URL:    i.Link,
		Author: &news.Author{
			Name: i.Creator,
		},
		Snippet:   i.Description,
		Tag:       news.TagArticle,
		Score:     news.ScoreOf(news.SourcePlanetGolang, news.TagArticle, 0, false),
		Published: published.UTC(),
	}
}

type (
	planetGolangRSS struct {
		XMLName xml.Name            `xml:"rss"`
		Channel planetGolangChannel `xml:"channel"`
	}
	planetGolangChannel struct {
		Items []planetGolangItem `xml:"item"`
	}
	planetGolangItem struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Creator     string `xml:"http://purl.org/dc/elements/1.1/ creator"`
		Description string `xml:"description"`
		PubDate     string `xml:"pubDate"`
	}
)
