// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mcpserver

import (
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HTTPHandler serves s over the MCP Streamable HTTP transport.
//
// The handler is stateless and answers with plain JSON rather than an SSE
// stream. That suits Vercel's serverless runtime, where consecutive requests
// from one client may land on different instances and long-lived streams are
// cut off: every POST carries a complete JSON-RPC exchange, no session state
// is kept between requests, and GET/DELETE return 405. GoDaily's tools never
// call back into the client, so nothing is lost by running stateless.
//
// Authentication is not handled here; callers wrap the handler (see
// plugs.MCPAuth in pkg/api/mux).
func HTTPHandler(s *mcp.Server) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return s
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
		Logger:       slog.Default(),
	})
}
