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

func sourcesCmd(_ *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "sources",
		Usage: "Lists registered source names",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			for _, name := range news.Sources {
				fmt.Println(name) //nolint
			}
			return nil
		},
	}
}
