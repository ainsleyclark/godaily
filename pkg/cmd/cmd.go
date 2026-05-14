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
			collectCmd(app),
			sendCmd(app),
			runCmd(app),
			serveCmd(app),
			sourcesCmd(app),
			synthCmd(app),
			migrateCmd(app),
			fetchCmd(app),
			generateCmd(app),
			backupCmd(app),
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
