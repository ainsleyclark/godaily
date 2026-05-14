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
	"log/slog"
	"os"
	"path/filepath"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/backup"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v3"
)

func backupCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "backup",
		Usage: "Export the database as a .sql.gz file and upload it to Backblaze B2.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "output",
				Usage: "Write the backup to this local path instead of uploading to B2",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			slog.InfoContext(ctx, "Starting database backup export")

			data, err := backup.Export(ctx, a.DB)
			if err != nil {
				return errors.Wrap(err, "exporting database")
			}

			slog.InfoContext(ctx, "Database exported", "size_bytes", len(data))

			filename := fmt.Sprintf("godaily-backup-%s.sql.gz", time.Now().UTC().Format("2006-01-02"))

			if out := cmd.String("output"); out != "" {
				if err = os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
					return err
				}
				if err = os.WriteFile(out, data, 0o600); err != nil {
					return errors.Wrap(err, "writing backup file")
				}
				slog.InfoContext(ctx, "Backup written to disk", "path", out)
				return nil
			}

			cfg := a.Config
			if cfg.B2KeyID == "" || cfg.B2AppKey == "" || cfg.B2BucketName == "" || cfg.B2Endpoint == "" {
				return errors.New("B2_KEY_ID, B2_APP_KEY, B2_BUCKET_NAME, and B2_ENDPOINT must be set to upload backups")
			}

			if err = backup.Upload(ctx, *cfg, filename, data); err != nil {
				return err
			}

			slog.InfoContext(ctx, "Backup uploaded to B2", "filename", filename, "size_bytes", len(data))
			return nil
		},
	}
}
