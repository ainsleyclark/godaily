// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/urfave/cli/v3"
)

func buildCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "build",
		Usage: "Build the daily digest issue from collected items.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return a.Service.Digest.Build(ctx, time.Now().UTC())
		},
	}
}
