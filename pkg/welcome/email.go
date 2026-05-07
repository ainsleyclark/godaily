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

// Package welcome handles sending the one-time welcome email to new subscribers.
package welcome

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	texttemplate "text/template"

	"github.com/ainsleyclark/godaily/pkg/email"
	"github.com/ainsleyclark/godaily/pkg/templates"
)

type sender interface {
	Send(ctx context.Context, req email.SendEmailRequest) error
}

type welcomeData struct {
	LatestIssueURL   string
	LatestIssueTitle string
	UnsubscribeURL   string
}

var (
	htmlTmpl = htmltemplate.Must(htmltemplate.New("welcome-html").Parse(templates.EmailLayout + templates.WelcomeHTML))
	textTmpl = texttemplate.Must(texttemplate.New("welcome-text").Parse(templates.EmailLayoutText + templates.WelcomeText))
)

// Send dispatches a welcome email to a new subscriber via the provided sender.
func Send(ctx context.Context, client sender, to, unsubURL, latestIssueURL, latestIssueTitle string) error {
	data := welcomeData{
		LatestIssueURL:   latestIssueURL,
		LatestIssueTitle: latestIssueTitle,
		UnsubscribeURL:   unsubURL,
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.ExecuteTemplate(&htmlBuf, "email-layout", data); err != nil {
		return fmt.Errorf("rendering welcome html: %w", err)
	}

	var textBuf bytes.Buffer
	if err := textTmpl.ExecuteTemplate(&textBuf, "email-layout-text", data); err != nil {
		return fmt.Errorf("rendering welcome text: %w", err)
	}

	return client.Send(ctx, email.SendEmailRequest{
		From:    "noreply@godaily.dev",
		To:      []string{to},
		Subject: "Welcome to GoDaily!",
		Html:    htmlBuf.String(),
		Text:    textBuf.String(),
	})
}
