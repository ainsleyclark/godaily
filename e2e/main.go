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

// e2e/main.go starts a combined web+API server on :4000 for Playwright tests.
// Run with: go run ./e2e
package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/apimux"
	"github.com/ainsleyclark/godaily/pkg/db"
	"github.com/ainsleyclark/godaily/pkg/digest"
	"github.com/ainsleyclark/godaily/pkg/email"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/items"
	"github.com/ainsleyclark/godaily/pkg/store/subscribers"
	"github.com/ainsleyclark/godaily/pkg/subscriber"
	webserver "github.com/ainsleyclark/godaily/web/server"
	"github.com/ainsleydev/webkit/pkg/cache"
)

// spyEmail captures every email.Send call; used so Playwright tests can assert
// against sent emails without making real external API calls.
type spyEmail struct {
	mu   sync.Mutex
	sent []email.SendEmailRequest
}

func (s *spyEmail) Send(_ context.Context, req email.SendEmailRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, req)
	return nil
}

// stubRunner satisfies digest.Runner without performing any work.
type stubRunner struct{}

func (stubRunner) Collect(_ context.Context, _ digest.CollectOptions) ([]news.SourceItems, error) {
	return nil, nil
}
func (stubRunner) SendDigest(_ context.Context, _ time.Time, _ bool) error { return nil }
func (stubRunner) SendSuggestion(_ context.Context, _ time.Time) error     { return nil }

func main() {
	// Resolve repo root so relative paths (web/dist/, migrations/) are correct.
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(file), "..")
	if err := os.Chdir(repoRoot); err != nil {
		log.Fatalf("chdir to repo root: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "godaily-e2e-*")
	if err != nil {
		log.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	conn, err := db.New(ctx, "file:"+filepath.Join(tmpDir, "godaily.db"), "")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Up(ctx, conn); err != nil {
		log.Fatalf("migrate db: %v", err)
	}

	issueStore := issues.New(conn)
	subsStore := subscribers.New(conn)
	store := cache.NewInMemory(24 * time.Hour)
	cached := issues.NewCaching(issueStore, store)

	spy := &spyEmail{}

	app := &godaily.App{
		Config:      &env.Config{APISecret: "e2e-test-secret"},
		DB:          conn,
		Repository:  &godaily.Repository{Issues: cached, Items: items.New(conn), Subscribers: subsStore},
		Runner:      stubRunner{},
		Cache:       store,
		Subscribers: subscriber.New(subsStore, cached, spy),
	}

	webH := webserver.Handler(app)

	apiH := http.StripPrefix("/api", apimux.Handler(app))

	combined := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/e2e/emails":
			// Debug endpoint — exposes captured emails so Playwright tests can read
			// the unsubscribe/confirm token without needing a real email provider.
			spy.mu.Lock()
			defer spy.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(spy.sent)
		case strings.HasPrefix(r.URL.Path, "/api/"):
			apiH.ServeHTTP(w, r)
		default:
			webH.ServeHTTP(w, r)
		}
	})

	ln, err := net.Listen("tcp", ":4000") // #nosec G102 -- e2e server must accept connections from Playwright
	if err != nil {
		log.Fatalf("listen :4000: %v", err)
	}

	srv := &http.Server{Handler: combined, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("server: %v", err)
		}
	}()

	log.Println("E2E server listening on :4000")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	_ = srv.Shutdown(context.Background())
}
