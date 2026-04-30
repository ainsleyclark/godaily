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

	"github.com/ainsleyclark/godaily/internal/digest"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/urfave/cli/v3"
)

var collectCmd = &cli.Command{
	Name:  "collect",
	Usage: "Fetch Go news from all sources and store the digest as a draft.",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "dry-run",
			Usage: "Skip persisting the digest; only gather and return raw items",
		},
		&cli.StringFlag{
			Name:  "output",
			Usage: "Write aggregated items as JSON to this path (skipped if empty)",
		},
		&cli.StringSliceFlag{
			Name:  "source",
			Usage: "Only run the named sources (repeatable). Defaults to all.",
		},
		&cli.BoolFlag{
			Name:  "synth",
			Value: true,
			Usage: "Generate suggested social posts via Anthropic and include them in the digest",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		issueStore, itemStore, conn, err := openStores(ctx)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		if conn != nil {
			defer conn.Close()
		}

		var (
			issueRepo news.IssueRepository
			itemRepo  news.ItemRepository
		)
		if issueStore != nil {
			issueRepo = issueStore
			itemRepo = itemStore
		}

		runner, err := digest.New(issueRepo, itemRepo)
		if err != nil {
			return err
		}

		sources, err := parseSources(cmd.StringSlice("source"))
		if err != nil {
			return err
		}

		_, raw, err := runner.Collect(ctx, digest.CollectOptions{
			DryRun:       cmd.Bool("dry-run"),
			Sources:      sources,
			IncludeSynth: cmd.Bool("synth"),
		})
		if err != nil {
			return err
		}

		out := cmd.String("output")
		if out == "" {
			return nil
		}

		indent, err := json.MarshalIndent(raw, "", "\t")
		if err != nil {
			return err
		}

		if err = os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
			return err
		}

		return os.WriteFile(out, indent, 0o600)
	},
}
