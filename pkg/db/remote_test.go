// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// connKey carries a per-connection request counter through ConnContext.
type connKey struct{}

// staleTurso is a fake Turso /v2/pipeline endpoint. When dropReused is set, it
// drops the next request that arrives on an already-used keep-alive connection
// without replying, so the client sees `Post ".../v2/pipeline": EOF`. That is
// what happens in production when a warm serverless instance reuses a socket
// that Turso closed while the instance was frozen.
type staleTurso struct {
	*httptest.Server
	dropReused atomic.Bool
	dropped    atomic.Int32
	requests   atomic.Int32
}

func newStaleTurso(t *testing.T) *staleTurso {
	t.Helper()

	st := &staleTurso{}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(st.serve))
	srv.Config.ConnContext = func(ctx context.Context, _ net.Conn) context.Context {
		return context.WithValue(ctx, connKey{}, new(atomic.Int32))
	}
	srv.Start()
	t.Cleanup(srv.Close)
	st.Server = srv

	return st
}

func (st *staleTurso) serve(w http.ResponseWriter, r *http.Request) {
	st.requests.Add(1)

	served := r.Context().Value(connKey{}).(*atomic.Int32)
	if served.Add(1) > 1 && st.dropReused.CompareAndSwap(true, false) {
		st.dropped.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
		return
	}

	var req struct {
		Requests []struct {
			Type string `json:"type"`
		} `json:"requests"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results := make([]map[string]any, 0, len(req.Requests))
	for _, sr := range req.Requests {
		resp := map[string]any{"type": sr.Type}
		if sr.Type == "execute" {
			resp["result"] = map[string]any{
				"cols":               []map[string]any{{"name": "1"}},
				"rows":               [][]map[string]any{{{"type": "integer", "value": "1"}}},
				"affected_row_count": 0,
			}
		}
		results = append(results, map[string]any{"type": "ok", "response": resp})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
}

// newRemote opens a remote DB against st and runs one query, which leaves a
// keep-alive socket in the HTTP pool, as a warm serverless instance has after
// serving a request. The idle timeout is shortened so tests don't wait.
func newRemote(t *testing.T, st *staleTurso) *sql.DB {
	t.Helper()

	old := idleTimeout
	idleTimeout = 50 * time.Millisecond
	t.Cleanup(func() { idleTimeout = old })

	conn, err := New(t.Context(), st.URL, "secret")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	_, err = conn.ExecContext(t.Context(), "SELECT 1")
	require.NoError(t, err)

	return conn
}

func TestNew_Remote(t *testing.T) {
	// Guards the harness: reusing a dead pooled socket must fail the way
	// production did, or the next test proves nothing.
	t.Run("Reused Dead Socket Fails", func(t *testing.T) {
		st := newStaleTurso(t)
		conn := newRemote(t, st)

		st.dropReused.Store(true)

		_, err := conn.ExecContext(t.Context(), "SELECT 1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "/v2/pipeline\": EOF")
	})

	t.Run("Idle Socket Is Not Reused", func(t *testing.T) {
		st := newStaleTurso(t)
		conn := newRemote(t, st)

		// As after a frozen invocation: the pooled socket is dead and has
		// been idle for longer than the idle timeout.
		st.dropReused.Store(true)
		time.Sleep(100 * time.Millisecond)

		_, err := conn.ExecContext(t.Context(), "SELECT 1")
		require.NoError(t, err)
		assert.Equal(t, int32(0), st.dropped.Load(), "the query should have used a fresh connection")
		assert.Equal(t, int32(2), st.requests.Load(), "the query should run exactly once")
	})
}
