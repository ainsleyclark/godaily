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
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// e2e/main.go starts a combined web+API server on :4000 for Playwright tests.
// Run with: go run ./e2e
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

	apihandlers "github.com/ainsleyclark/godaily/api"
	webhookhandler "github.com/ainsleyclark/godaily/api/webhooks"
	godaily "github.com/ainsleyclark/godaily/pkg"
	pkgapi "github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/api/mux"
	"github.com/ainsleyclark/godaily/pkg/db"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/services/digest"
	"github.com/ainsleyclark/godaily/pkg/services/emailevent"
	"github.com/ainsleyclark/godaily/pkg/services/subscriber"
	_ "github.com/ainsleyclark/godaily/pkg/source" // registers all source fetchers via init()
	"github.com/ainsleyclark/godaily/pkg/store/emailevents"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/items"
	"github.com/ainsleyclark/godaily/pkg/store/subscribers"
	webserver "github.com/ainsleyclark/godaily/web/server"
	"github.com/ainsleydev/webkit/pkg/cache"
)

// e2eWebhookSecret is the test Resend webhook secret used for all webhook E2E
// tests. It is base64(test-webhook-secret-key) with the whsec_ prefix.
const e2eWebhookSecret = "whsec_dGVzdC13ZWJob29rLXNlY3JldC1rZXk=" // #nosec G101 -- intentional test credential

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

// noopSlack satisfies slack.Sender without making any API calls.
type noopSlack struct{}

func (noopSlack) Send(_ context.Context, _ string) error { return nil }
func (noopSlack) MustSend(_ context.Context, _ string)   {}

// seedRunner satisfies digest.Runner for E2E tests. Collect inserts fixed fake
// items directly into the DB (bypassing real HTTP sources). Build and
// SendDigest delegate to the real digest.Aggregator so the full pipeline logic
// is exercised with the spy email sender.
type seedRunner struct {
	items      news.ItemRepository
	aggregator *digest.Aggregator
}

