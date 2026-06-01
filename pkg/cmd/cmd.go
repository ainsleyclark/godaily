// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"log/slog"
	"os"

	godaily "github.com/ainsleyclark/godaily/pkg"
	_ "github.com/ainsleyclark/godaily/pkg/source"
	"github.com/urfave/cli/v3"
)

// Run executes the cli command and runs the program.
func Run() {
	ctx := context.Background()

	app, teardown, err := godaily.Bootstrap(ctx)
	defer teardown()
	if err != nil {
		exit(ctx, err)
	}

	cmd := &cli.Command{
		Name:  "godaily",
		Usage: "Daily Go news, straight to your inbox",
		Commands: []*cli.Command{
			buildCmd(app),
			collectCmd(app),
			previewCmd(app),
			sendCmd(app),
			nudgeCmd(app),
			socialCmd(app),
			runCmd(app),
			serveCmd(app),
			sourcesCmd(app),
			synthCmd(app),
			introCmd(app),
			migrateCmd(app),
			fetchCmd(app),
			generateCmd(app),
			emailCmd(app),
		},
	}

	if err = cmd.Run(context.Background(), os.Args); err != nil {
		exit(ctx, err)
	}
}

func exit(ctx context.Context, err error) {
	if err != nil {
		slog.ErrorContext(ctx, err.Error())
		os.Exit(1)
	}
}
