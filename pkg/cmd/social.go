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
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	godaily "github.com/ainsleyclark/godaily/pkg"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	"github.com/ainsleyclark/godaily/pkg/services/social"
)

func socialCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "social",
		Usage: "Publish today's digest to social platforms (Bluesky, LinkedIn, Mastodon).",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Generate posts via the AI but skip platform HTTP and DB writes.",
			},
			&cli.StringSliceFlag{
				Name:  "platform",
				Usage: "Only post to the named platforms (repeatable: bluesky, linkedin, mastodon).",
			},
			&cli.StringFlag{
				Name:  "date",
				Usage: "Date of the digest to post (YYYY-MM-DD). Defaults to today (UTC).",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			date := time.Now().UTC().Truncate(24 * time.Hour)
			if raw := cmd.String("date"); raw != "" {
				d, err := time.Parse("2006-01-02", raw)
				if err != nil {
					return fmt.Errorf("invalid date %q: must be YYYY-MM-DD", raw)
				}
				date = d
			}

			platforms, err := parsePlatforms(cmd.StringSlice("platform"))
			if err != nil {
				return err
			}

			results, err := a.Social.Post(ctx, social.PostOptions{
				Date:      date,
				DryRun:    cmd.Bool("dry-run"),
				Platforms: platforms,
			})
			if err != nil {
				a.Slack.MustSend(ctx, "Social CLI run failed: "+err.Error())
				printResults(results)
				return err
			}

			printResults(results)
			return nil
		},
	}
}

// parsePlatforms validates and converts the --platform flag values into
// the gateway's Platform type.
func parsePlatforms(raw []string) ([]socialgw.Platform, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	known := map[string]socialgw.Platform{
		"bluesky":  socialgw.PlatformBluesky,
		"linkedin": socialgw.PlatformLinkedIn,
		"mastodon": socialgw.PlatformMastodon,
	}

	out := make([]socialgw.Platform, 0, len(raw))
	for _, name := range raw {
		p, ok := known[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			return nil, fmt.Errorf("unknown platform %q (expected one of: bluesky, linkedin, mastodon)", name)
		}
		out = append(out, p)
	}
	return out, nil
}

func printResults(results []social.PostResult) {
	if len(results) == 0 {
		fmt.Fprintln(os.Stdout, "no posts produced")
		return
	}
	for _, r := range results {
		fmt.Fprintln(os.Stdout, "---")
		fmt.Fprintf(os.Stdout, "platform: %s\n", r.Platform)
		switch {
		case r.Err != nil:
			fmt.Fprintf(os.Stdout, "status:   error — %s\n", r.Err)
		case r.Skipped:
			fmt.Fprintln(os.Stdout, "status:   skipped (already posted today)")
		case r.PostURL == "":
			fmt.Fprintln(os.Stdout, "status:   dry-run")
		default:
			fmt.Fprintf(os.Stdout, "status:   posted — %s\n", r.PostURL)
		}
		if r.Text != "" {
			fmt.Fprintln(os.Stdout, "text:")
			fmt.Fprintln(os.Stdout, r.Text)
		}
	}
}
