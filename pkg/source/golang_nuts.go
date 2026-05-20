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

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// GolangNuts defines the type that implements news.Fetcher for the
// golang-nuts mailing list via mail-archive.com RSS 2.0 feed.
// The legacy Google Groups Atom endpoint (/forum/feed/…) returns 404.
type GolangNuts struct {
	url string
}

var _ news.Fetcher = &GolangNuts{}

func init() {
	news.Register(news.SourceGolangNuts, func(cfg env.Config) news.Fetcher { return NewGolangNuts(cfg) })
}

const golangNutsURL = "https://www.mail-archive.com/golang-nuts@googlegroups.com/maillist.xml"

// NewGolangNuts creates a golang-nuts mail-archive.com RSS client.
func NewGolangNuts(_ env.Config) *GolangNuts {
	return &GolangNuts{url: golangNutsURL}
}

// Fetch retrieves the latest threads from the golang-nuts mailing list RSS feed.
func (g GolangNuts) Fetch(ctx context.Context) ([]news.Item, error) {
	feed, err := ingest.Fetch[golangNutsRSS](ctx, g.url, "golang-nuts", xml.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, feed.Channel.Items), nil
}

func (e golangNutsItem) ShouldInclude() bool   { return true }
func (e golangNutsItem) EnrichmentURL() string { return "" }

func (e golangNutsItem) Transform() news.Item {
	published, _ := time.Parse(time.RFC1123, e.PubDate)
	return news.Item{
		Source:    news.SourceGolangNuts,
		Title:     e.title(),
		URL:       e.Link,
		Author:    &news.Author{Name: e.author()},
		Tag:       news.TagDiscussion,
		Score:     news.ScoreOf(news.SourceGolangNuts, news.TagDiscussion, 0, false),
		Published: published.UTC(),
	}
}

// title strips the mailing-list tag prefixes from the subject line.
func (e golangNutsItem) title() string {
	for _, prefix := range []string{"Re: [go-nuts] ", "[go-nuts] "} {
		if strings.HasPrefix(e.Title, prefix) {
			return e.Title[len(prefix):]
		}
	}
	return e.Title
}

// author extracts the author name from the description HTML produced by
// mail-archive.com: <font ...>date</font> -- <a href="...">Author</a>
func (e golangNutsItem) author() string {
	s := e.Description
	end := strings.LastIndex(s, "</a>")
	if end == -1 {
		return ""
	}
	start := strings.LastIndex(s[:end], ">")
	if start == -1 {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}

type (
	golangNutsRSS struct {
		XMLName xml.Name          `xml:"rss"`
		Channel golangNutsChannel `xml:"channel"`
	}
	golangNutsChannel struct {
		Items []golangNutsItem `xml:"item"`
	}
	golangNutsItem struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		PubDate     string `xml:"pubDate"`
	}
)
