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
	"database/sql"
	"fmt"
	"os"

	"github.com/ainsleyclark/godaily/internal/db"
	"github.com/urfave/cli/v3"
)

var migrateCmd = &cli.Command{
	Name:  "migrate",
	Usage: "Manage database migrations",
	Commands: []*cli.Command{
		{
			Name:  "up",
			Usage: "Apply pending database migrations",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runMigrate(ctx, db.Up)
			},
		},
		{
			Name:  "down",
			Usage: "Roll back the most recent migration",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runMigrate(ctx, db.Down)
			},
		},
	},
}

// runMigrate opens the configured database and invokes fn. It is shared
// between the `migrate up` and `migrate down` subcommands.
func runMigrate(ctx context.Context, fn func(context.Context, *sql.DB) error) error {
	url := os.Getenv("TURSO_URL")
	if url == "" {
		return fmt.Errorf("TURSO_URL is required for the migrate command")
	}

	conn, err := db.New(ctx, url, os.Getenv("TURSO_AUTH_TOKEN"))
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer conn.Close()

	return fn(ctx, conn)
}
