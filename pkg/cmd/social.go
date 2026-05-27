// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform/bluesky"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform/linkedin"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform/mastodon"
)

func socialCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "social",
		Usage: "Social media posting commands.",
		Commands: []*cli.Command{
			socialPostCmd(a),
			socialPublishCmd(a),
			socialRotationCmd(a),
		},
	}
}

// socialPostCmd posts a raw string directly to one or more platforms,
// bypassing the AI content-generation pipeline.
func socialPostCmd(app *godaily.App) *cli.Command {
	return &cli.Command{
		Name:      "post",
		Usage:     "Post raw text directly to social platforms (no AI generation).",
		ArgsUsage: "<text>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "platform",
				Usage: "Platforms to post to (bluesky, linkedin, mastodon). Defaults to all configured.",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			text := strings.Join(c.Args().Slice(), " ")
			if text == "" {
				return fmt.Errorf("provide post text as an argument")
			}

			posters, err := postersForFlags(app, c.StringSlice("platform"))
			if err != nil {
				return err
			}
			if len(posters) == 0 {
				return fmt.Errorf("no platforms configured — set credentials in .env or narrow with --platform")
			}

			var anyErr error
			for _, p := range posters {
				slog.InfoContext(ctx, "Posting", "platform", p.Platform(), "chars", len(text))
				res, err := p.Post(ctx, platform.PostRequest{Text: text})
				if err != nil {
					slog.ErrorContext(ctx, "Post failed", "platform", p.Platform(), "error", err)
					anyErr = err
					continue
				}
				slog.InfoContext(ctx, "Posted", "platform", p.Platform(), "url", res.PostURL)
				fmt.Printf("%s: %s\n", p.Platform(), res.PostURL) //nolint
			}
			return anyErr
		},
	}
}

// socialPublishCmd runs the full AI-driven pipeline: picks the day's best
// item, generates per-platform copy, and posts (or dry-runs) to each platform.
func socialPublishCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "publish",
		Usage: "Generate and publish AI-crafted posts for today's digest.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Generate posts via AI but skip platform HTTP and DB writes.",
			},
			&cli.StringSliceFlag{
				Name:  "platform",
				Usage: "Only post to the named platforms (bluesky, linkedin, mastodon).",
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

			results, err := a.Service.Social.Post(ctx, social.PostOptions{
				Date:      date,
				DryRun:    cmd.Bool("dry-run"),
				Platforms: platforms,
			})
			if err != nil {
				a.Slack.MustSend(ctx, "Social publish CLI failed: "+err.Error())
				printResults(results)
				return err
			}

			printResults(results)
			return nil
		},
	}
}

// socialRotationCmd drives the Tue/Fri rotation slot manually. --kind
// forces a specific candidate (skipping the day-aware routing) which is
// the main reason this CLI exists: testing each path end-to-end.
func socialRotationCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "rotation",
		Usage: "Run the Tue/Fri rotation slot (new_source|spotlight|cta|recap).",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Run the full pipeline (eligibility + AI) but skip platform HTTP and DB writes.",
			},
			&cli.StringSliceFlag{
				Name:  "platform",
				Usage: "Only post to the named platforms (bluesky, linkedin, mastodon).",
			},
			&cli.StringFlag{
				Name:  "kind",
				Usage: "Force a specific candidate kind (new_source, spotlight, cta, recap). Bypasses day-of-week routing.",
			},
			&cli.StringFlag{
				Name:  "now",
				Usage: "Override the wall clock (RFC3339). Useful to test Tue/Fri routing on other days.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			platforms, err := parsePlatforms(cmd.StringSlice("platform"))
			if err != nil {
				return err
			}

			now := time.Now().UTC()
			if raw := cmd.String("now"); raw != "" {
				parsed, err := time.Parse(time.RFC3339, raw)
				if err != nil {
					return fmt.Errorf("invalid --now (RFC3339 expected): %w", err)
				}
				now = parsed.UTC()
			}

			results, err := a.Service.Social.Rotate(ctx, social.RotateOptions{
				Now:       now,
				DryRun:    cmd.Bool("dry-run"),
				Platforms: platforms,
				ForceKind: social.PostKind(strings.TrimSpace(cmd.String("kind"))),
			})
			if err != nil {
				a.Slack.MustSend(ctx, "Social rotation CLI failed: "+err.Error())
				printResults(results)
				return err
			}

			printResults(results)
			return nil
		},
	}
}

// postersForFlags returns Poster implementations for the requested platforms,
// or all configured ones when platforms is empty.
func postersForFlags(app *godaily.App, platforms []string) ([]platform.Poster, error) {
	c := app.Config

	all := map[social.Platform]platform.Poster{}
	if c.BlueskyHandle != "" && c.BlueskyAppPassword != "" {
		all[social.Bluesky] = bluesky.New(c.BlueskyHandle, c.BlueskyAppPassword)
	}
	if c.LinkedInOAuthToken != "" && c.LinkedInOrgURN != "" {
		all[social.LinkedIn] = linkedin.New(c.LinkedInOAuthToken, c.LinkedInOrgURN)
	}
	if c.MastodonServer != "" && c.MastodonAppToken != "" {
		all[social.Mastodon] = mastodon.New(c.MastodonServer, c.MastodonAppToken)
	}

	if len(platforms) == 0 {
		out := make([]platform.Poster, 0, len(all))
		for _, p := range all {
			out = append(out, p)
		}
		return out, nil
	}

	out := make([]platform.Poster, 0, len(platforms))
	for _, name := range platforms {
		key := social.Platform(strings.ToLower(strings.TrimSpace(name)))
		p, ok := all[key]
		if !ok {
			return nil, fmt.Errorf("platform %q not configured or unknown (expected: bluesky, linkedin, mastodon)", name)
		}
		out = append(out, p)
	}
	return out, nil
}

// parsePlatforms validates and converts --platform flag values.
func parsePlatforms(raw []string) ([]social.Platform, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	known := map[string]social.Platform{
		"bluesky":  social.Bluesky,
		"linkedin": social.LinkedIn,
		"mastodon": social.Mastodon,
	}
	out := make([]social.Platform, 0, len(raw))
	for _, name := range raw {
		p, ok := known[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			return nil, fmt.Errorf("unknown platform %q (expected: bluesky, linkedin, mastodon)", name)
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
