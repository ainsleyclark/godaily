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
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/ingest"
	"github.com/ainsleyclark/godaily/pkg/news"
)

// GolangNuts defines the type that implements news.Fetcher for the
// golang-nuts Google Groups mailing list Atom feed.
type GolangNuts struct {
	url string
}

var _ news.Fetcher = &GolangNuts{}

func init() {
	news.Register(news.SourceGolangNuts, NewGolangNuts())
}

const golangNutsURL = "https://groups.google.com/forum/feed/golang-nuts/msgs/atom.xml?num=25"

// NewGolangNuts creates a golang-nuts Google Groups client.
func NewGolangNuts() *GolangNuts {
	return &GolangNuts{url: golangNutsURL}
}

// Fetch retrieves the latest threads from the golang-nuts mailing list Atom feed.
func (g GolangNuts) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := ingest.Fetch[golangNutsFeed](ctx, g.url, "golang-nuts", xml.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, feed.Entries), nil
}

func (e golangNutsEntry) ShouldInclude() bool   { return true }
func (e golangNutsEntry) EnrichmentURL() string { return "" }

func (e golangNutsEntry) Transform() news.Item {
	published, _ := time.Parse(time.RFC3339, e.Updated)
	return news.Item{
		Source:    news.SourceGolangNuts,
		Title:     strings.TrimPrefix(e.Title, "[golang-nuts] "),
		URL:       e.link(),
		Author:    &news.Author{Name: e.Author.Name},
		Snippet:   e.Content,
		Tag:       news.TagDiscussion,
		Score:     news.ScoreOf(news.SourceGolangNuts, news.TagDiscussion, 0, false),
		Published: published.UTC(),
	}
}

// link returns the href of the first alternate or untyped link in the entry.
func (e golangNutsEntry) link() string {
	for _, l := range e.Links {
		if l.Rel == "alternate" || l.Rel == "" {
			return l.Href
		}
	}
	return ""
}

type (
	golangNutsFeed struct {
		XMLName xml.Name          `xml:"http://www.w3.org/2005/Atom feed"`
		Entries []golangNutsEntry `xml:"http://www.w3.org/2005/Atom entry"`
	}
	golangNutsEntry struct {
		Title   string           `xml:"http://www.w3.org/2005/Atom title"`
		Links   []golangNutsLink `xml:"http://www.w3.org/2005/Atom link"`
		Author  golangNutsAuthor `xml:"http://www.w3.org/2005/Atom author"`
		Updated string           `xml:"http://www.w3.org/2005/Atom updated"`
		Content string           `xml:"http://www.w3.org/2005/Atom content"`
	}
	golangNutsLink struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	}
	golangNutsAuthor struct {
		Name string `xml:"http://www.w3.org/2005/Atom name"`
	}
)
