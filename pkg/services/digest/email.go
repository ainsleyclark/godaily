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

package digest

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"net/url"
	"sort"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/templates"
)

var (
	htmlTmpl = htmltemplate.Must(htmltemplate.New("digest-html").Parse(templates.EmailLayout + templates.EmailHTML))
	textTmpl = texttemplate.Must(texttemplate.New("digest-text").Parse(templates.EmailLayoutText + templates.EmailText))
)

// Per-section accent colours used in the email HTML. Inline styles only —
// most email clients strip <style> blocks, so colour is passed through the
// template rather than driven from a CSS class.
var sectionAccents = map[news.Tag]string{
	news.TagEvent:      "#16a34a",
	news.TagRelease:    "#9333ea",
	news.TagSecurity:   "#dc2626",
	news.TagProposal:   "#6366f1",
	news.TagArticle:    "#1a7fa8",
	news.TagDiscussion: "#0d9488",
	news.TagVideo:      "#ec4899",
	news.TagTrending:   "#f59e0b",
}

type (
	emailItem struct {
		URL            string
		ReadOnURL      string
		Title          string
		Snippet        string
		Meta           string
		Source         string
		SourceNiceName string
		SourceLabel    string
		MarkURL        string
	}
	emailSection struct {
		Tag    string // canonical section tag, e.g. "release"
		Title  string // display heading, e.g. "Releases"
		Accent string // hex colour for the section bar
		Count  int
		Items  []emailItem
	}
	digestData struct {
		Date           time.Time
		Intro          string
		Sections       []emailSection
		UnsubscribeURL string
		CanonicalURL   string
		ShareLinkedIn  string
		ShareBluesky   string
		ShareTwitter   string
	}
	// renderedDigest carries the rendered email payload so the caller can
	// both ship it via email and persist it to the issues table without
	// re-rendering.
	renderedDigest struct {
		Subject        string
		HTML           string
		Text           string
		UnsubscribeURL string
	}

	digestOptions struct {
		Day            time.Time
		Subject        string // AI-generated title; falls back to static date format when empty
		Intro          string // AI-generated intro paragraph; omitted from email when empty
		Sources        []news.SourceItems
		UnsubscribeURL string
		CanonicalURL   string
	}
)

func renderDigest(opts digestOptions) (renderedDigest, error) {
	sections := buildSections(opts.Sources)

	subject := opts.Subject
	if subject == "" {
		subject = "GoDaily - " + opts.Day.Format("January 2, 2006")
	}

	data := digestData{
		Date:           opts.Day,
		Intro:          opts.Intro,
		Sections:       sections,
		UnsubscribeURL: opts.UnsubscribeURL,
		CanonicalURL:   opts.CanonicalURL,
	}
	if opts.CanonicalURL != "" {
		data.ShareLinkedIn = "https://www.linkedin.com/sharing/share-offsite/?url=" + url.QueryEscape(opts.CanonicalURL)
		data.ShareBluesky = "https://bsky.app/intent/compose?text=" + url.QueryEscape(subject+" "+opts.CanonicalURL)
		data.ShareTwitter = "https://twitter.com/intent/tweet?url=" + url.QueryEscape(opts.CanonicalURL) + "&text=" + url.QueryEscape(subject)
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.ExecuteTemplate(&htmlBuf, "email-layout", data); err != nil {
		return renderedDigest{}, errors.Wrap(err, "rendering html")
	}

	var textBuf bytes.Buffer
	if err := textTmpl.ExecuteTemplate(&textBuf, "email-layout-text", data); err != nil {
		return renderedDigest{}, errors.Wrap(err, "rendering text")
	}

	return renderedDigest{
		Subject:        subject,
		HTML:           htmlBuf.String(),
		Text:           textBuf.String(),
		UnsubscribeURL: opts.UnsubscribeURL,
	}, nil
}

// buildSections flattens the per-source items and re-groups them by section
// tag (item.Tag.Section()). Sections are emitted in news.SectionTags order;
// empty sections are skipped. Items within a section are sorted by score
// descending so the strongest signal across all sources lands at the top.
func buildSections(sources []news.SourceItems) []emailSection {
	bucket := map[news.Tag][]news.Item{}
	for _, si := range sources {
		for _, item := range si.Items {
			if item.Source == "" {
				item.Source = si.Source
			}
			section := item.Tag.Section()
			bucket[section] = append(bucket[section], item)
		}
	}

	sections := make([]emailSection, 0, len(news.SectionTags))
	for _, tag := range news.SectionTags {
		items := bucket[tag]
		if len(items) == 0 {
			continue
		}
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].Score > items[j].Score
		})
		if limit := news.SectionLimits[tag]; limit > 0 && len(items) > limit {
			items = items[:limit]
		}
		sec := emailSection{
			Tag:    string(tag),
			Title:  tag.Title(),
			Accent: sectionAccents[tag],
			Count:  len(items),
		}
		for _, item := range items {
			sec.Items = append(sec.Items, toEmailItem(item))
		}
		sections = append(sections, sec)
	}
	return sections
}

func toEmailItem(item news.Item) emailItem {
	parts := []string{item.Source.NiceName()}
	if item.Comments > 0 {
		parts = append(parts, fmt.Sprintf("%d comments", item.Comments))
	}
	markURL := item.Source.MarkURL()
	if markURL != "" {
		markURL = env.AppURL + markURL
	}
	readOnURL := item.URL
	if item.OriginalURL != "" {
		readOnURL = item.OriginalURL
	}
	return emailItem{
		URL:            item.URL,
		ReadOnURL:      readOnURL,
		Title:          item.Title,
		Snippet:        item.Snippet,
		Meta:           strings.Join(parts, " · "),
		Source:         string(item.Source),
		SourceNiceName: item.Source.NiceName(),
		SourceLabel:    item.Source.ShortLabel(),
		MarkURL:        markURL,
	}
}

// buildEmailRequest constructs the outbound email payload for a single
// rendered digest. tags are attached so webhook events can be correlated
// back to the issue and subscriber they belong to.
func buildEmailRequest(to string, d renderedDigest, tags []email.Tag) *email.SendEmailRequest {
	req := &email.SendEmailRequest{
		From:    "GoDaily <digest@godaily.dev>",
		To:      []string{to},
		Subject: d.Subject,
		Html:    d.HTML,
		Text:    d.Text,
		Tags:    tags,
	}
	if d.UnsubscribeURL != "" {
		req.Headers = map[string]string{
			"List-Unsubscribe":      "<" + d.UnsubscribeURL + ">",
			"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
		}
	}
	return req
}

// sendRendered ships a rendered digest to a single recipient. Used by the
// admin preview path where batching is not needed.
func (a Aggregator) sendRendered(ctx context.Context, to string, d renderedDigest, tags []email.Tag) error {
	return a.email.Send(ctx, *buildEmailRequest(to, d, tags))
}
