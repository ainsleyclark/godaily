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

// Fetch retrieves the latest articles from the Planet Golang Atom feed.
func (p PlanetGolang) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := ingest.Fetch[planetGolangFeed](ctx, p.url, "planet golang", xml.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, feed.Entries), nil
}

func (e planetGolangEntry) ShouldInclude() bool   { return true }
func (e planetGolangEntry) EnrichmentURL() string { return e.url() }

// url returns the canonical URL by finding the first link with rel="alternate"
// or an empty rel, mirroring the same logic used by the Go Blog source.
func (e planetGolangEntry) url() string {
	for _, l := range e.Links {
		if l.Rel == "alternate" || l.Rel == "" {
			return l.Href
		}
	}
	return ""
}

func (e planetGolangEntry) Transform() news.Item {
	published, _ := time.Parse(time.RFC3339, e.Published)
	return news.Item{
		Source: news.SourcePlanetGolang,
		Title:  e.Title,
		URL:    e.url(),
		Author: &news.Author{
			Name: e.Author.Name,
		},
		Snippet:   e.Summary,
		Tag:       news.TagArticle,
		Score:     news.ScoreOf(news.SourcePlanetGolang, news.TagArticle, 0, false),
		Published: published.UTC(),
	}
}

type (
	planetGolangFeed struct {
		XMLName xml.Name             `xml:"http://www.w3.org/2005/Atom feed"`
		Entries []planetGolangEntry  `xml:"http://www.w3.org/2005/Atom entry"`
	}
	planetGolangEntry struct {
		Title     string               `xml:"http://www.w3.org/2005/Atom title"`
		Links     []planetGolangLink   `xml:"http://www.w3.org/2005/Atom link"`
		Author    planetGolangAuthor   `xml:"http://www.w3.org/2005/Atom author"`
		Published string               `xml:"http://www.w3.org/2005/Atom published"`
		Summary   string               `xml:"http://www.w3.org/2005/Atom summary"`
	}
	planetGolangLink struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	}
	planetGolangAuthor struct {
		Name string `xml:"http://www.w3.org/2005/Atom name"`
	}
)
