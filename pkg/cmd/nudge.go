// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"log/slog"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/urfave/cli/v3"
)

func nudgeCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "nudge",
		Usage: "Send a one-time confirmation reminder to unconfirmed sign-ups.",
		Action: func(ctx context.Context, _ *cli.Command) error {
			sent, failed, err := a.Service.Subscribers.SendConfirmationNudges(ctx)
			if err != nil {
				a.Slack.MustSend(ctx, slack.Error("Send confirmation nudges failed", err))
				return err
			}
			slog.InfoContext(ctx, "Sent confirmation nudges", "sent", sent, "failed", failed)
			return nil
		},
	}
}
