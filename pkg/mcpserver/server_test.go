// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mcpserver

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var wantTools = []string{
	"engagement_trend",
	"get_issue",
	"get_issue_candidates",
	"get_issue_engagement",
	"get_item",
	"issue_engagement_trend",
	"list_issue_engagement",
	"list_issues",
	"list_item_facets",
	"metrics_summary",
	"search_items",
	"social_post_metrics",
	"subscriber_growth",
	"top_items",
	"top_sources",
	"top_tags",
}

func TestServer_Initialize(t *testing.T) {
	t.Parallel()

	h := setup(t)
	init := h.Session.InitializeResult()
	require.NotNil(t, init)
	assert.Equal(t, Name, init.ServerInfo.Name)
	assert.Equal(t, Version, init.ServerInfo.Version)
	assert.Contains(t, init.Instructions, "GoDaily")
	require.NotNil(t, init.Capabilities.Tools)
}

func TestServer_ListTools(t *testing.T) {
	t.Parallel()

	h := setup(t)
	res, err := h.Session.ListTools(context.Background(), nil)
	require.NoError(t, err)

	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)

		t.Run(tool.Name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, tool.Description, "description")
			require.NotNil(t, tool.Annotations, "annotations")
			assert.True(t, tool.Annotations.ReadOnlyHint, "every tool must be read-only")
			assert.True(t, tool.Annotations.IdempotentHint)
			require.NotNil(t, tool.Annotations.OpenWorldHint)
			assert.False(t, *tool.Annotations.OpenWorldHint)
			assert.NotEmpty(t, tool.Annotations.Title)
			assert.NotNil(t, tool.InputSchema, "input schema")
			assert.NotNil(t, tool.OutputSchema, "output schema")
		})
	}
	slices.Sort(names)
	assert.Equal(t, wantTools, names)
}

// TestServer_InputSchemas checks the schemas clients see, since they are what
// the model relies on to build valid calls.
func TestServer_InputSchemas(t *testing.T) {
	t.Parallel()

	h := setup(t)
	res, err := h.Session.ListTools(context.Background(), nil)
	require.NoError(t, err)

	schemas := map[string]schemaView{}
	for _, tool := range res.Tools {
		raw, err := json.Marshal(tool.InputSchema)
		require.NoError(t, err)
		var s schemaView
		require.NoError(t, json.Unmarshal(raw, &s))
		schemas[tool.Name] = s
	}

	t.Run("Every input is an object", func(t *testing.T) {
		t.Parallel()
		for name, s := range schemas {
			assert.Equal(t, "object", s.Type, name)
		}
	})

	t.Run("Window fields are flattened and period is an enum", func(t *testing.T) {
		t.Parallel()
		s := schemas["metrics_summary"]
		require.Contains(t, s.Properties, "period")
		require.Contains(t, s.Properties, "from")
		require.Contains(t, s.Properties, "to")
		assert.Equal(t, []any{"day", "week", "month", "year", "all"}, s.Properties["period"].Enum)
		assert.Empty(t, s.Required, "window arguments are optional")
	})

	t.Run("Issue key is required", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{"get_issue", "get_issue_candidates", "get_issue_engagement", "issue_engagement_trend"} {
			assert.Equal(t, []string{"key"}, schemas[name].Required, name)
		}
	})

	t.Run("Enums are advertised", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []any{"draft", "sent", "error"}, schemas["list_issues"].Properties["status"].Enum)
		assert.Equal(t, []any{"new", "top", "hot"}, schemas["search_items"].Properties["sort"].Enum)
		assert.Contains(t, schemas["list_issue_engagement"].Properties["sort"].Enum, "click_rate")
		assert.Equal(t, []any{"hour", "day", "week"}, schemas["engagement_trend"].Properties["bucket"].Enum)
		assert.Equal(t, []any{"day", "week", "month"}, schemas["subscriber_growth"].Properties["bucket"].Enum)
		assert.Contains(t, schemas["issue_engagement_trend"].Properties["metric"].Enum, "unique_clicks")
	})

	t.Run("Properties are described", func(t *testing.T) {
		t.Parallel()
		for name, s := range schemas {
			for prop, p := range s.Properties {
				assert.NotEmpty(t, p.Desc, "%s.%s", name, prop)
			}
		}
	})
}

func TestServer_InvalidArgumentsAreRejected(t *testing.T) {
	t.Parallel()

	h := setup(t)

	tt := map[string]struct {
		tool string
		args map[string]any
	}{
		"Unknown enum value": {tool: "list_issues", args: map[string]any{"status": "published"}},
		"Unknown property":   {tool: "metrics_summary", args: map[string]any{"periodd": "week"}},
		"Wrong type":         {tool: "get_item", args: map[string]any{"id": "abc"}},
		"Missing required":   {tool: "get_issue", args: map[string]any{}},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// No repository expectations: invalid input must never reach a handler.
			res, err := h.Session.CallTool(context.Background(), &mcp.CallToolParams{Name: test.tool, Arguments: test.args})
			if err == nil {
				assert.True(t, res.IsError, "expected rejection, got %s", resultText(res))
			}
		})
	}
}

// schemaView is the subset of JSON Schema the tests inspect.
type schemaView struct {
	Type       any                   `json:"type"`
	Required   []string              `json:"required"`
	Properties map[string]schemaView `json:"properties"`
	Enum       []any                 `json:"enum"`
	Desc       string                `json:"description"`
	Items      *schemaView           `json:"items"`
}
