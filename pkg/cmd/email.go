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
