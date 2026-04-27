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

	"github.com/ainsleyclark/godaily/internal/news"
)

// GoBlog defines the type that implements news.Fetcher.
type GoBlog struct {
	url string
}

var _ news.Fetcher = &GoBlog{}

func init() {
	news.Register(news.SourceGoBlog, NewGoBlog())
}

const goBlogURL = "https://go.dev/blog/feed.atom"

// NewGoBlog creates a Go Dev Blog client.
func NewGoBlog() *GoBlog {
	return &GoBlog{
		url: goBlogURL,
	}
}

// Fetch retrieves all the news items from the Go Dev Blog Atom feed.
func (g GoBlog) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := fetch[goBlogFeed](ctx, g.url, "go blog", xml.Unmarshal)
	if err != nil {
		return nil, err
	}
	return transformAll(feed.Entries), nil
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

func (e goBlogEntry) shouldInclude() bool { return true }

func (e goBlogEntry) transform() news.Item {
	published, _ := time.Parse(time.RFC3339, e.Published)
	return news.Item{
		Source:    news.SourceGoBlog,
		Title:     e.Title,
		URL:       e.url(),
		Author:    e.Author.Name,
		Snippet:   e.Summary,
		Tag:       news.TagArticle,
		Published: published,
	}
}
