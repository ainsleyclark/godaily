// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/urfave/cli/v3"
)

func fetchCmd(_ *godaily.App) *cli.Command {
	return &cli.Command{
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

			fmt.Println(string(prettyJSON(items))) //nolint
			return nil
		},
	}
}
