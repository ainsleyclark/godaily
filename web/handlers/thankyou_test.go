// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

func TestThankYou(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		query    string
		wantHTML string
	}{
		"Without email": {
			query:    "",
			wantHTML: "Check your email",
		},
		"With email": {
			query:    "?email=hello%40example.com",
			wantHTML: "hello@example.com",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kit := webkit.New()
			kit.Get("/thank-you/", ThankYou())

			req := httptest.NewRequest(http.MethodGet, "/thank-you/"+test.query, nil)
			rec := httptest.NewRecorder()
			kit.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Contains(t, rec.Body.String(), test.wantHTML)
		})
	}
}
