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
	// Each mail-archive message page carries both the message body and the
	// original Google Groups permalink. Fetch it once per item to pull the
	// snippet (no usable meta description exists) and rewrite the link to the
	// canonical groups.google.com URL, keeping the mail-archive link when the
	// permalink is absent.
	ingest.EnrichFromHTML(ctx, items, enrichFromMessage)
	return items, nil
}

// enrichFromMessage recovers the snippet and canonical Google Groups URL from a
// mail-archive message page's raw HTML.
func enrichFromMessage(rawHTML string) ingest.HTMLEnrichment {
	return ingest.HTMLEnrichment{
		Snippet: extractMHonArcBody(rawHTML),
		URL:     extractGoogleGroupsURL(rawHTML),
	}
}

// ShouldInclude returns false for reply threads (subjects containing "Re: "),
// keeping only original posts from the mailing list.
func (e golangNutsItem) ShouldInclude() bool {
	return !strings.HasPrefix(e.Title, "Re: ") && !strings.HasPrefix(e.Title, "[go-nuts] Re: ")
}

// EnrichmentURL returns "" because the snippet is filled from the message
// body via extractMHonArcBody, not from meta tags.
func (e golangNutsItem) EnrichmentURL() string { return "" }

const (
	mhonArcBodyStart = "<!--X-Body-of-Message-->"
	mhonArcBodyEnd   = "<!--X-Body-of-Message-End-->"
	// mail-archive.com omits the canonical end marker, so we fall back to
	// the next structural block.
	mhonArcBodyEndFallback = `<div class="msgButtons`
)

// Dropping quoted-reply lines keeps the snippet to the author's own words.
var quotedLineRe = regexp.MustCompile(`(?m)^\s*(?:<[^>]+>\s*)*(?:&gt;|>).*$`)

const googleGroupsMsgPrefix = "https://groups.google.com/d/msgid/golang-nuts/"

var (
	// The message footer links the canonical Google Groups permalink directly;
	// it is already correctly percent-encoded, so prefer it verbatim.
	googleGroupsLinkRe = regexp.MustCompile(`https://groups\.google\.com/d/msgid/golang-nuts/[^\s"'<>]+`)
	// The reply form exposes the raw RFC Message-ID as a fallback when the
	// footer link is absent (e.g. older or trimmed pages).
	googleGroupsMsgIDRe = regexp.MustCompile(`name="msgid"\s+value="([^"]+)"`)
)

// extractGoogleGroupsURL recovers the canonical groups.google.com permalink for
// a mail-archive message page, preferring the footer's ready-made link and
// falling back to building one from the reply form's Message-ID. Returns "" when
// neither is present so callers retain the mail-archive link.
func extractGoogleGroupsURL(rawHTML string) string {
	if u := googleGroupsLinkRe.FindString(rawHTML); u != "" {
		return u
	}
	if m := googleGroupsMsgIDRe.FindStringSubmatch(rawHTML); len(m) == 2 {
		if id := strings.TrimSpace(m[1]); id != "" {
			return googleGroupsMsgPrefix + strings.ReplaceAll(id, "@", "%40")
		}
	}
	return ""
}

// sigDelimRe matches the RFC 3676 §4.3 signature delimiter ("-- " on its own
// line). The Google Groups boilerplate ("You received this message because…")
// always follows this delimiter, and it sits inside the MHonArc body region
// — before the msgButtons fallback — so cutting the body here drops both the
// author's signature and the list footer in one shot.
var sigDelimRe = regexp.MustCompile(`(?m)^\s*--\s*$`)

func extractMHonArcBody(rawHTML string) string {
	start := strings.Index(rawHTML, mhonArcBodyStart)
	if start == -1 {
		return ""
	}
	start += len(mhonArcBodyStart)
	rest := rawHTML[start:]

	end := strings.Index(rest, mhonArcBodyEnd)
	if end == -1 {
		end = strings.Index(rest, mhonArcBodyEndFallback)
	}
	if end == -1 {
		return ""
	}
	body := rest[:end]
	if loc := sigDelimRe.FindStringIndex(body); loc != nil {
		body = body[:loc[0]]
	}
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
