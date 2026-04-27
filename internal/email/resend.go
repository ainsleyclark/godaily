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

package email

import (
	"context"
	"log/slog"

	"github.com/resend/resend-go/v3"
)

type Client struct {
	resend *resend.Client
}

func New() *Client {
	return &Client{
		resend: resend.NewClient("re_xxxxxxxxx"),
	}
}

type SendEmailRequest = resend.SendEmailRequest

func (c Client) Send(ctx context.Context, req SendEmailRequest) error {
	sent, err := c.resend.Emails.Send(&req)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "Successfully sent email", "id", sent.Id, "subject", req.Subject)
	return nil
}

//nolint:unused
//func test() {
//	client := resend.NewClient("re_xxxxxxxxx")
//
//	params := &resend.SendEmailRequest{
//		From:    "Acme <onboarding@resend.dev>",
//		To:      []string{"delivered@resend.dev"},
//		Html:    "<strong>hello world</strong>",
//		Subject: "Hello from Golang",
//		Cc:      []string{"cc@example.com"},
//		Bcc:     []string{"bcc@example.com"},
//		ReplyTo: "replyto@example.com",
//	}
//
//	sent, err := client.Emails.Send(params)
//	if err != nil {
//		fmt.Println(err.Error()) //nolint
//		return
//	}
//	fmt.Println(sent.Id) //nolint
//}
