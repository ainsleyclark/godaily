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

// Package mux wires all API handler functions into a single http.Handler.
// It lives outside api/ so Vercel's serverless function glob ("api/**/*.go")
// does not pick it up as a function entry point.
package mux

import (
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	pkgapi "github.com/ainsleyclark/godaily/pkg/api"
	handlers "github.com/ainsleyclark/godaily/pkg/api/handlers"
	digesthandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/digest"
	issuehandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/issues"
	itemhandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/items"
	metricshandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/metrics"
	metricsissues "github.com/ainsleyclark/godaily/pkg/api/handlers/metrics/issues"
	socialhandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/social"
	webhookhandlers "github.com/ainsleyclark/godaily/pkg/api/handlers/webhooks"
)

// Handler returns an http.Handler for all API routes with app injected into
// every request context. Routes are relative to /api — callers should mount
// or strip the /api prefix before dispatching here.
func Handler(app *godaily.App) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handlers.HandleHealthz)
	mux.HandleFunc("POST /subscribe", digesthandlers.HandleSubscribe)
	mux.HandleFunc("GET /confirm", digesthandlers.HandleConfirm)
	// Accept both GET (link click) and POST (RFC 8058 one-click).
	mux.HandleFunc("/unsubscribe", digesthandlers.HandleUnsubscribe)
	mux.HandleFunc("GET /digest/collect", digesthandlers.HandleCollect)
	mux.HandleFunc("GET /digest/build", digesthandlers.HandleBuild)
	mux.HandleFunc("GET /digest/send", digesthandlers.HandleSend)
	mux.HandleFunc("GET /digest/preview", digesthandlers.HandlePreview)
	mux.HandleFunc("GET /digest/issues", digesthandlers.HandleIssues)
	mux.HandleFunc("GET /digest/subscribers", digesthandlers.HandleSubscribers)
	mux.HandleFunc("GET /issues/{slug}", issuehandlers.HandleBySlug)
	mux.HandleFunc("GET /items/{id}", itemhandlers.HandleByID)
	mux.HandleFunc("GET /social/featured", socialhandlers.HandleFeatured)
	mux.HandleFunc("GET /social/rotation", socialhandlers.HandleRotation)
	mux.HandleFunc("GET /social/metrics", socialhandlers.HandleMetrics)
	mux.HandleFunc("GET /metrics/summary", metricshandlers.HandleSummary)
	mux.HandleFunc("GET /metrics/issues", metricshandlers.HandleIssues)
	mux.HandleFunc("GET /metrics/issues/{slug}", metricsissues.HandleBySlug)
	mux.HandleFunc("GET /metrics/items", metricshandlers.HandleItems)
	mux.HandleFunc("GET /metrics/tags", metricshandlers.HandleTags)
	mux.HandleFunc("GET /metrics/sources", metricshandlers.HandleSources)
	mux.HandleFunc("GET /metrics/trend", metricshandlers.HandleTrend)
	mux.HandleFunc("GET /metrics/subscribers", metricshandlers.HandleSubscribers)
	mux.HandleFunc("GET /metrics/roundup", metricshandlers.HandleRoundup)
	mux.HandleFunc("POST /webhooks/resend", webhookhandlers.HandleResend)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(pkgapi.WithApp(r.Context(), app)))
	})
}
