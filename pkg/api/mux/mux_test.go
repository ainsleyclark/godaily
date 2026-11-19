// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mux

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/env"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
)

const (
	testMCPKey    = "mcp-test-key"
	testAPISecret = "admin-test-secret"
)

// newMCPServer serves the full API mux the way production does: mounted
// under /api with the prefix stripped, so requests run through CORS, rate
// limiting, logging, trailing-slash handling and MCP auth.
func newMCPServer(t *testing.T) (*httptest.Server, *mockdigest.MockIssueRepository) {
	t.Helper()

	issues := mockdigest.NewMockIssueRepository(gomock.NewController(t))
	app := &godaily.App{
		Config:     &env.Config{APISecret: testAPISecret, MCPAPIKey: testMCPKey},
		Repository: &godaily.Repository{Issues: issues},
		Service:    &godaily.Service{},
	}
	srv := httptest.NewServer(http.StripPrefix("/api", Handler(app)))
	t.Cleanup(srv.Close)
	return srv, issues
}

// headerTransport adds fixed headers to every request.
type headerTransport struct {
	headers map[string]string
}

func (h headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	for k, v := range h.headers {
		r.Header.Set(k, v)
	}
	return http.DefaultTransport.RoundTrip(r)
}

func connect(t *testing.T, endpoint string, headers map[string]string) (*mcp.ClientSession, error) {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "mux-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             endpoint,
		HTTPClient:           &http.Client{Transport: headerTransport{headers: headers}},
		MaxRetries:           -1,
		DisableStandaloneSSE: true,
	}, nil)
	if err == nil {
		t.Cleanup(func() { _ = session.Close() })
	}
	return session, err
}

func TestMCPEndpoint(t *testing.T) {
	t.Parallel()

	issue := digest.Issue{ID: 42, Slug: "2026-11-18", Subject: "Hello gophers", Status: digest.IssueStatusSent}

	tt := map[string]struct {
		path    string
		headers map[string]string
	}{
		"MCP key as bearer, Vercel trailing slash": {
			path:    "/api/mcp/",
			headers: map[string]string{"Authorization": "Bearer " + testMCPKey},
		},
		"MCP key as X-API-Key": {
			path:    "/api/mcp",
			headers: map[string]string{"X-API-Key": testMCPKey},
		},
		"MCP key in query string": {
			path: "/api/mcp/?key=" + testMCPKey,
		},
		"API_SECRET still works": {
			path:    "/api/mcp/",
			headers: map[string]string{"Authorization": "Bearer " + testAPISecret},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			srv, issues := newMCPServer(t)
			issues.EXPECT().FindBySlug(gomock.Any(), "2026-11-18").Return(issue, nil)

			session, err := connect(t, srv.URL+test.path, test.headers)
			require.NoError(t, err)
			assert.Equal(t, "godaily", session.InitializeResult().ServerInfo.Name)

			tools, err := session.ListTools(context.Background(), nil)
			require.NoError(t, err)
			assert.Len(t, tools.Tools, 16)

			res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      "get_issue",
				Arguments: map[string]any{"key": "2026-11-18"},
			})
			require.NoError(t, err)
			require.False(t, res.IsError)
			raw, err := json.Marshal(res.StructuredContent)
			require.NoError(t, err)
			assert.Contains(t, string(raw), "Hello gophers")
		})
	}
}

func TestMCPEndpoint_Rejections(t *testing.T) {
	t.Parallel()

	t.Run("Unauthenticated clients cannot connect", func(t *testing.T) {
		t.Parallel()
		srv, _ := newMCPServer(t)

		_, err := connect(t, srv.URL+"/api/mcp/", nil)
		require.Error(t, err)
	})

	t.Run("Wrong key", func(t *testing.T) {
		t.Parallel()
		srv, _ := newMCPServer(t)

		_, err := connect(t, srv.URL+"/api/mcp/", map[string]string{"Authorization": "Bearer wrong"})
		require.Error(t, err)
	})

	t.Run("Raw 401 response", func(t *testing.T) {
		t.Parallel()
		srv, _ := newMCPServer(t)

		resp := post(t, srv.URL+"/api/mcp/", nil, initializeBody)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, `Bearer realm="godaily"`, resp.Header.Get("WWW-Authenticate"))
	})

	t.Run("Admin secret is refused in the query string", func(t *testing.T) {
		t.Parallel()
		srv, _ := newMCPServer(t)

		resp := post(t, srv.URL+"/api/mcp/?key="+testAPISecret, nil, initializeBody)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("GET is not allowed on the stateless transport", func(t *testing.T) {
		t.Parallel()
		srv, _ := newMCPServer(t)

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/mcp/", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+testMCPKey)
		req.Header.Set("Accept", "text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("CORS preflight succeeds without credentials", func(t *testing.T) {
		t.Parallel()
		srv, _ := newMCPServer(t)

		req, err := http.NewRequest(http.MethodOptions, srv.URL+"/api/mcp/", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Mcp-Protocol-Version")
	})
}

// TestMCPEndpoint_NotRateLimited guards against MCP clients being throttled
// by the public per-IP limiter (burst 50). Hosted clients such as claude.ai
// share egress IPs, and some can only send the key as X-API-Key or ?key=.
func TestMCPEndpoint_NotRateLimited(t *testing.T) {
	t.Parallel()

	srv, _ := newMCPServer(t)
	const listTools = `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

	for i := range 80 {
		resp := post(t, srv.URL+"/api/mcp/?key="+testMCPKey, nil, listTools)
		require.Equal(t, http.StatusOK, resp.StatusCode, "query key request %d", i)
		resp = post(t, srv.URL+"/api/mcp/", map[string]string{"X-API-Key": testMCPKey}, listTools)
		require.Equal(t, http.StatusOK, resp.StatusCode, "X-API-Key request %d", i)
	}
}

const initializeBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"curl","version":"1"}}}`

func post(t *testing.T, url string, headers map[string]string, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// TestMCPEndpoint_QueryKeyIsNotLogged guards against the ?key= credential
// reaching the request logger. It swaps the default slog logger, so it does
// not run in parallel with the other tests in this package.
func TestMCPEndpoint_QueryKeyIsNotLogged(t *testing.T) { //nolint:paralleltest // mutates slog.Default
	var (
		mu  sync.Mutex
		buf bytes.Buffer
	)
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		return buf.Write(p)
	}), &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	srv, _ := newMCPServer(t)
	resp := post(t, srv.URL+"/api/mcp/?key="+testMCPKey, nil, initializeBody)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	assert.Contains(t, string(body), `"serverInfo"`)

	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, buf.String(), "/api/mcp/", "the request should still be logged")
	assert.NotContains(t, buf.String(), testMCPKey, "the key must not appear in logs")
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }
