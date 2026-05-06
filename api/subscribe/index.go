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

// Package handler is the Vercel serverless function for POST /api/subscribe.
package handler

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/ainsleyclark/godaily/internal/db"
	"github.com/ainsleyclark/godaily/internal/store/issues"
)

// Handler is the Vercel serverless function entry point.
//
// Temporary: lists the 5 most recent issues from Turso to verify DB
// connectivity works from a Vercel Function before full implementation.
func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	conn, err := db.New(ctx, os.Getenv("TURSO_URL"), os.Getenv("TURSO_AUTH_TOKEN"))
	if err != nil {
		http.Error(w, "failed to connect to database: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	store := issues.New(conn)
	latest, err := store.Latest(ctx, 5)
	if err != nil {
		http.Error(w, "failed to query issues: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"issues": len(latest),
	})
}
