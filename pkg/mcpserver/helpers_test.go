// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
)

// harness wires a real MCP server to an in-memory client so tests exercise
// the full protocol path: schema validation, argument decoding, output
// validation and result encoding.
type harness struct {
	Issues        *mockdigest.MockIssueRepository
	Items         *mocknews.MockItemRepository
	Metrics       *mockengagement.MockMetricsRepository
	EmailEvents   *mockengagement.MockEmailEventRepository
	SocialMetrics *mockengagement.MockSocialMetricRepository
	Session       *mcp.ClientSession
}

func setup(t *testing.T) *harness {
	t.Helper()

	ctrl := gomock.NewController(t)
	h := &harness{
		Issues:        mockdigest.NewMockIssueRepository(ctrl),
		Items:         mocknews.NewMockItemRepository(ctrl),
		Metrics:       mockengagement.NewMockMetricsRepository(ctrl),
		EmailEvents:   mockengagement.NewMockEmailEventRepository(ctrl),
		SocialMetrics: mockengagement.NewMockSocialMetricRepository(ctrl),
	}

	server := New(Deps{
		Issues:        h.Issues,
		Items:         h.Items,
		Metrics:       h.Metrics,
		EmailEvents:   h.EmailEvents,
		SocialMetrics: h.SocialMetrics,
	})

	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverT, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	h.Session, err = client.Connect(ctx, clientT, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = h.Session.Close()
		_ = serverSession.Wait()
	})
	return h
}

// call invokes a tool and fails the test on protocol-level errors. Tool
// errors are returned in the result (IsError) for the caller to assert on.
func (h *harness) call(t *testing.T, name string, args any) *mcp.CallToolResult {
	t.Helper()
	res, err := h.Session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	require.NoError(t, err)
	return res
}

// decode calls a tool, requires success and decodes its structured output.
func decode[T any](t *testing.T, h *harness, name string, args any) T {
	t.Helper()
	res := h.call(t, name, args)
	require.False(t, res.IsError, "tool returned an error: %s", resultText(res))

	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var out T
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// toolError calls a tool, requires it to fail and returns the error text.
func toolError(t *testing.T, h *harness, name string, args any) string {
	t.Helper()
	res := h.call(t, name, args)
	require.True(t, res.IsError, "expected a tool error, got: %s", resultText(res))
	return resultText(res)
}

func resultText(res *mcp.CallToolResult) string {
	var s string
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			s += tc.Text
		}
	}
	return s
}
