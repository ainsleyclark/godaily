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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	godaily "github.com/ainsleyclark/godaily/internal"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/synth"
	"github.com/urfave/cli/v3"
)

func synthCmd(_ *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "synth",
		Usage: "Suggest a tweet and LinkedIn post from a scored news JSON file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "input",
				Usage:    "Path to a scored news JSON file (the output of `godaily run --output`)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Write suggestion to this path (otherwise prints to stdout)",
			},
			&cli.StringFlag{
				Name:  "format",
				Value: "md",
				Usage: "Output format: md or json",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			raw, err := os.ReadFile(cmd.String("input"))
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}

			var sections []news.SourceItems
			if err = json.Unmarshal(raw, &sections); err != nil {
				return fmt.Errorf("parse input: %w", err)
			}

			sug, err := synth.New().Suggest(ctx, time.Now().AddDate(0, 0, -1).Truncate(24*time.Hour), sections)
			if err != nil {
				return err
			}

			var rendered []byte
			switch cmd.String("format") {
			case "json":
				rendered, err = sug.JSON()
				if err != nil {
					return err
				}
			case "md", "":
				rendered = []byte(sug.Markdown())
			default:
				return fmt.Errorf("unknown format %q (want md or json)", cmd.String("format"))
			}

			out := cmd.String("output")
			if out == "" {
				fmt.Println(string(rendered)) //nolint
				return nil
			}
			if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
				return err
			}
			return os.WriteFile(out, rendered, 0o600)
		},
	}
}
