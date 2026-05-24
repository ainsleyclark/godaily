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
	apimetrics "github.com/ainsleyclark/godaily/api/metrics"
	metricsissueslug "github.com/ainsleyclark/godaily/api/metrics/issues"
	apisocial "github.com/ainsleyclark/godaily/api/social"
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
	mux.HandleFunc("GET /social/featured", apisocial.HandleFeatured)
	mux.HandleFunc("GET /social/rotation", apisocial.HandleRotation)
	mux.HandleFunc("GET /social/metrics", apisocial.Handler)
	mux.HandleFunc("GET /healthz", apihandlers.HandleHealthz)
	mux.HandleFunc("GET /metrics/summary", apimetrics.HandleSummary)
	mux.HandleFunc("GET /metrics/issues", apimetrics.HandleIssues)
	mux.HandleFunc("GET /metrics/issues/slug", metricsissueslug.Handler)
	mux.HandleFunc("GET /metrics/items", apimetrics.HandleItems)
	mux.HandleFunc("GET /metrics/tags", apimetrics.HandleTags)
	mux.HandleFunc("GET /metrics/sources", apimetrics.HandleSources)
	mux.HandleFunc("GET /metrics/trend", apimetrics.HandleTrend)
	mux.HandleFunc("GET /metrics/subscribers", apimetrics.HandleSubscribers)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(pkgapi.WithApp(r.Context(), app)))
	})
}
