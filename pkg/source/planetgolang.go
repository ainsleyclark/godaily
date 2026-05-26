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
	// The feed omits <published> and uses <updated> as the only date field.
	dateStr := e.Published
	if dateStr == "" {
		dateStr = e.Updated
	}
	published, _ := time.Parse(time.RFC3339, dateStr)
	var author *news.Author
	if e.Author.Name != "" {
		author = &news.Author{Name: e.Author.Name}
	}
	return news.Item{
		Source:    news.SourcePlanetGolang,
		Title:     e.Title,
		URL:       e.url(),
		Author:    author,
		Snippet:   e.Summary,
		Tag:       news.TagArticle,
		Score:     news.ScoreOf(news.SourcePlanetGolang, news.TagArticle, 0, false),
		Published: published.UTC(),
	}
}

type (
	planetGolangFeed struct {
		XMLName xml.Name            `xml:"http://www.w3.org/2005/Atom feed"`
		Entries []planetGolangEntry `xml:"http://www.w3.org/2005/Atom entry"`
	}
	planetGolangEntry struct {
		Title     string             `xml:"http://www.w3.org/2005/Atom title"`
		Links     []planetGolangLink `xml:"http://www.w3.org/2005/Atom link"`
		Author    planetGolangAuthor `xml:"http://www.w3.org/2005/Atom author"`
		Published string             `xml:"http://www.w3.org/2005/Atom published"`
		Updated   string             `xml:"http://www.w3.org/2005/Atom updated"`
		Summary   string             `xml:"http://www.w3.org/2005/Atom summary"`
	}
	planetGolangLink struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	}
	planetGolangAuthor struct {
		Name string `xml:"http://www.w3.org/2005/Atom name"`
	}
)
