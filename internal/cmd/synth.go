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
	"context"
	"fmt"
	"time"

	godaily "github.com/ainsleyclark/godaily/internal"
	"github.com/urfave/cli/v3"
)

func synthCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "synth",
		Usage: "Generate an AI post suggestion from the stored digest and email it to the owner.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "date",
				Usage: "Date of the digest to use (YYYY-MM-DD). Defaults to yesterday.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			date := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
			if raw := cmd.String("date"); raw != "" {
				d, err := time.Parse("2006-01-02", raw)
				if err != nil {
					return fmt.Errorf("invalid date %q: must be YYYY-MM-DD", raw)
				}
				date = d
			}
			return a.Aggregator.SendSuggestion(ctx, date)
		},
	}
}
