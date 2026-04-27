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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ainsleyclark/godaily/internal/news"
	_ "github.com/ainsleyclark/godaily/internal/source"
	"github.com/urfave/cli/v3"
)

var cmd = &cli.Command{
	Name:  "godaily",
	Usage: "Daily Go news, straight to your inbox",
	Commands: []*cli.Command{
		{
			Name:  "run",
			Usage: "Gather all news from sources.",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return nil
			},
		},
		{
			Name:  "sources",
			Usage: "Lists registered source names",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				for _, name := range news.Sources {
					fmt.Println(name) //nolint
				}
				return nil
			},
		},
		{
			Name: "fetch",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "provider",
					Usage: "Provider of source information",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				fetcher, err := news.Get(news.Source(cmd.String("provider")))
				if err != nil {
					return err
				}

				items, err := fetcher.Fetch(ctx)
				if err != nil {
					return err
				}

				indent, err := json.MarshalIndent(items, "", "  ")
				if err != nil {
					return err
				}

				fmt.Println(string(indent)) //nolint
				return nil
			},
		},
	},
}

func main() {
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
