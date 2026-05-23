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

	apihandlers "github.com/ainsleyclark/godaily/api"
	metricsissueslug "github.com/ainsleyclark/godaily/api/metrics/issues"
	metricsissueslugslug "github.com/ainsleyclark/godaily/api/metrics/issues/slug"
	metricsitems "github.com/ainsleyclark/godaily/api/metrics/items"
	metricssources "github.com/ainsleyclark/godaily/api/metrics/sources"
	metricssubscribers "github.com/ainsleyclark/godaily/api/metrics/subscribers"
	metricssummary "github.com/ainsleyclark/godaily/api/metrics/summary"
	metricstags "github.com/ainsleyclark/godaily/api/metrics/tags"
	metricstrend "github.com/ainsleyclark/godaily/api/metrics/trend"
	godaily "github.com/ainsleyclark/godaily/pkg"
	pkgapi "github.com/ainsleyclark/godaily/pkg/api"
)

// Handler returns an http.Handler for all API routes with app injected into
// every request context. Routes are relative to /api — callers should mount
// or strip the /api prefix before dispatching here.
func Handler(app *godaily.App) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /subscribe", apihandlers.HandleSubscribe)
	mux.HandleFunc("GET /confirm", apihandlers.HandleConfirm)
	// Accept both GET (link click) and POST (RFC 8058 one-click).
	mux.HandleFunc("/unsubscribe", apihandlers.HandleUnsubscribe)
	mux.HandleFunc("GET /collect", apihandlers.HandleCollect)
	mux.HandleFunc("GET /send", apihandlers.HandleSend)
	mux.HandleFunc("GET /issues", apihandlers.HandleIssues)
	mux.HandleFunc("GET /subscribers", apihandlers.HandleSubscribers)
	mux.HandleFunc("GET /social", apihandlers.HandleSocial)
	mux.HandleFunc("GET /healthz", apihandlers.HandleHealthz)
	mux.HandleFunc("GET /metrics/summary", metricssummary.Handler)
	mux.HandleFunc("GET /metrics/issues", metricsissueslug.Handler)
	mux.HandleFunc("GET /metrics/issues/slug", metricsissueslugslug.Handler)
	mux.HandleFunc("GET /metrics/items", metricsitems.Handler)
	mux.HandleFunc("GET /metrics/tags", metricstags.Handler)
	mux.HandleFunc("GET /metrics/sources", metricssources.Handler)
	mux.HandleFunc("GET /metrics/trend", metricstrend.Handler)
	mux.HandleFunc("GET /metrics/subscribers", metricssubscribers.Handler)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(pkgapi.WithApp(r.Context(), app)))
	})
}
