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

// GoBlog defines the type that implements news.Fetcher.
type GoBlog struct {
	url string
}

var _ news.Fetcher = &GoBlog{}

func init() {
	news.Register(news.SourceGoBlog, func(cfg env.Config) news.Fetcher { return NewGoBlog(cfg) })
}

const goBlogURL = "https://go.dev/blog/feed.atom"

// NewGoBlog creates a Go Dev Blog client.
func NewGoBlog(_ env.Config) *GoBlog {
	return &GoBlog{
		url: goBlogURL,
	}
}

// Fetch retrieves all the news items from the Go Dev Blog Atom feed.
func (g GoBlog) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := ingest.Fetch[goBlogFeed](ctx, g.url, "go blog", xml.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, feed.Entries), nil
}

type (
	goBlogFeed struct {
		XMLName xml.Name      `xml:"http://www.w3.org/2005/Atom feed"`
		Entries []goBlogEntry `xml:"http://www.w3.org/2005/Atom entry"`
	}
	goBlogEntry struct {
		Title     string       `xml:"http://www.w3.org/2005/Atom title"`
		Links     []goBlogLink `xml:"http://www.w3.org/2005/Atom link"`
		Author    goBlogAuthor `xml:"http://www.w3.org/2005/Atom author"`
		Published string       `xml:"http://www.w3.org/2005/Atom published"`
		Summary   string       `xml:"http://www.w3.org/2005/Atom summary"`
	}
	goBlogLink struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	}
	goBlogAuthor struct {
		Name string `xml:"http://www.w3.org/2005/Atom name"`
	}
)

// url returns the canonical URL of the entry by finding the first link with
// rel="alternate" or an empty rel. Returns an empty string if no such link exists.
func (e goBlogEntry) url() string {
	for _, l := range e.Links {
		if l.Rel == "alternate" || l.Rel == "" {
			return l.Href
		}
	}
	return ""
}

func (e goBlogEntry) ShouldInclude() bool   { return true }
func (e goBlogEntry) EnrichmentURL() string { return e.url() }

func (e goBlogEntry) Transform() news.Item {
	published, _ := time.Parse(time.RFC3339, e.Published)
	return news.Item{
		Source: news.SourceGoBlog,
		Title:  e.Title,
		URL:    e.url(),
		Author: &news.Author{
			Name: e.Author.Name,
		},
		Snippet:   e.Summary,
		Tag:       news.TagArticle,
		Score:     news.ScoreOf(news.SourceGoBlog, news.TagArticle, 0, false),
		Published: published.UTC(),
	}
}
