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

package synth

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Client struct {
	anthropic anthropic.Client
}

func New() *Client {
	return &Client{
		anthropic: anthropic.NewClient(
			option.WithAPIKey("my-anthropic-api-key"), // Defaults to os.LookupEnv("ANTHROPIC_API_KEY")
		),
	}
}

//func main() {
//	client := anthropic.NewClient(
//		option.WithAPIKey("my-anthropic-api-key"), // defaults to os.LookupEnv("ANTHROPIC_API_KEY")
//	)
//	message, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
//		MaxTokens: 1024,
//		Messages: []anthropic.MessageParam{
//			anthropic.NewUserMessage(anthropic.NewTextBlock("What is a quaternion?")),
//		},
//		Model: anthropic.ModelClaudeOpus4_7,
//	})
//	if err != nil {
//		panic(err.Error())
//	}
//	fmt.Printf("%+v\n", message.Content)
//}
