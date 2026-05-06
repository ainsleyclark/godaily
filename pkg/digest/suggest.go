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
	"log/slog"
	texttemplate "text/template"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/email"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/synth"
	"github.com/ainsleyclark/godaily/pkg/templates"
)

var (
	suggestHTMLTmpl = htmltemplate.Must(htmltemplate.New("suggest").Parse(templates.SuggestHTML))
	suggestTextTmpl = texttemplate.Must(texttemplate.New("suggest").Parse(templates.SuggestText))
)

// SendSuggestion generates an AI post suggestion from the stored digest
// items for the given date and emails it to the owner address only.
func (a Aggregator) SendSuggestion(ctx context.Context, date time.Time) error {
	if a.suggester == nil {
		return errors.New("synth send requires ANTHROPIC_API_KEY")
	}
	if a.issues == nil || a.items == nil {
		return errors.New("synth send requires persistence (TURSO_URL not set)")
	}
	if a.adminEmailAddress == "" {
		slog.WarnContext(ctx, "EMAIL_SEND_ADDRESS not set, skipping synth send")
		return nil
	}

	slug := date.Format("2006-01-02")

	issue, err := a.issues.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("no digest found for %s — run `godaily collect` first", slug)
		}
		return errors.Wrap(err, "loading digest")
	}

	sections, err := loadSections(ctx, a.items, issue.ID)
	if err != nil {
		return errors.Wrap(err, "loading items")
	}

	if len(sections) == 0 {
		slog.InfoContext(ctx, "No items for synth suggestion, skipping")
		return nil
	}

	s, err := a.suggester.Suggest(ctx, date, sections)
	if err != nil {
		return errors.Wrap(err, "synth")
	}
	s.Date = date

	html, text, err := renderSuggestion(s)
	if err != nil {
		return err
	}

	return a.email.Send(ctx, email.SendEmailRequest{
		From:    "noreply@mail.ainsley.dev",
		To:      []string{a.adminEmailAddress},
		Subject: "GoDaily Synth - " + date.Format("2006-01-02"),
		Html:    html,
		Text:    text,
	})
}

func renderSuggestion(s synth.Suggestion) (html, text string, err error) {
	var htmlBuf bytes.Buffer
	if err = suggestHTMLTmpl.Execute(&htmlBuf, s); err != nil {
		return "", "", errors.Wrap(err, "rendering suggest html")
	}

	var textBuf bytes.Buffer
	if err = suggestTextTmpl.Execute(&textBuf, s); err != nil {
		return "", "", errors.Wrap(err, "rendering suggest text")
	}

	return htmlBuf.String(), textBuf.String(), nil
}
