// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v3"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/mcpserver"
)

// mcpCmd serves the read-only MCP server over stdio, for local MCP clients
// such as Claude Code or Claude Desktop pointed at a development database.
// stdout carries the protocol, so all logging must go to stderr (slog's
// default handler already does).
func mcpCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "Run the read-only GoDaily MCP server over stdio.",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return mcpserver.New(mcpserver.DepsFromApp(a)).Run(ctx, &mcp.StdioTransport{})
		},
	}
}
