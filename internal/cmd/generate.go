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

	godaily "github.com/ainsleyclark/godaily/internal"
	"github.com/ainsleyclark/godaily/web/generate"
	"github.com/urfave/cli/v3"
)

func generateCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Generate static HTML files into out/ for Vercel deployment.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "out",
				Usage: "Output directory for static files",
				Value: "out",
			},
			&cli.StringFlag{
				Name:  "assets",
				Usage: "Path to compiled frontend assets",
				Value: "web/dist",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return generate.Site(ctx, a.Repository.Issues, cmd.String("out"), cmd.String("assets"))
		},
	}
}