func (r seedRunner) Collect(ctx context.Context, _ digest.CollectOptions) ([]news.SourceItems, error) {
	// Items are published yesterday so they fall within buildWindow's [yesterday, today) range.
	yesterday := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -1)
	seeds := []news.Item{
		{Source: news.SourceHN, Title: "E2E: Understanding Go Channels", URL: "https://e2e.test/go-channels", Tag: news.TagArticle, Score: 0.8, Published: yesterday},
		{Source: news.SourceDevTo, Title: "E2E: Go Interfaces Deep Dive", URL: "https://e2e.test/go-interfaces", Tag: news.TagArticle, Score: 0.6, Published: yesterday},
	}
	for i, item := range seeds {
		if _, err := r.items.Create(ctx, nil, i+1, item); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (r seedRunner) Build(ctx context.Context, date time.Time) error {
	return r.aggregator.Build(ctx, date)
}

func (r seedRunner) SendDigest(ctx context.Context, date time.Time, force bool) error {
	return r.aggregator.SendDigest(ctx, date, force)
}

func (r seedRunner) SendSuggestion(_ context.Context, _ time.Time) error { return nil }

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
	itemStore := items.New(conn)
	subsStore := subscribers.New(conn)
	eventsStore := emailevents.New(conn)
	store := cache.NewInMemory(24 * time.Hour)
	cached := issues.NewCaching(issueStore, store)

	spy := &spyEmail{}

	// The aggregator uses the non-cached issueStore so SendDigest can find a
	// freshly-built draft without a cache miss. nil prompter → static subject.
	aggregator, err := digest.New(spy, "admin@e2e.test", nil, noopSlack{}, issueStore, itemStore, subsStore)
	if err != nil {
		log.Fatalf("create aggregator: %v", err)
	}

	subscriberSvc := subscriber.New(subsStore, cached, spy)

	app := &godaily.App{
		Config: &env.Config{
			APISecret:           "e2e-test-secret",
			ResendWebhookSecret: e2eWebhookSecret,
		},
		DB:          conn,
		Repository:  &godaily.Repository{Issues: cached, Items: itemStore, Subscribers: subsStore, EmailEvents: eventsStore},
		Runner:      seedRunner{items: itemStore, aggregator: aggregator},
		Cache:       store,
		Subscribers: subscriberSvc,
		EmailEvents: emailevent.New(eventsStore, subscriberSvc),
		Slack:       noopSlack{},
	}

	webH := webserver.Handler(app)
	apiH := http.StripPrefix("/api", mux.Handler(app))

	// withApp injects app into the request context for Vercel function handlers
	// that are not registered in the mux (e.g. HandleBuild).
	withApp := func(r *http.Request) *http.Request {
		return r.WithContext(pkgapi.WithApp(r.Context(), app))
	}

	combined := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		// ── E2E debug: email spy ──────────────────────────────────────────────
		case "/api/e2e/emails":
			spy.mu.Lock()
			defer spy.mu.Unlock()
			writeJSON(w, spy.sent)

		// ── E2E pipeline: bypass weekend guard, call runner directly ──────────
		case "/api/e2e/pipeline/collect":
			if _, err := app.Runner.Collect(r.Context(), digest.CollectOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "/api/e2e/pipeline/build":
			if err := app.Runner.Build(r.Context(), time.Now().UTC()); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "/api/e2e/pipeline/send":
			// force=true bypasses the draft-status guard so tests aren't order-sensitive.
			if err := app.Runner.SendDigest(r.Context(), time.Now().UTC(), true); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)

		// ── E2E debug: raw DB subscriber lookup ──────────────────────────────
		case "/api/e2e/db/subscriber":
			email := r.URL.Query().Get("email")
			if email == "" {
				http.Error(w, "email query param required", http.StatusBadRequest)
				return
			}
			row := conn.QueryRowContext(r.Context(),
				"SELECT id, email, COALESCE(confirm_token,''), confirmed_at, unsubscribed_at, bounced_at FROM subscribers WHERE email = ? LIMIT 1",
				email)
			var sub struct {
				ID             int64   `json:"id"`
				Email          string  `json:"email"`
				ConfirmToken   string  `json:"confirm_token"`
				ConfirmedAt    *string `json:"confirmed_at"`
				UnsubscribedAt *string `json:"unsubscribed_at"`
				BouncedAt      *string `json:"bounced_at"`
			}
			if err := row.Scan(&sub.ID, &sub.Email, &sub.ConfirmToken, &sub.ConfirmedAt, &sub.UnsubscribedAt, &sub.BouncedAt); err != nil {
				http.Error(w, "subscriber not found: "+err.Error(), http.StatusNotFound)
				return
			}
			writeJSON(w, sub)

		// ── E2E webhook signing helper ────────────────────────────────────────
		case "/api/e2e/sign":
			handleSign(w, r)

		// ── Routes that are Vercel functions (not in mux.go) ─────────────────
		case "/api/build":
			apihandlers.HandleBuild(w, withApp(r))

		default:
			switch {
			case strings.HasPrefix(r.URL.Path, "/api/webhooks/"):
				webhookhandler.Handler(w, withApp(r))
			case strings.HasPrefix(r.URL.Path, "/api/"):
				apiH.ServeHTTP(w, r)
			default:
				webH.ServeHTTP(w, r)
			}
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

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// handleSign generates a valid Svix-style webhook signature so Playwright tests
// can POST signed payloads to /api/webhooks/resend without implementing HMAC in TS.
func handleSign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Body      string `json:"body"`
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(e2eWebhookSecret, "whsec_"))
	if err != nil {
		http.Error(w, "invalid secret: "+err.Error(), http.StatusInternalServerError)
		return
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(req.ID + "." + req.Timestamp + "." + req.Body))
	sig := "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	writeJSON(w, map[string]string{
		"svix-id":        req.ID,
		"svix-timestamp": req.Timestamp,
		"svix-signature": sig,
	})
}
