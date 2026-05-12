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

package e2e_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	apihandlers "github.com/ainsleyclark/godaily/api"
	godaily "github.com/ainsleyclark/godaily/pkg"
	pkgapi "github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/db"
	"github.com/ainsleyclark/godaily/pkg/email"
	"github.com/ainsleyclark/godaily/pkg/env"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/items"
	"github.com/ainsleyclark/godaily/pkg/store/subscribers"
	"github.com/ainsleyclark/godaily/pkg/subscriber"
	"github.com/ainsleydev/webkit/pkg/cache"
)

// spyEmail captures every email.Send call for assertions.
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

// newTestServer returns an isolated httptest.Server backed by a real SQLite
// database, a spyEmail for assertion, and a MockRunner for digest operations.
// App is injected per-request via context — no global state is touched.
func newTestServer(t *testing.T) (*httptest.Server, *spyEmail, *mockdigest.MockRunner) {
	t.Helper()

	dbURL := "file:" + filepath.Join(t.TempDir(), "godaily.db")
	conn, err := db.New(t.Context(), dbURL, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, db.Up(t.Context(), conn))

	issueStore := issues.New(conn)
	subsStore := subscribers.New(conn)
	var store cache.Store = cache.NewInMemory(24 * time.Hour)
	cached := issues.NewCaching(issueStore, store)

	spy := &spyEmail{}
	ctrl := gomock.NewController(t)
	runner := mockdigest.NewMockRunner(ctrl)

	app := &godaily.App{
		Config:      &env.Config{},
		DB:          conn,
		Repository:  &godaily.Repository{Issues: cached, Items: items.New(conn), Subscribers: subsStore},
		Runner:      runner,
		Cache:       store,
		Subscribers: subscriber.New(subsStore, cached, spy),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/subscribe", apihandlers.HandleSubscribe)
	mux.HandleFunc("GET /api/unsubscribe", apihandlers.HandleUnsubscribe)
	mux.HandleFunc("GET /api/collect", apihandlers.HandleCollect)
	mux.HandleFunc("GET /api/send", apihandlers.HandleSend)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(pkgapi.WithApp(r.Context(), app)))
	}))
	t.Cleanup(srv.Close)

	return srv, spy, runner
}

// noRedirectClient returns a client that does not follow redirects, allowing
// 3xx responses to be asserted directly.
func noRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// TestSubscriberLifecycle covers the core subscriber flow end-to-end:
// subscribe → confirmation email → unsubscribe via token → re-subscribe.
func TestSubscriberLifecycle(t *testing.T) {
	srv, spy, _ := newTestServer(t)
	client := noRedirectClient()

	// Subscribe
	res, err := client.Post(
		srv.URL+"/api/subscribe",
		"application/json",
		strings.NewReader(`{"email":"test@example.com"}`),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// Confirm welcome email was sent
	spy.mu.Lock()
	require.Len(t, spy.sent, 1)
	sentReq := spy.sent[0]
	spy.mu.Unlock()
	assert.Equal(t, "Welcome to GoDaily!", sentReq.Subject)
	assert.Contains(t, sentReq.To, "test@example.com")

	// Extract unsubscribe token from the List-Unsubscribe header
	raw := strings.Trim(sentReq.Headers["List-Unsubscribe"], "<>")
	u, err := url.Parse(raw)
	require.NoError(t, err)
	token := u.Query().Get("token")
	require.NotEmpty(t, token)

	// Unsubscribe via token
	res, err = client.Get(srv.URL + "/api/unsubscribe?token=" + token)
	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, res.StatusCode)
	assert.Equal(t, "/unsubscribed/", res.Header.Get("Location"))

	// Re-subscribe (reactivation of an unsubscribed address)
	res, err = client.Post(
		srv.URL+"/api/subscribe",
		"application/json",
		strings.NewReader(`{"email":"test@example.com"}`),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

// TestCollect verifies the collect endpoint delegates to the runner and returns 200.
func TestCollect(t *testing.T) {
	srv, _, runner := newTestServer(t)

	runner.EXPECT().Collect(gomock.Any(), gomock.Any()).Return(nil, nil)

	res, err := http.Get(srv.URL + "/api/collect")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

// TestSend verifies the send endpoint delegates to the runner and returns 200.
func TestSend(t *testing.T) {
	srv, _, runner := newTestServer(t)

	runner.EXPECT().SendDigest(gomock.Any(), gomock.Any(), false).Return(nil)
	runner.EXPECT().SendSuggestion(gomock.Any(), gomock.Any()).Return(nil)

	res, err := http.Get(srv.URL + "/api/send")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}
