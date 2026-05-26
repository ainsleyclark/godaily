// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/services/digest/prompts"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/templates"
)

var (
	suggestHTMLTmpl = htmltemplate.Must(htmltemplate.New("suggest-html").Parse(templates.EmailLayout + templates.SuggestHTML))
	suggestTextTmpl = texttemplate.Must(texttemplate.New("suggest-text").Parse(templates.EmailLayoutText + templates.SuggestText))
)

type (
	// suggestPost is the per-post view model: the prompts.Post fields plus
	// a 1-based number for display.
	suggestPost struct {
		Num        int
		Text       string
		References []prompts.Ref
	}
	// suggestData feeds the shared email layout. The CanonicalURL/share/
	// unsubscribe fields are referenced by the "email-layout" block and
	// stay empty — the synth email is owner-only, so the footer just
	// reads "Sent by GoDaily".
	suggestData struct {
		Date           time.Time
		Posts          []suggestPost
		CanonicalURL   string
		UnsubscribeURL string
		ShareLinkedIn  string
		ShareBluesky   string
		ShareTwitter   string
	}
)

// SendSuggestion generates an AI post suggestion from the stored digest
// items for the given date and emails it to the owner address only.
func (s Service) SendSuggestion(ctx context.Context, date time.Time) error {
	if s.prompter == nil {
		return errors.New("synth send requires ANTHROPIC_API_KEY")
	}
	if s.issues == nil || s.items == nil {
		return errors.New("synth send requires persistence (TURSO_URL not set)")
	}
	if s.adminEmailAddress == "" {
		slog.WarnContext(ctx, "EMAIL_SEND_ADDRESS not set, skipping synth send")
		return nil
	}

	slug := date.Format("2006-01-02")

	issue, err := s.issues.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("no digest found for %s — run `godaily collect` first", slug)
		}
		return errors.Wrap(err, "loading digest")
	}

	sections, err := loadSections(ctx, s.items, issue.ID)
	if err != nil {
		return errors.Wrap(err, "loading items")
	}

	if len(sections) == 0 {
		slog.InfoContext(ctx, "No items for synth suggestion, skipping")
		return nil
	}

	sug, err := prompts.Suggest(ctx, s.prompter, date, sections)
	if err != nil {
		if s.slack != nil {
			s.slack.MustSend(ctx, "AI suggestion failed: "+err.Error())
		}
		return errors.Wrap(err, "synth")
	}

	html, text, err := renderSuggestion(sug)
	if err != nil {
		return err
	}

	return s.email.Send(ctx, email.SendEmailRequest{
		From:    "GoDaily <digest@godaily.dev>",
		To:      []string{s.adminEmailAddress},
		Subject: "GoDaily Synth - " + date.Format("2006-01-02"),
		Html:    html,
		Text:    text,
	})
}

func renderSuggestion(s prompts.Suggestion) (html, text string, err error) {
	data := suggestData{Date: s.Date}
	for i, p := range s.Posts {
		data.Posts = append(data.Posts, suggestPost{
			Num:        i + 1,
			Text:       p.Text,
			References: p.References,
		})
	}

	var htmlBuf bytes.Buffer
	if err = suggestHTMLTmpl.ExecuteTemplate(&htmlBuf, "email-layout", data); err != nil {
		return "", "", errors.Wrap(err, "rendering suggest html")
	}

	var textBuf bytes.Buffer
	if err = suggestTextTmpl.ExecuteTemplate(&textBuf, "email-layout-text", data); err != nil {
		return "", "", errors.Wrap(err, "rendering suggest text")
	}

	return htmlBuf.String(), textBuf.String(), nil
}
