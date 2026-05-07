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
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/email"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/templates"
)

var (
	htmlTmpl = htmltemplate.Must(htmltemplate.New("digest").Parse(templates.EmailHTML))
	textTmpl = texttemplate.Must(texttemplate.New("digest").Parse(templates.EmailText))
)

type (
	emailItem struct {
		URL     string
		Title   string
		Snippet string
		Meta    string
		Rank    int
	}
	emailSection struct {
		Emoji string
		Title string
		Items []emailItem
	}
	digestData struct {
		Date     time.Time
		Sections []emailSection
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

func renderDigest(day time.Time, sources []news.SourceItems) (renderedDigest, error) {
	sections := make([]emailSection, 0, len(sources))
	for _, si := range sources {
		sec := emailSection{
			Emoji: si.Source.Emoji(),
			Title: si.Source.NiceName(),
		}
		for i, item := range si.Items {
			rank := 0
			if si.Source.IsRanked() {
				rank = i + 1
			}
			parts := []string{item.Source.NiceName()}
			if item.Score > 0 {
				parts = append(parts, fmt.Sprintf("%.0f pts", item.Score))
			}
			if item.Comments > 0 {
				parts = append(parts, fmt.Sprintf("%d comments", item.Comments))
			}
			sec.Items = append(sec.Items, emailItem{
				URL:     item.URL,
				Title:   item.Title,
				Snippet: item.Snippet,
				Meta:    strings.Join(parts, " · "),
				Rank:    rank,
			})
		}
		sections = append(sections, sec)
	}

	data := digestData{Date: day, Sections: sections}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return renderedDigest{}, errors.Wrap(err, "rendering html")
	}

	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return renderedDigest{}, errors.Wrap(err, "rendering text")
	}

	return renderedDigest{
		Subject: "GoDaily - " + day.Format("January 2, 2006"),
		HTML:    htmlBuf.String(),
		Text:    textBuf.String(),
	}, nil
}

func (a Aggregator) sendDigest(ctx context.Context, d renderedDigest) error {
	return a.email.Send(ctx, email.SendEmailRequest{
		From:    "noreply@godaily.dev",
		To:      []string{a.adminEmailAddress},
		Subject: d.Subject,
		Html:    d.HTML,
		Text:    d.Text,
	})
}
