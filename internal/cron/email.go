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

type digestData struct {
	Date     time.Time
	Sections []news.SourceItems
}

func (a Aggregator) sendDigest(ctx context.Context, day time.Time, sources []news.SourceItems) error {
	data := digestData{Date: day, Sections: sources}

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
		Subject: "GoDaily - " + day.Format("January 2, 2006"),
		Html:    htmlBuf.String(),
		Text:    textBuf.String(),
	})
}
