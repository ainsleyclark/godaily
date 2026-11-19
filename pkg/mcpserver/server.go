// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mcpserver exposes GoDaily over the Model Context Protocol (MCP) so
// agents such as Claude can query issues, news items and engagement metrics.
//
// It is a delivery layer, like pkg/api: it holds no domain logic of its own
// and calls the domain repositories in-process. Every tool is read-only.
//
// The same *mcp.Server is served over two transports:
//   - Streamable HTTP at /api/mcp (see HTTPHandler and pkg/api/mux).
//   - stdio via the `godaily mcp` CLI subcommand (see pkg/cmd/mcp.go).
package mcpserver

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
)

const (
	// Name is the MCP implementation name advertised to clients.
	Name = "godaily"

	// Version is the MCP implementation version advertised to clients. Bump
	// it when the tool surface changes in a way clients should notice.
	Version = "1.0.0"
)

// instructions are sent to the client on initialisation and typically end up
// in the model's context, so they double as a usage guide for the tools.
const instructions = `GoDaily (https://godaily.dev) is a daily Go programming newsletter. Each
weekday it collects news items from sources such as Hacker News, Reddit,
Lobsters, the Go blog and GitHub, builds them into a digest "issue" and emails
it to subscribers. All tools are read-only.

Concepts:
- Issue: one newsletter edition. Identified by numeric ID or by its date slug
  (YYYY-MM-DD). Status is draft, sent or error.
- Item: a single collected news link. Items linked to an issue have in_digest=true.
- Engagement: email opens/clicks recorded per issue and per link.

Tips:
- get_issue accepts "latest" (most recent sent issue) and "draft" (most recent
  draft) as well as an ID or slug.
- Metrics tools take a window: either period (day, week, month, year, all) or
  explicit from/to dates (YYYY-MM-DD). Omitting both means all time.
- Use list_item_facets to discover valid source and tag values before
  filtering search_items.
- Rates (open_rate, click_rate) are fractions between 0 and 1.`

// Deps holds the narrow set of repositories the MCP tools read from. It
// mirrors the API handler convention of declaring only what is needed rather
// than depending on the whole App.
type Deps struct {
	Issues        digest.IssueRepository
	Items         news.ItemRepository
	Metrics       engagement.MetricsRepository
	EmailEvents   engagement.EmailEventRepository
	SocialMetrics engagement.SocialMetricRepository
}

// DepsFromApp extracts the MCP dependencies from the application container.
func DepsFromApp(a *godaily.App) Deps {
	return Deps{
		Issues:        a.Repository.Issues,
		Items:         a.Repository.Items,
		Metrics:       a.Repository.Metrics,
		EmailEvents:   a.Repository.EmailEvents,
		SocialMetrics: a.Repository.SocialMetrics,
	}
}

// New builds an MCP server with every GoDaily tool registered. The returned
// server is safe to share across concurrent sessions and requests.
func New(d Deps) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:       Name,
		Title:      "GoDaily",
		Version:    Version,
		WebsiteURL: env.AppURL,
	}, &mcp.ServerOptions{
		Instructions: instructions,
	})

	t := &tools{deps: d}
	t.registerIssues(s)
	t.registerItems(s)
	t.registerMetrics(s)

	return s
}

// tools carries the dependencies shared by every tool handler.
type tools struct {
	deps Deps
}

// readOnly returns the annotations shared by every GoDaily tool: they read
// from GoDaily's own database only, so they are read-only, idempotent and
// closed-world.
func readOnly(title string) *mcp.ToolAnnotations {
	closed := false
	return &mcp.ToolAnnotations{
		Title:          title,
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  &closed,
	}
}

// inputSchema infers the JSON schema for T and restricts the named
// properties to a fixed set of values. Advertising enums lets the model pick
// valid values up front instead of learning them from validation errors. For
// array properties the enum applies to the array's items.
//
// It panics on a programming error (unknown property), which surfaces at
// server construction and therefore in tests.
func inputSchema[T any](enums map[string][]string) *jsonschema.Schema {
	s, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Sprintf("mcpserver: inferring schema for %T: %v", *new(T), err))
	}
	for prop, values := range enums {
		p, ok := s.Properties[prop]
		if !ok {
			panic(fmt.Sprintf("mcpserver: schema for %T has no property %q", *new(T), prop))
		}
		if p.Items != nil {
			p = p.Items
		}
		p.Enum = make([]any, len(values))
		for i, v := range values {
			p.Enum[i] = v
		}
	}
	return s
}
