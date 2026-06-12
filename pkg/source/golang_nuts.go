// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/xml"
	"regexp"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
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
	items := ingest.TransformAll(ctx, feed.Channel.Items)
	// The feed's <description> carries only the date and author, and the
	// mail-archive message pages expose no meta description, so meta-tag
	// enrichment finds nothing. Pull the snippet from the MHonArc message
	// body instead.
	ingest.EnrichSnippetsFromHTML(ctx, items, extractMHonArcBody)
	return items, nil
}

// ShouldInclude returns false for reply threads (subjects containing "Re: "),
// keeping only original posts from the mailing list.
func (e golangNutsItem) ShouldInclude() bool {
	return !strings.HasPrefix(e.Title, "Re: ") && !strings.HasPrefix(e.Title, "[go-nuts] Re: ")
}

// EnrichmentURL returns "" because mail-archive message pages carry no meta
// description for the generic enricher to read; the snippet is filled from the
// message body in Fetch via extractMHonArcBody instead.
func (e golangNutsItem) EnrichmentURL() string { return "" }

const (
	mhonArcBodyStart = "<!--X-Body-of-Message-->"
	mhonArcBodyEnd   = "<!--X-Body-of-Message-End-->"
)

// quotedLineRe matches MHonArc message lines that are quoted replies — after
// any leading inline tags/whitespace they begin with ">" (rendered as &gt;).
// Dropping them keeps the snippet to the author's own words.
var quotedLineRe = regexp.MustCompile(`(?m)^\s*(?:<[^>]+>\s*)*(?:&gt;|>).*$`)

// extractMHonArcBody returns the raw HTML of the message body that MHonArc
// wraps between its X-Body-of-Message markers, with quoted-reply lines
// removed. ingest.EnrichSnippetsFromHTML sanitises and truncates the result,
// so tags, entities and the mailing list's *markdown*-style emphasis are
// stripped downstream.
func extractMHonArcBody(rawHTML string) string {
	start := strings.Index(rawHTML, mhonArcBodyStart)
	if start == -1 {
		return ""
	}
	start += len(mhonArcBodyStart)
	end := strings.Index(rawHTML[start:], mhonArcBodyEnd)
	if end == -1 {
		return ""
	}
	body := rawHTML[start : start+end]
	return quotedLineRe.ReplaceAllString(body, "")
}

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
