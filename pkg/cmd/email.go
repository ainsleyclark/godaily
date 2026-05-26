// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"context"
	htmltemplate "html/template"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/templates"
	"github.com/urfave/cli/v3"
)

var messageTmpl = htmltemplate.Must(htmltemplate.New("message-html").Parse(templates.EmailLayout + templates.MessageHTML))

func emailCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "email",
		Usage: "Send an email to a single recipient via the Resend batch API.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "to",
				Usage:    "Recipient email address.",
				Required: true,
				Value:    "hello@godaily.dev",
			},
			&cli.StringFlag{
				Name:     "body",
				Usage:    "Message body.",
				Required: true,
				Value:    "Test Email",
			},
			&cli.StringFlag{
				Name:  "subject",
				Usage: "Email subject line.",
				Value: "Message from GoDaily",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			to := cmd.String("to")
			body := cmd.String("body")

			var buf bytes.Buffer
			data := struct {
				Body           string
				CanonicalURL   string
				UnsubscribeURL string
				ShareLinkedIn  string
				ShareBluesky   string
				ShareTwitter   string
			}{Body: body}
			if err := messageTmpl.ExecuteTemplate(&buf, "email-layout", data); err != nil {
				return err
			}

			client := email.New(a.Config.ResendToken)
			req := &email.SendEmailRequest{
				From:    "test@godaily.dev",
				To:      []string{to},
				Subject: cmd.String("subject"),
				Text:    body,
				Html:    buf.String(),
			}

			return client.SendBatch(ctx, []*email.SendEmailRequest{req})
		},
	}
}
