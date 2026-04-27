package cron

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	htmltemplate "html/template"
	"log/slog"
	texttemplate "text/template"
	"time"

	"github.com/ainsleyclark/godaily/internal/email"
	"github.com/ainsleyclark/godaily/internal/news"
)

//go:embed email.html
var emailHTML string

//go:embed email.txt
var emailText string

var htmlTmpl = htmltemplate.Must(htmltemplate.New("digest").Parse(emailHTML))
var textTmpl = texttemplate.Must(texttemplate.New("digest").Parse(emailText))

type (
	digestData struct {
		Date     string
		Sections []sectionData
	}
	sectionData struct {
		Source string
		Items  []itemData
	}
	itemData struct {
		Title     string
		URL       string
		Author    string
		Published string
	}
)

func (a Aggregator) sendDigest(ctx context.Context, items map[news.Source][]news.Item) error {
	yesterday := time.Now().AddDate(0, 0, -1)
	date := yesterday.Format("January 2, 2006")

	data := digestData{Date: date}
	for _, source := range news.Sources {
		its, ok := items[source]
		if !ok || len(its) == 0 {
			continue
		}
		section := sectionData{Source: source.NiceName()}
		for _, item := range its {
			section.Items = append(section.Items, itemData{
				Title:     item.Title,
				URL:       item.URL,
				Author:    item.Author,
				Published: item.Published.Format("Jan 2"),
			})
		}
		data.Sections = append(data.Sections, section)
	}

	if len(data.Sections) == 0 {
		slog.InfoContext(ctx, "no items to send in digest")
		return nil
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return fmt.Errorf("rendering html: %w", err)
	}

	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return fmt.Errorf("rendering text: %w", err)
	}

	return a.email.Send(ctx, email.SendEmailRequest{
		From:    "noreply@mail.ainsley.dev",
		To:      []string{a.sendToAddress},
		Subject: "GoDaily - " + date,
		Html:    htmlBuf.String(),
		Text:    textBuf.String(),
	})
}
