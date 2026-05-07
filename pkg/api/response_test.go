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

package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		status  int
		v       any
		wantBody string
	}{
		"Object": {
			status:   http.StatusOK,
			v:        map[string]string{"key": "value"},
			wantBody: `{"key":"value"}`,
		},
		"Boolean": {
			status:   http.StatusOK,
			v:        map[string]bool{"ok": true},
			wantBody: `{"ok":true}`,
		},
		"Created": {
			status:   http.StatusCreated,
			v:        map[string]string{"id": "123"},
			wantBody: `{"id":"123"}`,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			JSON(w, test.status, test.v)
			assert.Equal(t, test.status, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.Equal(t, test.wantBody, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestError(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		status  int
		message string
	}{
		"Bad Request": {
			status:  http.StatusBadRequest,
			message: "email is required",
		},
		"Internal Server Error": {
			status:  http.StatusInternalServerError,
			message: "something went wrong",
		},
		"Conflict": {
			status:  http.StatusConflict,
			message: "already subscribed",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			Error(w, test.status, test.message)
			assert.Equal(t, test.status, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.JSONEq(t, `{"error":"`+test.message+`"}`, w.Body.String())
		})
	}
}

func TestOK(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	OK(w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"ok":true}`, w.Body.String())
}
