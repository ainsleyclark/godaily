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
	"sort"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/email"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/templates"
)

var (
	htmlTmpl = htmltemplate.Must(htmltemplate.New("digest-html").Parse(templates.EmailLayout + templates.EmailHTML))
	textTmpl = texttemplate.Must(texttemplate.New("digest-text").Parse(templates.EmailLayoutText + templates.EmailText))
)

// Mark file paths used in the email HTML. Sources without an entry render the
// ShortLabel text chip instead. Kept private to the email render layer; the
// site-side templ render has its own equivalent in marks.templ.
var emailMarkURLs = map[news.Source]string{
	news.SourceArdanLabs:    "/assets/images/marks/ardanlabs_podcast.svg",
	news.SourceDevTo:        "/assets/images/marks/dev_to.svg",
	news.SourceGitHub:       "/assets/images/marks/github.svg",
	news.SourceGoBlog:       "/assets/images/marks/go_blog.svg",
	news.SourceGoPodcast:    "/assets/images/marks/go_podcast.png",
	news.SourceGolangBridge: "/assets/images/marks/golangbridge.png",
	news.SourceHN:           "/assets/images/marks/hacker_news.svg",
	news.SourceJetBrains:    "/assets/images/marks/goland.svg",
	news.SourceLobsters:     "/assets/images/marks/lobsters.png",
	news.SourceMastodon:     "/assets/images/marks/mastodon.svg",
	news.SourceMedium:       "/assets/images/marks/medium.svg",
	news.SourceReddit:       "/assets/images/marks/reddit.svg",
	news.SourceYouTube:      "/assets/images/marks/youtube.svg",
}

// Per-section accent colours used in the email HTML. Inline styles only —
// most email clients strip <style> blocks, so colour is passed through the
// template rather than driven from a CSS class.
var sectionAccents = map[news.Tag]string{
	news.TagRelease:    "#9333ea",
	news.TagProposal:   "#6366f1",
	news.TagArticle:    "#1a7fa8",
	news.TagDiscussion: "#0d9488",
	news.TagVideo:      "#ec4899",
	news.TagTrending:   "#f59e0b",
}

// Short text chips used as a fallback when a source has no mark image, and
// also shown beside the mark as an accessible label.
var emailShortLabels = map[news.Source]string{
	news.SourceArdanLabs:      "AL",
	news.SourceAwesomeGo:      "AG",
	news.SourceDevTo:          "DEV",
	news.SourceFallthrough:    "FT",
	news.SourceGitHub:         "GH",
	news.SourceGitHubTrending: "GH",
	news.SourceGoBlog:         "go",
	news.SourceGoPodcast:      "GP",
	news.SourceGoRelease:      "go",
	news.SourceGolangBridge:   "GB",
	news.SourceHN:             "HN",
	news.SourceJetBrains:      "JB",
	news.SourceLobsters:       "LO",
	news.SourceMastodon:       "M",
	news.SourceMedium:         "M",
	news.SourceReddit:         "r/",
	news.SourceYouTube:        "YT",
}

type (
	emailItem struct {
		URL            string
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
		Sections       []emailSection
		UnsubscribeURL string
	}
	// renderedDigest carries the rendered email payload so the caller can
	// both ship it via email and persist it to the issues table without
	// re-rendering.
	renderedDigest struct {
		Subject string
		HTML    string
		Text    string
	}
)

func renderDigest(day time.Time, sources []news.SourceItems, unsubscribeURL string) (renderedDigest, error) {
	sections := buildSections(sources)

	data := digestData{Date: day, Sections: sections, UnsubscribeURL: unsubscribeURL}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.ExecuteTemplate(&htmlBuf, "email-layout", data); err != nil {
		return renderedDigest{}, errors.Wrap(err, "rendering html")
	}

	var textBuf bytes.Buffer
	if err := textTmpl.ExecuteTemplate(&textBuf, "email-layout-text", data); err != nil {
		return renderedDigest{}, errors.Wrap(err, "rendering text")
	}

	return renderedDigest{
		Subject: "GoDaily - " + day.Format("January 2, 2006"),
		HTML:    htmlBuf.String(),
		Text:    textBuf.String(),
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
	if item.Score > 0 {
		parts = append(parts, fmt.Sprintf("%.0f pts", item.Score))
	}
	if item.Comments > 0 {
		parts = append(parts, fmt.Sprintf("%d comments", item.Comments))
	}
	return emailItem{
		URL:            item.URL,
		Title:          item.Title,
		Snippet:        item.Snippet,
		Meta:           strings.Join(parts, " · "),
		Source:         string(item.Source),
		SourceNiceName: item.Source.NiceName(),
		SourceLabel:    emailShortLabels[item.Source],
		MarkURL:        emailMarkURLs[item.Source],
	}
}

func (a Aggregator) sendRendered(ctx context.Context, to string, d renderedDigest) error {
	return a.email.Send(ctx, email.SendEmailRequest{
		From:    "noreply@godaily.dev",
		To:      []string{to},
		Subject: d.Subject,
		Html:    d.HTML,
		Text:    d.Text,
	})
}
