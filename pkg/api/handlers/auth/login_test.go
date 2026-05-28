// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
)

func invoke(h func(*webkit.Context) error, w *httptest.ResponseRecorder, r *http.Request) {
	c := webkit.NewContext(w, r)
	if err := h(c); err != nil {
		var e *webkit.Error
		if errors.As(err, &e) {
			_ = c.JSON(e.Code, map[string]string{"error": e.Message})
		} else {
			_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
}

func TestLogin(t *testing.T) {
	tt := map[string]struct {
		password   string
		apiSecret  string
		body       string
		wantStatus int
		wantBody   string
	}{
		"Correct Password": {
			password:   "hunter2",
			apiSecret:  "secret-token",
			body:       `{"password":"hunter2"}`,
			wantStatus: http.StatusOK,
			wantBody:   "secret-token",
		},
		"Wrong Password": {
			password:   "hunter2",
			apiSecret:  "secret-token",
			body:       `{"password":"nope"}`,
			wantStatus: http.StatusUnauthorized,
		},
		"Empty Configured Password Passthrough": {
			password:   "",
			apiSecret:  "secret-token",
			body:       `{"password":"anything"}`,
			wantStatus: http.StatusOK,
			wantBody:   "secret-token",
		},
		"Invalid Body": {
			password:   "hunter2",
			body:       `not-json`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			h := &Handler{password: test.password, apiSecret: test.apiSecret}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(test.body))
			invoke(h.Login, w, r)

			assert.Equal(t, test.wantStatus, w.Code)
			if test.wantBody != "" {
				assert.Contains(t, w.Body.String(), test.wantBody)
			}
		})
	}
}
